/// Confirms that a malformed `expires_at` in the sign-in response does NOT
/// flip the user back to [AuthError] / [AuthUnauthenticated] after a
/// successful credential exchange.
///
/// Root cause of the "back to email step" symptom: `_scheduleRefresh`
/// called `DateTime.parse(session.expiresAt)` unguarded. If the backend
/// returned `expires_at` in any non-ISO format (Unix timestamp string,
/// space-separated, missing timezone, etc.), the parse threw a
/// `FormatException`, the exception propagated up to `signIn`'s catch,
/// `_handleError` reset state to `AuthError`, and the example app's
/// router redirect bounced the user from `/` back to `/sign-in` —
/// remounting `SignInPage` with a fresh `_step = email`.
library;

import 'dart:convert';

import 'package:authsome_core/authsome_core.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:test/test.dart';

AuthSomeClient _client(MockClient mock) {
  return AuthSomeClient(
    AuthClientConfig(baseUrl: 'http://test.local', httpClient: mock),
  );
}

void main() {
  group('AuthManager post-signin resilience', () {
    test(
      'signIn stays in AuthAuthenticated when expires_at is not a valid ISO 8601 string',
      () async {
        final mock = MockClient((req) async {
          return http.Response(
            jsonEncode({
              'session_token': 't_sess',
              'refresh_token': 't_refresh',
              // Unix-seconds string — DateTime.parse can't handle this.
              'expires_at': '1748371200',
              'user': {
                'app_id': 'app_1',
                'banned': false,
                'created_at': '2024-01-01T00:00:00Z',
                'email': 'user@example.com',
                'email_verified': true,
                'env_id': 'env_1',
                'first_name': 'Test',
                'id': 'u_1',
                'last_name': 'User',
                'phone_verified': false,
                'updated_at': '2024-01-01T00:00:00Z',
              },
            }),
            200,
            headers: {'content-type': 'application/json'},
          );
        });

        final states = <AuthState>[];
        final manager = AuthManager.withClient(
          client: _client(mock),
          onStateChange: states.add,
        );

        await manager.signIn('user@example.com', 'pw');

        // The user successfully exchanged credentials. Even if we cannot
        // schedule the next refresh, we must NOT undo the authentication.
        expect(
          manager.state,
          isA<AuthAuthenticated>(),
          reason: 'a malformed expires_at must not flip state back to Error',
        );
        expect(
          states.whereType<AuthError>(),
          isEmpty,
          reason: 'AuthError must not appear in the transition log',
        );

        manager.dispose();
      },
    );
  });
}
