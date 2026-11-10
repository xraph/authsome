import 'dart:convert';

import 'package:authsome_flutter_ui/authsome_flutter_ui.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:mocktail/mocktail.dart';

import '../_helpers/mock_auth_notifier.dart';

MockAuthNotifier _signedIn({String? activeOrgId}) {
  final mock = buildIdleMock(
    state: const AuthAuthenticated(
      user: null,
      session: Session(sessionToken: 'tok', refreshToken: 'r', expiresAt: '2099-01-01T00:00:00Z'),
    ),
  );
  when(() => mock.session).thenReturn(
    const Session(sessionToken: 'tok', refreshToken: 'r', expiresAt: '2099-01-01T00:00:00Z'),
  );
  when(() => mock.client).thenReturn(AuthSomeClient(AuthClientConfig(
    baseUrl: 'http://test.local',
    httpClient: MockClient((req) async => http.Response(jsonEncode({
          'organizations': [
            {'id': 'org_1', 'name': 'Acme', 'slug': 'acme', 'app_id': 'app_1', 'env_id': 'env_1', 'created_at': '2026-01-01T00:00:00Z', 'updated_at': '2026-01-01T00:00:00Z', 'created_by': 'usr_1'},
            {'id': 'org_2', 'name': 'Globex', 'slug': 'globex', 'app_id': 'app_1', 'env_id': 'env_1', 'created_at': '2026-01-01T00:00:00Z', 'updated_at': '2026-01-01T00:00:00Z', 'created_by': 'usr_1'},
          ],
        }), 200)),
  )));
  when(() => mock.activeOrgId).thenReturn(activeOrgId);
  when(() => mock.switchOrg(any())).thenAnswer((_) async {});
  return mock;
}

Future<void> _pump(WidgetTester tester, MockAuthNotifier auth, Widget child) async {
  await tester.pumpWidget(AuthProvider.test(
    notifier: auth,
    child: MaterialApp(home: Scaffold(body: Center(child: child))),
  ));
  await tester.pumpAndSettle();
}

void main() {
  testWidgets('without props it shows and switches the core active org', (tester) async {
    final auth = _signedIn(activeOrgId: 'org_1');
    await _pump(tester, auth, const OrgSwitcher());

    expect(find.text('Acme'), findsOneWidget);
    await tester.tap(find.text('Acme'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Globex').last);
    await tester.pumpAndSettle();

    verify(() => auth.switchOrg('org_2')).called(1);
  });

  testWidgets('controlled props still win', (tester) async {
    final auth = _signedIn(activeOrgId: 'org_1');
    String? picked;
    await _pump(tester, auth, OrgSwitcher(activeOrgId: 'org_2', onOrgChange: (id) => picked = id));

    expect(find.text('Globex'), findsOneWidget);
    await tester.tap(find.text('Globex'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Acme').last);
    await tester.pumpAndSettle();

    expect(picked, 'org_1');
    verifyNever(() => auth.switchOrg(any()));
  });
}
