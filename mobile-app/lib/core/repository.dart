import 'dart:async';
import 'dart:convert';

import 'package:flutter/foundation.dart';
import 'package:sqflite/sqflite.dart';
import 'package:uuid/uuid.dart';

import 'api.dart';

class SessionRepository extends ChangeNotifier {
  final ApiService api;
  Json? user;
  bool ready = false, offline = false;
  SessionRepository(this.api);
  String get destination {
    if (!ready) return '/splash';
    if (user == null) return '/auth';
    if (user!['is_enabled'] != true) return '/disabled';
    if (user!['role'] == 'customer') return '/buyer/home';
    return '/seller/home';
  }

  Future<void> restore() async {
    try {
      if (await api.refresh()) await reload();
    } catch (_) {
      final snapshot = await api.storage.read(key: 'last_verified_user');
      final verified = DateTime.tryParse(
        await api.storage.read(key: 'last_verified_at') ?? '',
      );
      if (snapshot != null &&
          verified != null &&
          DateTime.now().difference(verified) < const Duration(hours: 24)) {
        user = jsonDecode(snapshot) as Json;
        offline = true;
      }
    }
    ready = true;
    notifyListeners();
  }

  Future<void> verify(Json input) async {
    final tokens = await api.call('auth_verify', body: input) as Json;
    await api.saveTokens(tokens);
    await reload();
  }

  Future<void> reload() async {
    user = await api.call('me') as Json;
    offline = false;
    await api.storage.write(key: 'last_verified_user', value: jsonEncode(user));
    await api.storage.write(
      key: 'last_verified_at',
      value: DateTime.now().toUtc().toIso8601String(),
    );
    ready = true;
    notifyListeners();
  }

  Future<void> logout({bool all = false}) async {
    try {
      await api.call(all ? 'logout_all' : 'logout', body: {});
    } finally {
      await invalidate();
    }
  }

  Future<void> invalidate() async {
    await api.clear();
    user = null;
    ready = true;
    notifyListeners();
  }
}

class AuthViewModel {
  final SessionRepository session;
  String? testingOtp;
  AuthViewModel(this.session);
  Future<String> requestCode(
    String phone,
    String role, {
    String mode = 'login',
  }) async {
    testingOtp = null;
    final result = await session.api.call(
      'auth_request',
      body: {'phone': phone.trim(), 'role': role, 'mode': mode},
    ) as Json;
    final testingCode = result['testing_otp'];
    if (testingCode is String && RegExp(r'^[0-9]{6}$').hasMatch(testingCode)) {
      testingOtp = testingCode;
    }
    return result['challenge_id'] as String;
  }

  Future<void> verify(Json input) => session.verify(input);
}

class LocalStore {
  Database? _database;
  LocalStore({this._database});
  Future<Database> get database async => _database ??= await openDatabase(
    'doodhee.db',
    version: 1,
    onCreate: (db, version) async {
      await db.execute(
        'CREATE TABLE cache (key TEXT PRIMARY KEY, body TEXT NOT NULL)',
      );
      await db.execute(
        'CREATE TABLE pending (id TEXT PRIMARY KEY, user_id TEXT NOT NULL, order_id TEXT NOT NULL, body TEXT NOT NULL, state TEXT NOT NULL DEFAULT \'pending\', error TEXT)',
      );
    },
  );
  Future<void> cache(String key, dynamic value) async => (await database)
      .insert('cache', {
        'key': key,
        'body': jsonEncode(value),
      }, conflictAlgorithm: ConflictAlgorithm.replace)
      .then((_) {});
  Future<dynamic> read(String key) async {
    final rows = await (await database).query(
      'cache',
      where: 'key=?',
      whereArgs: [key],
    );
    return rows.isEmpty ? null : jsonDecode(rows.first['body'] as String);
  }

  Future<void> clear() async {
    final db = await database;
    await db.delete('cache');
    await db.delete('pending');
  }
}

class MarketplaceRepository {
  final ApiService api;
  final LocalStore local;
  final SessionRepository session;
  MarketplaceRepository(this.api, this.local, this.session);
  String cacheKey(String op, String? id, Map<String, String>? q) =>
      '${session.user?['id']}:$op:$id:${jsonEncode(q)}';
  Future<dynamic> read(
    String op, {
    String? id,
    Map<String, String>? query,
  }) async {
    final result = await api.call(op, id: id, query: query);
    await local.cache(cacheKey(op, id, query), result);
    return result;
  }

  Future<dynamic> cached(String op, {String? id, Map<String, String>? query}) =>
      local.read(cacheKey(op, id, query));
  Future<dynamic> mutate(String op, {String? id, Json? body, String? key}) =>
      api.call(op, id: id, body: body, key: key ?? const Uuid().v4());
  Future<void> delivery(String id, Json body) async {
    if ((await pending()).any((operation) => operation['order_id'] == id)) {
      throw ApiFailure(
        409,
        'DELIVERY_PENDING_SYNC',
        'A delivery result is already waiting to sync for this order.',
      );
    }
    final key = const Uuid().v4();
    try {
      await mutate('seller_order_result', id: id, body: body, key: key);
    } on ApiFailure {
      rethrow;
    } catch (_) {
      await (await local.database).insert('pending', {
        'id': key,
        'user_id': session.user!['id'],
        'order_id': id,
        'body': jsonEncode(body),
      });
    }
  }

  Future<List<Json>> pending() async => (await (await local.database).query(
    'pending',
    where: 'user_id=?',
    whereArgs: [session.user?['id']],
  )).map((e) => Map<String, dynamic>.from(e)).toList();
  Future<void> sync() async {
    final db = await local.database;
    for (final item in await pending()) {
      if (item['state'] == 'conflict') continue;
      try {
        await mutate(
          'seller_order_result',
          id: item['order_id'] as String,
          body: jsonDecode(item['body'] as String) as Json,
          key: item['id'] as String,
        );
        await db.delete('pending', where: 'id=?', whereArgs: [item['id']]);
      } on ApiFailure catch (e) {
        if (e.status == 409 || e.status == 404 || e.status == 422) {
          await db.update(
            'pending',
            {'state': 'conflict', 'error': e.message},
            where: 'id=?',
            whereArgs: [item['id']],
          );
        } else {
          rethrow;
        }
      } catch (_) {
        break;
      }
    }
  }
}

class DataViewModel extends ChangeNotifier {
  String? _operation, _id;
  Map<String, String>? _query;
  final MarketplaceRepository repository;
  dynamic data;
  bool loading = false, busy = false, stale = false;
  String? error;
  DataViewModel(this.repository);
  bool _disposed = false;
  @override
  void notifyListeners() {
    if (!_disposed) {
      super.notifyListeners();
    }
  }

  @override
  void dispose() {
    _disposed = true;
    super.dispose();
  }

  Future<void> load(String op, {String? id, Map<String, String>? query}) async {
    _operation = op;
    _id = id;
    _query = query;
    loading = true;
    error = null;
    notifyListeners();
    try {
      data = await repository.read(op, id: id, query: query);
      stale = false;
    } catch (e) {
      data ??= await repository.cached(op, id: id, query: query);
      stale = data != null;
      error = e.toString();
      if (e is ApiFailure &&
          (e.status == 401 || e.code == 'ACCOUNT_DISABLED')) {
        await repository.session.invalidate();
      }
    }
    loading = false;
    notifyListeners();
  }

  Future<void> more() async {
    if (_operation == null || data is! Map || data['next_cursor'] == '') return;
    await action(() async {
      final next = await repository.read(
        _operation!,
        id: _id,
        query: {...?_query, 'cursor': data['next_cursor'] as String},
      );
      data = {
        'items': [...data['items'] as List, ...next['items'] as List],
        'next_cursor': next['next_cursor'],
      };
    });
  }

  Future<bool> action(Future<void> Function() fn) async {
    if (busy) return false;
    busy = true;
    error = null;
    notifyListeners();
    try {
      await fn();
      return true;
    } catch (e) {
      error = e.toString();
      return false;
    } finally {
      busy = false;
      notifyListeners();
    }
  }
}
