import 'dart:async';
import 'dart:convert';

import 'package:http/http.dart' as http;
import 'package:flutter_secure_storage/flutter_secure_storage.dart';

import '../generated/public_api/operations.dart';

typedef Json = Map<String, dynamic>;

class ApiFailure implements Exception {
  final int status;
  final String code, message;
  ApiFailure(this.status, this.code, this.message);
  @override
  String toString() => message;
}

// Transport failures stay separate from HTTP rejections so offline operations
// can still queue and cached sessions can still be restored.
class ApiConnectionFailure implements Exception {
  final String message;
  const ApiConnectionFailure(this.message);
  @override
  String toString() => message;
}

class ApiService {
  final http.Client client;
  final FlutterSecureStorage storage;
  final String baseUrl;
  final Duration requestTimeout;
  String? accessToken;
  Future<bool>? _refreshing;
  ApiService({
    http.Client? client,
    FlutterSecureStorage? storage,
    String? baseUrl,
    this.requestTimeout = const Duration(seconds: 15),
  }) : client = client ?? http.Client(),
       storage = storage ?? const FlutterSecureStorage(),
       baseUrl =
           baseUrl ??
           const String.fromEnvironment(
             'PUBLIC_API_URL',
             defaultValue: 'http://10.0.2.2:18080',
           );
  Future<dynamic> call(
    String operation, {
    String? id,
    Json? body,
    Map<String, String>? query,
    String? key,
    bool retry = true,
  }) async {
    final op = publicOperations[operation];
    if (op == null) throw ArgumentError('Unknown API operation: $operation');
    final uri = Uri.parse('$baseUrl${op[1].replaceAll('{id}', id ?? '')}')
        .replace(queryParameters: query);
    final request = http.Request(op[0], uri);
    request.headers.addAll({
      'Content-Type': 'application/json',
      if (accessToken != null) 'Authorization': 'Bearer $accessToken',
      'Idempotency-Key': ?key,
    });
    if (op[0] != 'GET') request.body = jsonEncode(body ?? {});
    final response = await _send(request);
    if (response.statusCode == 204) return <String, dynamic>{};
    final payload = jsonDecode(response.body) as Json;
    if (response.statusCode == 401 && retry && !operation.startsWith('auth_')) {
      if (await refresh()) {
        return call(
          operation,
          id: id,
          body: body,
          query: query,
          key: key,
          retry: false,
        );
      }
    }
    if (response.statusCode >= 400) {
      throw ApiFailure(
        response.statusCode,
        payload['error']?['code'] ?? 'UNKNOWN',
        payload['error']?['message'] ?? 'Please try again.',
      );
    }
    return payload['data'];
  }

  Future<http.Response> _send(http.Request request) async {
    try {
      // Include reading the body in the timeout, not just receiving headers.
      return await client
          .send(request)
          .then(http.Response.fromStream)
          .timeout(requestTimeout);
    } on TimeoutException {
      throw const ApiConnectionFailure(
        'The server did not respond in time. Check your connection and try again.',
      );
    } on http.ClientException {
      throw const ApiConnectionFailure(
        'Cannot reach the Doodhee server. Check your connection and try again.',
      );
    }
  }

  Future<void> saveTokens(Json tokens) async {
    accessToken = tokens['access_token'] as String;
    await storage.write(
      key: 'refresh_token',
      value: tokens['refresh_token'] as String,
    );
  }

  Future<bool> refresh() {
    return _refreshing ??= _refresh().whenComplete(() => _refreshing = null);
  }

  Future<bool> _refresh() async {
    final token = await storage.read(key: 'refresh_token');
    if (token == null) return false;
    try {
      final result = await call(
        'auth_refresh',
        body: {'refresh_token': token},
        retry: false,
      );
      await saveTokens(result as Json);
      return true;
    } on ApiFailure {
      await clear();
      return false;
    }
  }

  Future<void> clear() async {
    accessToken = null;
    await storage.delete(key: 'refresh_token');
    await storage.delete(key: 'last_verified_user');
    await storage.delete(key: 'last_verified_at');
  }
}
