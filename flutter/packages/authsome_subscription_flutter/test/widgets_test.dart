import 'dart:async';
import 'dart:convert';

import 'package:authsome_subscription_flutter/authsome_subscription_flutter.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';

Map<String, Object?> planJson(String slug, [Map<String, Object?> overrides = const {}]) => {
      'id': 'plan_$slug',
      'name': '${slug[0].toUpperCase()}${slug.substring(1)}',
      'slug': slug,
      'description': '',
      'currency': 'usd',
      'status': 'active',
      'trial_days': 0,
      'is_addon': false,
      'amount_minor': 2900,
      'billing_period': 'monthly',
      'features': [
        {'key': 'sso', 'name': 'SSO', 'type': 'boolean', 'limit': 1, 'soft_limit': false},
      ],
      ...overrides,
    };

Map<String, Object?> subJson([Map<String, Object?> overrides = const {}]) => {
      'id': 'sub_1',
      'tenant_id': 'org_1',
      'status': 'active',
      'plan': planJson('pro'),
      'current_period_start': '2026-11-01T12:00:00Z',
      'current_period_end': '2026-12-01T12:00:00Z',
      'cancel_at': null,
      ...overrides,
    };

typedef Handler = FutureOr<http.Response> Function(http.Request req);
http.Response _json(Object? body, [int status = 200]) => http.Response(jsonEncode(body), status);

class _Harness {
  _Harness() {
    routes.addAll({
      'GET /v1/me': (_) => _json({
            'id': 'usr_1', 'app_id': 'app_1', 'env_id': 'env_1', 'email': 'a@test',
            'email_verified': true, 'first_name': 'A', 'last_name': 'B', 'banned': false,
            'phone_verified': false, 'roles': <String>[], 'active_org_id': 'org_1',
            'created_at': '2026-01-01T00:00:00Z', 'updated_at': '2026-01-01T00:00:00Z',
          }),
      'GET /v1/billing/subscription': (_) => _json({
            'subscription': subJson({'seats': {'feature_key': 'seats', 'used': 3, 'limit': 10}}),
            'can_manage': true,
          }),
      'GET /v1/billing/entitlements': (_) => _json({
            'entitlements': {
              'sso': {'allowed': true, 'feature': 'sso', 'used': 2, 'limit': 10, 'remaining': 8},
            },
          }),
      'GET /v1/billing/plans': (_) => _json({
            'plans': [
              planJson('free', {'amount_minor': null}),
              planJson('pro'),
              planJson('team', {'amount_minor': 9900}),
            ],
          }),
    });
  }

  final routes = <String, Handler>{};
  final calls = <String>[];

  Future<AuthNotifier> signedIn() => _auth(withSession: true);

  Future<AuthNotifier> signedOut() => _auth(withSession: false);

  Future<AuthNotifier> _auth({required bool withSession}) async {
    final storage = MemoryTokenStorage();
    if (withSession) {
      await storage.setItem('authsome:session', jsonEncode({
      'session_token': 'tok',
      'refresh_token': 'ref',
        'expires_at': DateTime.now().add(const Duration(hours: 1)).toUtc().toIso8601String(),
      }));
    }
    final auth = AuthNotifier(AuthConfig(
      baseUrl: 'https://api.test',
      storage: storage,
      plugins: const [SubscriptionPlugin()],
      httpClient: MockClient((req) async {
        final key = '${req.method} ${req.url.path}';
        calls.add(key);
        final handler = routes[key];
        return handler == null ? _json({'error': 'no route $key'}, 404) : handler(req);
      }),
    ));
    await auth.initialize();
    return auth;
  }

  Future<AuthNotifier> pump(WidgetTester tester, Widget child, {bool signedOut = false}) async {
    final auth = await tester.runAsync(signedOut ? this.signedOut : signedIn);
    await tester.pumpWidget(AuthProvider.test(
      notifier: auth!,
      child: MaterialApp(home: Scaffold(body: SingleChildScrollView(child: child))),
    ));
    await _settle(tester);
    return auth;
  }
}

// The mock client answers on real async gaps, so settle with real time.
Future<void> _settle(WidgetTester tester) async {
  for (var i = 0; i < 5; i++) {
    await tester.runAsync(() => Future<void>.delayed(const Duration(milliseconds: 10)));
    await tester.pump();
  }
}

void main() {
  testWidgets('SubscriptionBuilder loads the subscription', (tester) async {
    await _Harness().pump(tester, SubscriptionBuilder(
      builder: (context, snap) => Text(snap.subscription.data?.plan.name ?? 'none'),
    ));
    expect(find.text('Pro'), findsOneWidget);
  });

  testWidgets('EntitlementGate shows the child only when the plan allows it', (tester) async {
    final h = _Harness();
    await h.pump(tester, const Column(children: [
      EntitlementGate(feature: 'sso', fallback: Text('upgrade sso'), child: Text('sso settings')),
      EntitlementGate(feature: 'audit', fallback: Text('upgrade audit'), child: Text('audit log')),
    ]));
    expect(find.text('sso settings'), findsOneWidget);
    expect(find.text('audit log'), findsNothing);
    expect(find.text('upgrade audit'), findsOneWidget);
  });

  testWidgets('EntitlementGate fails closed on an error', (tester) async {
    final h = _Harness();
    h.routes['GET /v1/billing/entitlements'] = (_) => _json({'error': 'down'}, 500);
    await h.pump(tester, const EntitlementGate(feature: 'sso', fallback: Text('upgrade'), child: Text('sso settings')));
    expect(find.text('sso settings'), findsNothing);
    expect(find.text('upgrade'), findsOneWidget);
  });

  testWidgets('PricingTable lists plans and changes plan', (tester) async {
    final h = _Harness();
    var patched = 0;
    h.routes['PATCH /v1/billing/subscription'] = (_) {
      patched++;
      return _json({'subscription': subJson({'plan': planJson('team', {'amount_minor': 9900})})});
    };
    await h.pump(tester, const PricingTable());
    expect(find.text(r'$99.00'), findsOneWidget);
    expect(find.text('No charge'), findsOneWidget);
    await tester.tap(find.text('Choose Team'));
    await _settle(tester);
    expect(patched, 1);
    expect(find.text('Choose Pro'), findsOneWidget);
  });

  testWidgets('PricingTable hands a checkout URL to onCheckout', (tester) async {
    final h = _Harness();
    h.routes['GET /v1/billing/subscription'] = (_) => _json({'subscription': null, 'can_manage': true});
    h.routes['POST /v1/billing/subscription'] = (_) => _json({'checkout_url': 'https://pay.example/1'});
    Uri? opened;
    await h.pump(tester, PricingTable(onCheckout: (uri) => opened = uri));
    await tester.tap(find.text('Choose Pro'));
    await _settle(tester);
    expect(opened, Uri.parse('https://pay.example/1'));
  });

  testWidgets('PricingTable shows plain words, not the server message, when a change fails', (tester) async {
    final h = _Harness();
    h.routes['PATCH /v1/billing/subscription'] =
        (_) => _json({'error': 'already subscribed: use PATCH to change plan'}, 409);
    Object? reported;
    await h.pump(tester, PricingTable(onError: (e) => reported = e));
    await tester.tap(find.text('Choose Team'));
    await _settle(tester);
    expect(find.text("Couldn't change your plan. Try again."), findsOneWidget);
    expect(find.textContaining('already subscribed'), findsNothing);
    expect(reported, isA<AuthClientException>());
  });

  testWidgets('PricingTable reports a checkout it has nowhere to send', (tester) async {
    final h = _Harness();
    h.routes['GET /v1/billing/subscription'] = (_) => _json({'subscription': null, 'can_manage': true});
    h.routes['POST /v1/billing/subscription'] = (_) => _json({'checkout_url': 'https://pay.example/1'});
    Object? reported;
    await h.pump(tester, PricingTable(onError: (e) => reported = e));
    await tester.tap(find.text('Choose Pro'));
    await _settle(tester);
    expect(find.text("Couldn't open the payment page."), findsOneWidget);
    expect(reported, isA<StateError>());
    expect('$reported', contains('onCheckout'));
  });

  testWidgets('PricingTable says which plan is highlighted, not only with colour', (tester) async {
    await _Harness().pump(tester, const PricingTable(highlight: 'team'));
    expect(find.text('Recommended'), findsOneWidget);
  });

  testWidgets('widgets stop loading when nobody is signed in', (tester) async {
    await _Harness().pump(
      tester,
      const Column(children: [
        EntitlementGate(feature: 'sso', fallback: Text('locked'), loading: Text('checking'), child: Text('open')),
        CurrentPlanCard(),
        PricingTable(),
        UsageMeter(feature: 'sso'),
      ]),
      signedOut: true,
    );
    expect(find.text('locked'), findsOneWidget);
    expect(find.text('checking'), findsNothing);
    expect(find.byType(CircularProgressIndicator), findsNothing);
    expect(find.byType(LinearProgressIndicator), findsNothing);
    expect(find.text('No plan yet'), findsOneWidget);
  });

  testWidgets('CurrentPlanCard says when a canceled plan ends and keeps it', (tester) async {
    final h = _Harness();
    h.routes['GET /v1/billing/subscription'] = (_) => _json({
          'subscription': subJson({'cancel_at': '2026-12-01T12:00:00Z'}),
          'can_manage': true,
        });
    var resumed = 0;
    h.routes['POST /v1/billing/subscription/resume'] = (_) {
      resumed++;
      return _json({'subscription': subJson()});
    };
    await h.pump(tester, const CurrentPlanCard());
    expect(find.textContaining('Ends on Dec 1, 2026'), findsOneWidget);
    await tester.tap(find.text('Keep plan'));
    await _settle(tester);
    expect(resumed, 1);
    expect(find.textContaining('Renews on'), findsOneWidget);
  });

  testWidgets('CurrentPlanCard shows plain words when cancel fails', (tester) async {
    final h = _Harness();
    h.routes['POST /v1/billing/subscription/cancel'] = (_) => _json({'error': 'ledger: row locked'}, 500);
    Object? reported;
    await h.pump(tester, CurrentPlanCard(onError: (e) => reported = e));
    await tester.tap(find.text('Cancel plan'));
    await _settle(tester);
    expect(find.text("Couldn't update your plan. Try again."), findsOneWidget);
    expect(reported, isNotNull);
  });

  testWidgets('CurrentPlanCard shows seats and no buttons to a member', (tester) async {
    final h = _Harness();
    h.routes['GET /v1/billing/subscription'] = (_) => _json({
          'subscription': subJson({'seats': {'feature_key': 'seats', 'used': 3, 'limit': 10}}),
          'can_manage': false,
        });
    await h.pump(tester, const CurrentPlanCard());
    expect(find.text('3 of 10 seats'), findsOneWidget);
    expect(find.byType(FilledButton), findsNothing);
    expect(find.byType(OutlinedButton), findsNothing);
  });

  testWidgets('UsageMeter shows use against the limit', (tester) async {
    await _Harness().pump(tester, const UsageMeter(feature: 'sso', label: 'SSO connections'));
    expect(find.text('2 of 10'), findsOneWidget);
    expect(find.text('SSO connections'), findsOneWidget);
  });

  testWidgets('ManageBillingButton opens the portal', (tester) async {
    final h = _Harness();
    h.routes['POST /v1/billing/portal'] = (_) => _json({'url': 'https://pay.example/portal'});
    Uri? opened;
    await h.pump(tester, ManageBillingButton(onOpen: (uri) => opened = uri));
    await tester.tap(find.text('Manage billing'));
    await _settle(tester);
    expect(opened, Uri.parse('https://pay.example/portal'));
  });
}
