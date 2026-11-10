import 'dart:async';
import 'dart:convert';

import 'package:authsome_core/authsome_core.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:test/test.dart';

/// A plugin whose API hands back its context so tests can poke it.
class _ProbePlugin extends AuthPlugin<_Probe> {
  _ProbePlugin({this.pluginName = 'probe', this.fail = false});
  final String pluginName;
  final bool fail;
  int inits = 0;

  @override
  String get name => pluginName;

  @override
  _Probe init(PluginContext ctx) {
    inits++;
    if (fail) throw StateError('bad config');
    return _Probe(ctx);
  }
}

class _Probe {
  _Probe(this.ctx);
  final PluginContext ctx;
}

class _OtherPlugin extends AuthPlugin<String> {
  @override
  String get name => 'other';
  @override
  String init(PluginContext ctx) => 'other';
}

final _session = {
  'session_token': 'token-1',
  'refresh_token': 'refresh-1',
  'expires_at': DateTime.now().add(const Duration(hours: 1)).toUtc().toIso8601String(),
};

final _me = {
  'id': 'usr_1',
  'app_id': 'app_1',
  'env_id': 'env_1',
  'email': 'ada@test',
  'email_verified': true,
  'first_name': 'Ada',
  'last_name': 'Lovelace',
  'banned': false,
  'phone_verified': false,
  'roles': <String>[],
  'created_at': '2026-01-01T00:00:00Z',
  'updated_at': '2026-01-01T00:00:00Z',
};

typedef _Routes = Map<String, FutureOr<http.Response> Function(http.Request)>;

({AuthManager manager, _Routes routes, List<http.Request> calls, MemoryTokenStorage storage}) _setup({
  bool signedIn = false,
  Map<String, Object?> me = const {},
  List<AuthPlugin<Object>> plugins = const [],
  ClientConfig? clientConfig,
  String? publishableKey,
}) {
  final routes = <String, FutureOr<http.Response> Function(http.Request)>{
    'GET /v1/me': (_) => http.Response(jsonEncode({..._me, ...me}), 200),
    'POST /v1/signout': (_) => http.Response('{"status":"ok"}', 200),
  };
  final calls = <http.Request>[];
  final mock = MockClient((req) async {
    calls.add(req);
    final handler = routes['${req.method} ${req.url.path}'];
    if (handler == null) return http.Response('{"error":"no route"}', 404);
    return handler(req);
  });
  final storage = MemoryTokenStorage();
  if (signedIn) storage.setItem('authsome:session', jsonEncode(_session));
  final manager = AuthManager(AuthConfig(
    baseUrl: 'https://api.test',
    httpClient: mock,
    storage: storage,
    plugins: plugins,
    initialClientConfig: clientConfig,
    publishableKey: publishableKey,
  ));
  addTearDown(manager.dispose);
  return (manager: manager, routes: routes, calls: calls, storage: storage);
}

void main() {
  group('plugins', () {
    test('returns what init built, once and lazily', () {
      final probe = _ProbePlugin();
      final s = _setup(plugins: [probe]);
      expect(probe.inits, 0);
      final first = s.manager.plugin<_Probe>();
      expect(identical(s.manager.plugin<_Probe>(), first), isTrue);
      expect(probe.inits, 1);
      expect(s.manager.hasPlugin<_Probe>(), isTrue);
      expect(s.manager.hasPlugin<String>(), isFalse);
    });

    test('throws a named error for a plugin that was not registered', () {
      final s = _setup();
      expect(() => s.manager.plugin<_Probe>(), throwsA(isA<PluginNotRegisteredError>()));
    });

    test('refuses the same plugin name twice', () {
      expect(() => _setup(plugins: [_ProbePlugin(), _ProbePlugin()]), throwsA(isA<ArgumentError>()));
    });

    test('surfaces an init error on first use, not at construction', () {
      final s = _setup(plugins: [_ProbePlugin(fail: true)]);
      expect(() => s.manager.plugin<_Probe>(), throwsA(isA<StateError>()));
    });

    test('reports whether the server has the plugin', () {
      expect(_setup(plugins: [_ProbePlugin()]).manager.plugin<_Probe>().ctx.isServerEnabled, isNull);
      const config = ClientConfig(supportedPlugins: ['probe']);
      expect(_setup(plugins: [_ProbePlugin()], clientConfig: config).manager.plugin<_Probe>().ctx.isServerEnabled, isTrue);
      final other = _setup(plugins: [_ProbePlugin(pluginName: 'elsewhere')], clientConfig: config);
      expect(other.manager.plugin<_Probe>().ctx.isServerEnabled, isFalse);
      expect(_setup(plugins: [_OtherPlugin()]).manager.plugin<String>(), 'other');
    });

    test('sends authenticated requests with the session token and query', () async {
      final s = _setup(signedIn: true, plugins: [_ProbePlugin()], publishableKey: 'pk_test');
      s.routes['GET /v1/billing/subscription'] = (_) => http.Response('{"subscription":null}', 200);
      await s.manager.initialize();

      final body = await s.manager.plugin<_Probe>().ctx.request(
        'GET', '/v1/billing/subscription', query: {'tenant_id': 'org_1', 'skip': null});
      expect(body, {'subscription': null});
      final req = s.calls.last;
      expect(req.url.toString(), 'https://api.test/v1/billing/subscription?tenant_id=org_1');
      expect(req.headers['Authorization'], 'Bearer token-1');
      expect(req.headers['X-Publishable-Key'], 'pk_test');
    });

    test('throws AuthClientException with the server message', () async {
      final s = _setup(plugins: [_ProbePlugin()]);
      s.routes['POST /v1/thing'] = (_) => http.Response('{"error":"nope","type":"forbidden"}', 403);
      await expectLater(
        s.manager.plugin<_Probe>().ctx.request('POST', '/v1/thing', body: {'a': 1}),
        throwsA(isA<AuthClientException>()
            .having((e) => e.message, 'message', 'nope')
            .having((e) => e.code, 'code', 403)
            .having((e) => e.type, 'type', 'forbidden')),
      );
      expect(s.calls.last.body, '{"a":1}');
    });
  });

  group('lifecycle events', () {
    test('fires signedIn, sessionRefreshed and signedOut once each', () async {
      final s = _setup(signedIn: true);
      final seen = <AuthEvent>[];
      s.manager.events.listen(seen.add);
      await s.manager.initialize();
      expect(seen, [AuthEvent.signedIn]);

      s.routes['POST /v1/refresh'] = (_) => http.Response(jsonEncode({
            ..._session,
            'session_token': 'token-2',
            'refresh_token': 'refresh-2',
          }), 200);
      await s.manager.refreshNow();
      expect(seen, [AuthEvent.signedIn, AuthEvent.sessionRefreshed]);

      await s.manager.signOut();
      expect(seen, [AuthEvent.signedIn, AuthEvent.sessionRefreshed, AuthEvent.signedOut]);
    });
  });

  group('active organization', () {
    test('restores the active org from /v1/me', () async {
      final s = _setup(signedIn: true, me: {'active_org_id': 'org_1'});
      expect(s.manager.activeOrgId, isNull);
      await s.manager.initialize();
      expect(s.manager.activeOrgId, 'org_1');
    });

    test('switches org, keeps a reissued token and announces it', () async {
      final s = _setup(signedIn: true);
      await s.manager.initialize();
      final seen = <AuthEvent>[];
      s.manager.events.listen(seen.add);
      s.routes['POST /v1/me/switch-org'] = (req) {
        expect(jsonDecode(req.body), {'org_id': 'org_2'});
        expect(req.headers['Authorization'], 'Bearer token-1');
        return http.Response(jsonEncode({
          'session_id': 'ses_1',
          'org_id': 'org_2',
          'session_token': 'token-org2',
          'expires_at': _session['expires_at'],
        }), 200);
      };

      await s.manager.switchOrg('org_2');
      expect(s.manager.activeOrgId, 'org_2');
      expect(seen, [AuthEvent.orgChanged]);
      expect(s.manager.getSessionToken(), 'token-org2');
      final stored = jsonDecode((await s.storage.getItem('authsome:session'))!) as Map;
      expect(stored['session_token'], 'token-org2');
    });

    test('refuses to switch when signed out', () async {
      final s = _setup();
      await expectLater(s.manager.switchOrg('org_2'), throwsA(isA<StateError>()));
    });

    test('forgets the org on sign-out', () async {
      final s = _setup(signedIn: true, me: {'active_org_id': 'org_1'});
      await s.manager.initialize();
      expect(s.manager.activeOrgId, 'org_1');
      await s.manager.signOut();
      expect(s.manager.activeOrgId, isNull);
    });
  });

  group('review fixes', () {
    http.Response switched(String org, {String? token}) => http.Response(jsonEncode({
          'session_id': 'ses_1',
          'org_id': org,
          if (token != null) 'session_token': token,
          if (token != null) 'expires_at': _session['expires_at'],
        }), 200);

    test('keeps a token a refresh rotated while the switch was in flight', () async {
      final s = _setup(signedIn: true);
      await s.manager.initialize();
      final gate = Completer<http.Response>();
      s.routes['POST /v1/me/switch-org'] = (_) => gate.future;
      s.routes['POST /v1/refresh'] = (_) => http.Response(jsonEncode({
            ..._session,
            'session_token': 'token-2',
            'refresh_token': 'refresh-2',
          }), 200);

      final switching = s.manager.switchOrg('org_2');
      final refreshing = s.manager.refreshNow();
      await Future<void>.delayed(const Duration(milliseconds: 20));
      gate.complete(switched('org_2', token: 'token-org2'));
      await Future.wait([switching, refreshing]);

      final stored = jsonDecode((await s.storage.getItem('authsome:session'))!) as Map;
      expect(stored['refresh_token'], 'refresh-2');
    });

    test('does not bring back a session that signed out during the switch', () async {
      final s = _setup(signedIn: true);
      await s.manager.initialize();
      final gate = Completer<http.Response>();
      s.routes['POST /v1/me/switch-org'] = (_) => gate.future;

      final switching = s.manager.switchOrg('org_2');
      await s.manager.signOut();
      gate.complete(switched('org_2', token: 'token-org2'));
      await switching.catchError((_) {});

      expect(s.manager.state, isA<AuthUnauthenticated>());
      expect(await s.storage.getItem('authsome:session'), isNull);
      expect(s.manager.activeOrgId, isNull);
    });

    test('treats signing in as someone else as a sign-out and a sign-in', () async {
      final s = _setup(signedIn: true, me: {'active_org_id': 'org_A'});
      await s.manager.initialize();
      final seen = <AuthEvent>[];
      s.manager.events.listen(seen.add);
      s.routes['POST /v1/signin'] = (_) => http.Response(jsonEncode({
            'user': {..._me, 'id': 'usr_2'},
            'session_token': 't2',
            'refresh_token': 'r2',
            'expires_at': _session['expires_at'],
          }), 200);

      await s.manager.signIn('grace@test', 'pw');
      expect(seen, [AuthEvent.signedOut, AuthEvent.signedIn]);
      expect(s.manager.activeOrgId, isNull);
    });

    test('refuses an untyped or ambiguous plugin lookup', () {
      final untyped = _setup(plugins: [_ProbePlugin()]);
      expect(() => untyped.manager.plugin(), throwsA(isA<ArgumentError>()));

      final twice = _setup(plugins: [_OtherPlugin(), _OtherNamePlugin()]);
      expect(() => twice.manager.plugin<String>(), throwsA(isA<StateError>()));
    });

    test('clears the org when switching to none', () async {
      final s = _setup(signedIn: true, me: {'active_org_id': 'org_A'});
      await s.manager.initialize();
      s.routes['POST /v1/me/switch-org'] = (_) => http.Response('{"session_id":"ses_1"}', 200);
      await s.manager.switchOrg('');
      expect(s.manager.activeOrgId, isNull);
    });
  });
}

class _OtherNamePlugin extends AuthPlugin<String> {
  @override
  String get name => 'other-two';
  @override
  String init(PluginContext ctx) => 'two';
}
