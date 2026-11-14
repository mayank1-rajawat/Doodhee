import 'dart:async';

import 'package:doodhee/core/api.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';

void main() {
  test(
    'OTP connection timeout has a readable message without retrying',
    () async {
      var requests = 0;
      final api = ApiService(
        client: MockClient((_) {
          requests++;
          return Completer<http.Response>().future;
        }),
        requestTimeout: const Duration(milliseconds: 10),
      );
      await expectLater(
        api.call(
          'auth_request',
          body: {'phone': '+919876543210', 'role': 'customer'},
        ),
        throwsA(
          isA<ApiConnectionFailure>().having(
            (e) => e.toString(),
            'message',
            'The server did not respond in time. Check your connection and try again.',
          ),
        ),
      );
      expect(requests, 1);
    },
  );

  test('Timeout also covers a stalled response body', () async {
    final body = StreamController<List<int>>();
    final api = ApiService(
      client: MockClient.streaming(
        (_, _) async => http.StreamedResponse(body.stream, 200),
      ),
      requestTimeout: const Duration(milliseconds: 10),
    );
    await expectLater(
      api.call('auth_request'),
      throwsA(isA<ApiConnectionFailure>()),
    );
    await body.close();
  });

  test(
    'Unreachable server does not discard the current access token',
    () async {
      final api = ApiService(
        client: MockClient(
          (_) async => throw http.ClientException('Connection refused'),
        ),
      )..accessToken = 'existing-access-token';
      await expectLater(
        api.call('me'),
        throwsA(
          isA<ApiConnectionFailure>().having(
            (e) => e.toString(),
            'message',
            'Cannot reach the Doodhee server. Check your connection and try again.',
          ),
        ),
      );
      expect(api.accessToken, 'existing-access-token');
    },
  );
}
