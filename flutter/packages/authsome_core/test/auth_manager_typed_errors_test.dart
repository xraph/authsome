/// Tests the typed-error → state-machine path on [AuthManager].
///
/// Confirms parity with the React `auth.ts` behaviour:
/// - `type: 'mfa_required'` → emit [AuthMfaRequired] with the ticket
/// - `type: 'email_not_verified'` → emit [AuthEmailNotVerified] AND rethrow
/// - any other error → emit [AuthError]
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

http.Response _jsonError(int status, Map<String, dynamic> body) {
  return http.Response(
    jsonEncode(body),
    status,
    headers: {'content-type': 'application/json'},
  );
}

void main() {
  group('AuthManager._handleError → typed dispatch', () {
    test(
      'signIn emits AuthMfaRequired with ticket and methods on type:mfa_required',
      () async {
        final mock = MockClient((req) async {
          return _jsonError(403, {
            'error': 'MFA required',
            'code': 403,
            'type': 'mfa_required',
            'details': {
              'mfa_ticket': 'tk_abc',
              'available_methods': ['totp', 'sms'],
            },
          });
        });

        final states = <AuthState>[];
        final manager = AuthManager.withClient(
          client: _client(mock),
          onStateChange: states.add,
        );

        await manager.signIn('user@example.com', 'pw');

        final last = states.last;
        expect(last, isA<AuthMfaRequired>());
        final mfa = last as AuthMfaRequired;
        expect(mfa.email, 'user@example.com');
        expect(mfa.mfaTicket, 'tk_abc');
        expect(mfa.availableMethods, ['totp', 'sms']);

        manager.dispose();
      },
    );

    test(
      'signIn emits AuthEmailNotVerified and rethrows on type:email_not_verified',
      () async {
        final mock = MockClient((req) async {
          return _jsonError(403, {
            'error': 'Please verify your email',
            'code': 403,
            'type': 'email_not_verified',
          });
        });

        final states = <AuthState>[];
        final manager = AuthManager.withClient(
          client: _client(mock),
          onStateChange: states.add,
        );

        await expectLater(
          () => manager.signIn('user@example.com', 'pw'),
          throwsA(isA<AuthClientException>()
              .having((e) => e.isEmailNotVerified, 'isEmailNotVerified',
                  isTrue)),
        );

        final last = states.last;
        expect(last, isA<AuthEmailNotVerified>());
        expect((last as AuthEmailNotVerified).email, 'user@example.com');

        manager.dispose();
      },
    );

    test(
      'signIn emits AuthError with the message on a generic non-typed failure',
      () async {
        final mock = MockClient((req) async {
          return _jsonError(500, {
            'error': 'Database unavailable',
            'code': 500,
          });
        });

        final states = <AuthState>[];
        final manager = AuthManager.withClient(
          client: _client(mock),
          onStateChange: states.add,
        );

        await manager.signIn('user@example.com', 'pw');

        final last = states.last;
        expect(last, isA<AuthError>());
        expect((last as AuthError).error, 'Database unavailable');

        manager.dispose();
      },
    );
  });
}
