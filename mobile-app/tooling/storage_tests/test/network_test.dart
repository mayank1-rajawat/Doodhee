// Run from tooling/storage_tests to keep desktop SQLite out of mobile APKs.
import 'dart:convert';

import 'package:flutter_test/flutter_test.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:sqflite_common_ffi/sqflite_ffi.dart';
import 'package:doodhee/core/api.dart';
import 'package:doodhee/core/repository.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();
  test('Concurrent 401 responses trigger one rotating refresh', () async {
    FlutterSecureStorage.setMockInitialValues({'refresh_token': 'old-refresh'});
    var count = 0;
    final api = ApiService(
      baseUrl: 'http://test',
      client: MockClient((r) async {
        if (r.url.path.endsWith('/token/refresh')) {
          count++;
          await Future<void>.delayed(const Duration(milliseconds: 20));
          return http.Response(
            jsonEncode({
              'data': {
                'access_token': 'new-access',
                'refresh_token': 'new-refresh',
              },
            }),
            200,
          );
        }
        if (r.headers['Authorization'] != 'Bearer new-access') {
          return http.Response(
            '{"error":{"code":"AUTH_INVALID_TOKEN","message":"Expired"}}',
            401,
          );
        }
        return http.Response('{"data":{"id":"buyer"}}', 200);
      }),
    );
    api.accessToken = 'old-access';
    await Future.wait([api.call('me'), api.call('me')]);
    expect(count, 1);
    expect(await api.storage.read(key: 'refresh_token'), 'new-refresh');
  });
  test(
    'Offline delivery queues once, preserves idempotency and flags conflict',
    () async {
      sqfliteFfiInit();
      final db = await databaseFactoryFfi.openDatabase(inMemoryDatabasePath);
      await db.execute(
        'CREATE TABLE cache (key TEXT PRIMARY KEY, body TEXT NOT NULL)',
      );
      await db.execute(
        "CREATE TABLE pending (id TEXT PRIMARY KEY,user_id TEXT,order_id TEXT,body TEXT,state TEXT DEFAULT 'pending',error TEXT)",
      );
      var offline = true, conflict = false;
      String? sentKey;
      final api = ApiService(
        baseUrl: 'http://test',
        client: MockClient((r) async {
          if (offline) throw http.ClientException('offline');
          sentKey = r.headers['Idempotency-Key'];
          if (conflict) {
            return http.Response(
              '{"error":{"code":"ORDER_INVALID_TRANSITION","message":"Already completed"}}',
              409,
            );
          }
          return http.Response('{"data":{"status":"delivered"}}', 200);
        }),
      );
      final session = SessionRepository(api)
        ..user = {
          'id': 'seller',
          'role': 'seller',
          'is_enabled': true,
          'approval_status': 'approved',
        };
      final repo = MarketplaceRepository(
        api,
        LocalStore(database: db),
        session,
      );
      await repo.delivery('order-1', {'result': 'delivered'});
      final first = await repo.pending();
      expect(first.length, 1);
      final id = first.first['id'];
      await expectLater(
        repo.delivery('order-1', {'result': 'delivered'}),
        throwsA(isA<ApiFailure>()),
      );
      offline = false;
      await repo.sync();
      expect(sentKey, id);
      expect(await repo.pending(), isEmpty);
      offline = true;
      await repo.delivery('order-2', {'result': 'delivered'});
      offline = false;
      conflict = true;
      await repo.sync();
      expect((await repo.pending()).first['state'], 'conflict');
      await db.close();
    },
  );
}
