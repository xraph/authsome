/// Confirms the publishable key flows from [AuthConfig] all the way
/// down to the `X-Publishable-Key` HTTP header on outbound requests.
///
/// Original bug: two layers silently dropped the key — `AuthManager(...)`
/// built `AuthClientConfig(baseUrl: …)` without forwarding it, and
/// `AuthSomeClient.factory` re-wrapped its config the same way. The
/// generated client therefore never saw the key, every request went out
/// without the header, and TwinOS-style backends rejected sign-in with
/// `{"error": "app context required: send publishable key"}`.
library;

import 'package:authsome_core/authsome_core.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:test/test.dart';

void main() {
  group('Publishable key propagation', () {
    test(
      'AuthSomeClient factory preserves publishableKey from its config',
      () async {
        final captured = <String, String>{};
        final mock = MockClient((req) async {
          captured.addAll(req.headers);
          return http.Response(
            '{"error":"unused"}',
            500,
            headers: {'content-type': 'application/json'},
          );
        });

        final client = AuthSomeClient(AuthClientConfig(
          baseUrl: 'http://test.local',
          httpClient: mock,
          publishableKey: 'pk_test_propagation',
        ));

        // Any request will do — signIn is the relevant one for this case.
        try {
          await client.signInWithCredentials(
            email: 'a@b.com',
            password: 'pw',
          );
        } catch (_) {
          // Expected: the stub returns 500.
        }

        expect(
          captured['X-Publishable-Key'],
          'pk_test_propagation',
          reason:
              'AuthSomeClient.factory must forward publishableKey into the '
              'inner AuthClientConfig so the generated client emits the '
              'X-Publishable-Key header.',
        );
      },
    );

    test(
      'AuthManager forwards publishableKey from AuthConfig to outbound requests',
      () async {
        final captured = <String, String>{};
        final mock = MockClient((req) async {
          captured.addAll(req.headers);
          return http.Response(
            '{"error":"unused"}',
            500,
            headers: {'content-type': 'application/json'},
          );
        });

        // Constructs the manager via the public `AuthManager(config)`
        // path — the same code path the example app and TwinOS use.
        final manager = AuthManager(AuthConfig(
          baseUrl: 'http://test.local',
          publishableKey: 'pk_test_endtoend',
          httpClient: mock,
        ));

        try {
          await manager.signIn('a@b.com', 'pw');
        } catch (_) {
          // Expected.
        }

        expect(
          captured['X-Publishable-Key'],
          'pk_test_endtoend',
          reason:
              'AuthManager(AuthConfig) must thread publishableKey through to '
              'the generated HTTP client.',
        );

        manager.dispose();
      },
    );
  });
}
