/// Pins the client-config fields the Flutter UI auto-configures from, and
/// the request shapes they drive. The React SDK reads the same fields
/// (`ui/packages/core/src/types.ts` `ClientConfig`), so these keep the two
/// in step.
library;

import 'dart:convert';

import 'package:authsome_core/authsome_core.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:test/test.dart';

const _user = {
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
  'roles': <String>[],
  'updated_at': '2024-01-01T00:00:00Z',
};

final _authBody = jsonEncode({
  'session_token': 't_sess',
  'refresh_token': 't_refresh',
  'expires_at': '2099-01-01T00:00:00Z',
  'user': _user,
});

/// Records every request and answers with [body].
MockClient _recording(List<http.Request> seen, {String? body}) {
  return MockClient((req) async {
    seen.add(req);
    return http.Response(
      body ?? (req.url.path == '/v1/me' ? jsonEncode(_user) : _authBody),
      200,
      headers: {'content-type': 'application/json'},
    );
  });
}

AuthSomeClient _client(MockClient mock) => AuthSomeClient(
      AuthClientConfig(baseUrl: 'http://test.local', httpClient: mock),
    );

void main() {
  group('ClientConfig.fromJson', () {
    test('parses every auto-config section the server sends', () {
      final config = ClientConfig.fromJson({
        'branding': {'app_name': 'Acme', 'logo_url': 'https://x/logo.png'},
        'magiclink': {'enabled': true},
        'sso': {
          'enabled': true,
          'connections': [
            {'id': 'okta', 'name': 'Okta'},
          ],
        },
        'waitlist': {'enabled': false},
        'email_verification': {'enabled': true, 'required': true},
        'device_authorization': {'enabled': true},
        'captcha': {
          'required': true,
          'provider': 'turnstile',
          'site_key': '0x4AAA',
        },
        'signup_fields': [
          {
            'key': 'company',
            'label': 'Company',
            'type': 'text',
            'order': 2,
            'validation': {'required': true, 'min_len': 2, 'min': 1},
          },
          {
            'key': 'plan',
            'label': 'Plan',
            'type': 'select',
            'default': 'pro',
            'order': 1,
            'options': [
              {'label': 'Pro', 'value': 'pro'},
            ],
          },
        ],
      });

      expect(config.branding?.appName, 'Acme');
      expect(config.magiclink?.enabled, isTrue);
      expect(config.sso?.connections.single.id, 'okta');
      expect(config.waitlist?.enabled, isFalse);
      expect(config.emailVerification?.required, isTrue);
      expect(config.deviceAuthorization?.enabled, isTrue);
      expect(config.captcha?.siteKey, '0x4AAA');
      expect(config.captcha?.provider, 'turnstile');
      expect(config.signupFields, hasLength(2));
      final company = config.signupFields!.first;
      expect(company.validation?.required, isTrue);
      expect(company.validation?.minLen, 2);
      expect(company.validation?.min, 1);
      expect(config.signupFields![1].defaultValue, 'pro');
      expect(config.signupFields![1].options?.single.value, 'pro');
    });

    test('round-trips through toJson', () {
      final json = {
        'waitlist': {'enabled': true},
        'email_verification': {'enabled': true, 'required': false},
        'captcha': {'required': false},
        'signup_fields': [
          {'key': 'k', 'label': 'K', 'type': 'text', 'order': 0},
        ],
      };
      expect(ClientConfig.fromJson(json).toJson(), json);
    });

    test('leaves the new sections null when the server omits them', () {
      final config = ClientConfig.fromJson(const {});
      expect(config.signupFields, isNull);
      expect(config.waitlist, isNull);
      expect(config.emailVerification, isNull);
      expect(config.deviceAuthorization, isNull);
      expect(config.captcha, isNull);
    });
  });

  group('requests', () {
    test('signIn sends captcha_token', () async {
      final seen = <http.Request>[];
      final manager = AuthManager.withClient(client: _client(_recording(seen)));

      await manager.signIn('user@example.com', 'pw', captchaToken: 'tok');

      final body = jsonDecode(seen.single.body) as Map<String, dynamic>;
      expect(body['captcha_token'], 'tok');
    });

    test('signUp splits fields into names, username and metadata', () async {
      final seen = <http.Request>[];
      final manager = AuthManager.withClient(client: _client(_recording(seen)));

      await manager.signUp(
        'new@example.com',
        'pw',
        fields: const {
          'first_name': 'Jane',
          'last_name': 'Doe',
          'username': 'jdoe',
          'company': 'Acme',
        },
        captchaToken: 'tok',
      );

      final body = jsonDecode(seen.single.body) as Map<String, dynamic>;
      expect(body['first_name'], 'Jane');
      expect(body['last_name'], 'Doe');
      expect(body['username'], 'jdoe');
      expect(body['metadata'], {'company': 'Acme'});
      expect(body['captcha_token'], 'tok');
    });

    test('signUp still splits a display name when fields carry no name',
        () async {
      final seen = <http.Request>[];
      final manager = AuthManager.withClient(client: _client(_recording(seen)));

      await manager.signUp(
        'new@example.com',
        'pw',
        name: 'Jane Doe',
        fields: const {'company': 'Acme'},
      );

      final body = jsonDecode(seen.single.body) as Map<String, dynamic>;
      expect(body['first_name'], 'Jane');
      expect(body['last_name'], 'Doe');
      expect(body['metadata'], {'company': 'Acme'});
    });

    test(
        'signUp lands in AuthVerificationPending when verification is required',
        () async {
      final seen = <http.Request>[];
      final manager = AuthManager.withClient(
        client: _client(_recording(seen)),
        initialClientConfig: const ClientConfig(
          emailVerification:
              EmailVerificationConfig(enabled: true, required: true),
        ),
      );

      await manager.signUp('new@example.com', 'pw');

      expect(manager.state, isA<AuthVerificationPending>());
      expect(
          (manager.state as AuthVerificationPending).email, 'new@example.com');
      expect(manager.getSessionToken(), isNull,
          reason: 'the unverified session must not be kept');
    });

    test('signUp signs in when verification is not required', () async {
      final seen = <http.Request>[];
      final manager = AuthManager.withClient(
        client: _client(_recording(seen)),
        initialClientConfig: const ClientConfig(
          emailVerification:
              EmailVerificationConfig(enabled: true, required: false),
        ),
      );

      await manager.signUp('new@example.com', 'pw');

      expect(manager.state, isA<AuthAuthenticated>());
    });

    test('startSSOLogin returns the IdP URL and passes return_url', () async {
      final seen = <http.Request>[];
      final manager = AuthManager.withClient(
        client: _client(_recording(
          seen,
          body: jsonEncode({'login_url': 'https://idp/x', 'state': 's'}),
        )),
      );

      final url = await manager.startSSOLogin(
        'okta',
        returnUrl: 'https://app.example.com/sso/callback',
      );

      expect(url, 'https://idp/x');
      expect(seen.single.url.path, '/v1/sso/okta/login');
      expect(seen.single.url.queryParameters['return_url'],
          'https://app.example.com/sso/callback');
    });

    test('completeSSOLogin exchanges the code without a bearer token',
        () async {
      final seen = <http.Request>[];
      final manager = AuthManager.withClient(client: _client(_recording(seen)));

      await manager.completeSSOLogin('otc_123');

      expect(seen.single.url.path, '/v1/sso/exchange');
      expect(jsonDecode(seen.single.body), {'code': 'otc_123'});
      expect(seen.single.headers.containsKey('Authorization'), isFalse);
      expect(manager.state, isA<AuthAuthenticated>());
    });

    test('joinWaitlist posts email and name', () async {
      final seen = <http.Request>[];
      final manager = AuthManager.withClient(
        client: _client(_recording(seen, body: '{}')),
      );

      await manager.joinWaitlist('w@example.com', name: 'Wes');

      expect(seen.single.url.path, '/v1/waitlist/join');
      expect(jsonDecode(seen.single.body), {
        'email': 'w@example.com',
        'name': 'Wes',
      });
    });

    test('approveDeviceAuthorization refuses without a session', () async {
      final manager = AuthManager.withClient(
        client: _client(_recording(<http.Request>[])),
      );

      expect(
        () => manager.approveDeviceAuthorization('ABCDEFGH'),
        throwsA(isA<AuthClientException>()),
      );
    });
  });
}
