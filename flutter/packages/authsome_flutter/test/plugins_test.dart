import 'dart:convert';

import 'package:authsome_flutter/authsome_flutter.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';

class _GreeterPlugin extends AuthPlugin<_Greeter> {
  @override
  String get name => 'greeter';
  @override
  _Greeter init(PluginContext ctx) => _Greeter();
}

class _Greeter {
  String hello() => 'hi';
}

AuthNotifier _notifier({Map<String, Object?> me = const {}}) {
  final storage = MemoryTokenStorage()
    ..setItem('authsome:session', jsonEncode({
      'session_token': 'tok',
      'refresh_token': 'refresh',
      'expires_at': DateTime.now().add(const Duration(hours: 1)).toUtc().toIso8601String(),
    }));
  final mock = MockClient((req) async {
    switch (req.url.path) {
      case '/v1/me':
        return http.Response(jsonEncode({
          'id': 'usr_1', 'app_id': 'app_1', 'env_id': 'env_1', 'email': 'a@test',
          'email_verified': true, 'first_name': 'A', 'last_name': 'B', 'banned': false,
          'phone_verified': false, 'roles': <String>[],
          'created_at': '2026-01-01T00:00:00Z', 'updated_at': '2026-01-01T00:00:00Z',
          ...me,
        }), 200);
      case '/v1/me/switch-org':
        final orgId = (jsonDecode(req.body) as Map)['org_id'];
        return http.Response(jsonEncode({'session_id': 'ses_1', 'org_id': orgId}), 200);
    }
    return http.Response('{}', 404);
  });
  return AuthNotifier(AuthConfig(
    baseUrl: 'https://api.test',
    httpClient: mock,
    storage: storage,
    plugins: [_GreeterPlugin()],
  ));
}

void main() {
  test('AuthNotifier keeps the configured plugins', () {
    final auth = _notifier();
    addTearDown(auth.dispose);
    expect(auth.plugin<_Greeter>().hello(), 'hi');
  });

  test('AuthNotifier reports the active org and notifies on a switch', () async {
    final auth = _notifier(me: {'active_org_id': 'org_1'});
    addTearDown(auth.dispose);
    await auth.initialize();
    expect(auth.activeOrgId, 'org_1');

    var notified = 0;
    auth.addListener(() => notified++);
    await auth.switchOrg('org_2');
    expect(auth.activeOrgId, 'org_2');
    expect(notified, greaterThan(0));
  });

  testWidgets('context.auth.plugin finds the plugin from the widget tree', (tester) async {
    final auth = _notifier();
    addTearDown(auth.dispose);
    String? greeting;
    await tester.pumpWidget(AuthProvider.test(
      notifier: auth,
      child: Builder(builder: (context) {
        greeting = context.auth.plugin<_Greeter>().hello();
        return const SizedBox.shrink();
      }),
    ));
    expect(greeting, 'hi');
  });
}
