import 'dart:convert';

import 'package:authsome_core/authsome_core.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:test/test.dart';

void main() {
  for (final status in WaitlistStatus.values) {
    test('returns ${status.name} with the current app key', () async {
      final requests = <http.Request>[];
      final client = AuthSomeClient(AuthClientConfig(
        baseUrl: 'https://api.example.com/base///',
        publishableKey: 'pk_old',
        httpClient: MockClient((request) async {
          requests.add(request);
          return http.Response(jsonEncode({'status': status.name}), 200);
        }),
      ));
      client.publishableKey = 'pk_current';

      expect(await client.joinWaitlistWithStatus(' Ada@Example.com '), status);
      expect(requests.single.url.path, '/base/v1/waitlist/join');
      expect(requests.single.headers['X-Publishable-Key'], 'pk_current');
      expect(jsonDecode(requests.single.body), {'email': 'ada@example.com'});
    });
  }

  for (final status in [null, 'unknown']) {
    test('rejects an invalid waitlist status: $status', () async {
      final client = AuthSomeClient(AuthClientConfig(
        baseUrl: 'https://api.example.com',
        httpClient: MockClient(
            (_) async => http.Response(jsonEncode({'status': status}), 200)),
      ));
      expect(
        () => client.joinWaitlistWithStatus('ada@example.com'),
        throwsA(isA<AuthClientException>()),
      );
    });
  }

  test('preserves server errors and permits retry', () async {
    var fail = true;
    final client = AuthSomeClient(AuthClientConfig(
      baseUrl: 'https://api.example.com',
      httpClient: MockClient((_) async => fail
          ? http.Response(jsonEncode({'error': 'Try again later'}), 429)
          : http.Response(jsonEncode({'status': 'approved'}), 200)),
    ));
    await expectLater(
      () => client.joinWaitlistWithStatus('ada@example.com'),
      throwsA(isA<AuthClientException>()
          .having((error) => error.code, 'code', 429)
          .having((error) => error.message, 'message', 'Try again later')),
    );
    fail = false;
    expect(await client.joinWaitlistWithStatus('ada@example.com'),
        WaitlistStatus.approved);
  });
}
