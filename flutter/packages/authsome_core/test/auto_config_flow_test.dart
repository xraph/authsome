import 'dart:convert';
import 'package:authsome_core/authsome_core.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:test/test.dart';

const user = {
  'id': 'u_1',
  'app_id': 'app_1',
  'env_id': 'env_1',
  'email': 'ada@example.com',
  'email_verified': false,
  'first_name': 'Ada',
  'last_name': '',
  'banned': false,
  'phone_verified': false,
  'roles': <String>[],
  'created_at': '2024-01-01T00:00:00Z',
  'updated_at': '2024-01-01T00:00:00Z'
};
const authResponse = {
  'user': user,
  'session_token': 'token',
  'refresh_token': 'refresh',
  'expires_at': '2099-01-01T00:00:00Z'
};
http.Response jsonResponse(Object body, [int status = 200]) =>
    http.Response(jsonEncode(body), status,
        headers: {'content-type': 'application/json'});
void main() {
  test('signup_enabled round trips without a version', () {
    final config = ClientConfig.fromJson({'signup_enabled': false});
    expect(config.signupEnabled, false);
    expect(config.toJson(), {'signup_enabled': false});
  });
  test('cache is scoped by server and app key', () async {
    final storage = MemoryTokenStorage();
    var requests = 0;
    final httpClient =
        MockClient((_) async => jsonResponse({'app_id': 'app_${++requests}'}));
    Future<String?> config(String host, String key) async {
      final manager = AuthManager(AuthConfig(
          baseUrl: host,
          publishableKey: key,
          storage: storage,
          httpClient: httpClient));
      addTearDown(manager.dispose);
      return (await manager.fetchClientConfig()).appId;
    }

    expect(await config('https://a.test', 'pk_a'), 'app_1');
    expect(await config('https://a.test/', 'pk_a'), 'app_1');
    expect(await config('https://a.test', 'pk_b'), 'app_2');
    expect(await config('https://b.test', 'pk_a'), 'app_3');
    expect(requests, 3);
  });
  test('closed signup makes no account request', () async {
    final calls = <String>[];
    final manager = AuthManager(AuthConfig(
        baseUrl: 'https://a.test',
        initialClientConfig: const ClientConfig(signupEnabled: false),
        httpClient: MockClient((req) async {
          calls.add(req.url.path);
          return jsonResponse(authResponse);
        })));
    addTearDown(manager.dispose);
    await manager.signUp('ada@example.com', 'pw');
    expect(manager.state, isA<AuthError>());
    expect(calls, isEmpty);
  });
  test('unknown verification policy retains no session', () async {
    final manager = AuthManager(AuthConfig(
        baseUrl: 'https://a.test',
        httpClient: MockClient((_) async => jsonResponse(authResponse))));
    addTearDown(manager.dispose);
    await manager.signUp('ada@example.com', 'pw');
    expect(manager.state, isA<AuthVerificationPending>());
    expect(manager.getSessionToken(), isNull);
  });
  for (final status in [200, 401, 503]) {
    test(
        'optional verification validates the session before completing, status $status',
        () async {
      final paths = <String>[];
      final manager = AuthManager(AuthConfig(
          baseUrl: 'https://a.test',
          initialClientConfig: const ClientConfig(
              emailVerification:
                  EmailVerificationConfig(enabled: true, required: false)),
          httpClient: MockClient((req) async {
            paths.add(req.url.path);
            if (req.url.path == '/v1/me') {
              return jsonResponse(
                  status == 200 ? user : {'error': 'session rejected'}, status);
            }
            return jsonResponse(authResponse);
          })));
      addTearDown(manager.dispose);
      await manager.signUp('ada@example.com', 'pw');
      expect(paths, ['/v1/signup', '/v1/me']);
      if (status == 200) {
        expect(manager.state, isA<AuthAuthenticated>());
      } else {
        expect(manager.state,
            status == 401 ? isA<AuthVerificationPending>() : isA<AuthError>());
        expect(manager.getSessionToken(), isNull);
      }
    });
  }
  test('MFA uses the ticket and keeps it after a failed code', () async {
    var attempts = 0;
    final bodies = <Map<String, dynamic>>[];
    final manager = AuthManager(AuthConfig(
        baseUrl: 'https://a.test',
        httpClient: MockClient((req) async {
          if (req.url.path == '/v1/signin') {
            return jsonResponse({
              'error': 'MFA required',
              'type': 'mfa_required',
              'mfa_ticket': 'ticket',
              'available_methods': ['totp']
            }, 403);
          }
          bodies.add(jsonDecode(req.body) as Map<String, dynamic>);
          return ++attempts == 1
              ? jsonResponse({'error': 'Invalid code'}, 401)
              : jsonResponse(authResponse);
        })));
    addTearDown(manager.dispose);
    await manager.signIn('ada@example.com', 'pw');
    await expectLater(manager.submitMFAChallenge('000000'),
        throwsA(isA<AuthClientException>()));
    expect(manager.state, isA<AuthMfaRequired>());
    await manager.submitMFAChallenge('123456');
    expect(bodies, [
      {'mfa_ticket': 'ticket', 'code': '000000'},
      {'mfa_ticket': 'ticket', 'code': '123456'}
    ]);
    expect(manager.state, isA<AuthAuthenticated>());
  });
}
