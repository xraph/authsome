import 'dart:async';
import 'dart:convert';

import 'package:authsome_core/authsome_core.dart';
import 'package:authsome_subscription/authsome_subscription.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:test/test.dart';

Map<String, Object?> planJson(String slug, [Map<String, Object?> overrides = const {}]) => {
      'id': 'plan_$slug',
      'name': '${slug[0].toUpperCase()}${slug.substring(1)}',
      'slug': slug,
      'description': '',
      'currency': 'usd',
      'status': 'active',
      'trial_days': 0,
      'features_count': 1,
      'is_addon': false,
      'app_id': 'app_1',
      'base_amount': '29.00',
      'billing_period': 'monthly',
      'amount_minor': 2900,
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
      'current_period_start': '2026-11-01T00:00:00Z',
      'current_period_end': '2026-12-01T00:00:00Z',
      'trial_start': null,
      'trial_end': null,
      'cancel_at': null,
      'canceled_at': null,
      'paused_at': null,
      'app_id': 'app_1',
      ...overrides,
    };

final _me = {
  'id': 'usr_1',
  'app_id': 'app_user',
  'env_id': 'env_1',
  'email': 'a@test',
  'email_verified': true,
  'first_name': 'A',
  'last_name': 'B',
  'banned': false,
  'phone_verified': false,
  'roles': <String>[],
  'active_org_id': 'org_1',
  'created_at': '2026-01-01T00:00:00Z',
  'updated_at': '2026-01-01T00:00:00Z',
};

typedef Handler = FutureOr<http.Response> Function(http.Request req);

http.Response _json(Object? body, [int status = 200]) => http.Response(jsonEncode(body), status);

({AuthManager manager, Map<String, Handler> routes, List<String> calls, SubscriptionApi Function() api})
    _setup({List<String>? serverPlugins, String? configAppId}) {
  final routes = <String, Handler>{
    'GET /v1/me': (_) => _json(_me),
    'POST /v1/signout': (_) => _json({'status': 'ok'}),
    'GET /v1/billing/subscription': (_) => _json({'subscription': subJson(), 'can_manage': true}),
    'GET /v1/billing/entitlements': (_) => _json({
          'entitlements': {
            'sso': {'allowed': true, 'feature': 'sso', 'used': 0, 'limit': 1, 'remaining': 1},
          },
        }),
  };
  final calls = <String>[];
  final storage = MemoryTokenStorage()
    ..setItem('authsome:session', jsonEncode({
      'session_token': 'tok',
      'refresh_token': 'ref',
      'expires_at': DateTime.now().add(const Duration(hours: 1)).toUtc().toIso8601String(),
    }));
  final manager = AuthManager(AuthConfig(
    baseUrl: 'https://api.test',
    storage: storage,
    plugins: [SubscriptionPlugin()],
    initialClientConfig: serverPlugins == null && configAppId == null
        ? null
        : ClientConfig(supportedPlugins: serverPlugins, appId: configAppId),
    httpClient: MockClient((req) async {
      final key = '${req.method} ${req.url.path}';
      calls.add('$key${req.url.hasQuery ? '?${req.url.query}' : ''}');
      final handler = routes[key];
      if (handler == null) return _json({'error': 'no route $key'}, 404);
      return handler(req);
    }),
  ));
  addTearDown(manager.dispose);
  return (manager: manager, routes: routes, calls: calls, api: () => manager.plugin<SubscriptionApi>());
}

void main() {
  group('models', () {
    test('maps the server view into typed models', () async {
      final s = _setup();
      s.routes['GET /v1/billing/subscription'] = (_) => _json({
            'can_manage': false,
            'subscription': subJson({
              'status': 'trialing',
              'trial_end': DateTime.now().add(const Duration(days: 3)).toUtc().toIso8601String(),
              'cancel_at': '2026-12-01T00:00:00Z',
              'seats': {'feature_key': 'seats', 'used': 3, 'limit': null},
              'plan': planJson('team', {
                'features': [
                  {'key': 'api', 'name': 'API', 'type': 'metered', 'limit': null, 'soft_limit': true, 'period': 'monthly'},
                ],
              }),
            }),
          });
      await s.manager.initialize();
      await s.api().load();

      final snap = s.api().snapshot;
      final sub = snap.subscription.data!;
      expect(snap.subscription.status, ResourceStatus.ready);
      expect(snap.canManage, isFalse);
      expect(sub.plan.slug, 'team');
      expect(sub.plan.price!.amountMinor, 2900);
      expect(sub.plan.price!.interval, 'monthly');
      expect(sub.plan.features.single.limit, isNull);
      expect(sub.plan.features.single.softLimit, isTrue);
      expect(sub.seats!.used, 3);
      expect(sub.seats!.limit, isNull);
      expect(sub.status, SubscriptionStatus.trialing);
      expect(sub.isTrialing, isTrue);
      expect(sub.isActive, isTrue);
      expect(sub.willCancel, isTrue);
      expect(sub.cancelAt, DateTime.utc(2026, 12, 1));
      expect(sub.trialDaysLeft, 3);
    });

    test('keeps an unknown status from a newer server', () {
      expect(SubscriptionStatus.parse('frozen'), SubscriptionStatus.unknown);
      expect(SubscriptionStatus.parse('past_due'), SubscriptionStatus.pastDue);
    });

    test('formats money by the currency digits', () {
      expect(const Money(amountMinor: 2900, currency: 'usd').format(), r'$29.00');
      expect(const Money(amountMinor: 500, currency: 'jpy').format(), '¥500');
      expect(const Money(amountMinor: 1050, currency: 'ngn').format(), 'NGN 10.50');
    });
  });

  group('loading', () {
    test('loads the subscription, can_manage and entitlements', () async {
      final s = _setup();
      await s.manager.initialize();
      await s.api().load();
      expect(s.api().snapshot.subscription.data!.id, 'sub_1');
      expect(s.api().snapshot.canManage, isTrue);
      expect(s.api().snapshot.entitlements.status, ResourceStatus.ready);
      expect(s.api().entitlement('sso')!.allowed, isTrue);
      expect(s.api().entitlement('nope'), isNull);
    });

    test('is unavailable without asking when the server has no subscription plugin', () async {
      final s = _setup(serverPlugins: ['organization']);
      await s.manager.initialize();
      await s.api().load();
      expect(s.api().snapshot.subscription.status, ResourceStatus.unavailable);
      expect(s.calls.where((c) => c.contains('/v1/billing')), isEmpty);
    });

    test('drops a response for the org the session has since left', () async {
      final s = _setup();
      await s.manager.initialize();
      final gate = Completer<void>();
      s.routes['GET /v1/billing/subscription'] = (_) async {
        await gate.future;
        return _json({'subscription': subJson(), 'can_manage': true});
      };
      final loading = s.api().load();

      s.routes['POST /v1/me/switch-org'] = (_) => _json({'session_id': 's', 'org_id': 'org_2'});
      s.routes['GET /v1/billing/subscription'] = (_) => _json({'subscription': null, 'can_manage': true});
      await s.manager.switchOrg('org_2');
      gate.complete();
      await loading;
      await Future<void>.delayed(Duration.zero);
      expect(s.api().snapshot.subscription.status, ResourceStatus.ready);
      expect(s.api().snapshot.subscription.data, isNull);
    });

    test('resets on sign-out', () async {
      final s = _setup();
      await s.manager.initialize();
      await s.api().load();
      await s.manager.signOut();
      expect(s.api().snapshot.subscription.status, ResourceStatus.idle);
      expect(s.api().snapshot.subscription.data, isNull);
    });

    test('reads plans for the client config app, else the user app', () async {
      final withConfig = _setup(configAppId: 'app_cfg');
      withConfig.routes['GET /v1/billing/plans'] = (_) => _json({
            'plans': [planJson('free', {'amount_minor': null}), planJson('pro')],
          });
      await withConfig.manager.initialize();
      await withConfig.api().loadPlans();
      expect(withConfig.calls, contains('GET /v1/billing/plans?app_id=app_cfg'));
      expect(withConfig.api().snapshot.plans.data.map((p) => p.slug), ['free', 'pro']);
      expect(withConfig.api().snapshot.plans.data.first.price, isNull);

      final fromUser = _setup();
      fromUser.routes['GET /v1/billing/plans'] = (_) => _json({'plans': []});
      await fromUser.manager.initialize();
      await fromUser.api().loadPlans();
      expect(fromUser.calls, contains('GET /v1/billing/plans?app_id=app_user'));
    });
  });

  group('writes', () {
    test('writes a changed subscription into the cache', () async {
      final s = _setup();
      await s.manager.initialize();
      await s.api().load();
      s.routes['PATCH /v1/billing/subscription'] = (req) {
        expect(jsonDecode(req.body), {'plan': 'team', 'seats': 5});
        return _json({'subscription': subJson({'plan': planJson('team')})});
      };
      final result = await s.api().changePlan('team', seats: 5);
      expect(result, isA<SubscriptionChanged>());
      expect(s.api().snapshot.subscription.data!.plan.slug, 'team');
    });

    test('leaves the cache alone when payment comes first', () async {
      final s = _setup();
      await s.manager.initialize();
      await s.api().load();
      s.routes['POST /v1/billing/subscription'] = (_) => _json({'checkout_url': 'https://pay.example/1'});
      final result = await s.api().subscribe('business', returnUrl: 'https://app/billing');
      expect(result, isA<CheckoutRequired>());
      expect((result as CheckoutRequired).url, Uri.parse('https://pay.example/1'));
      expect(s.api().snapshot.subscription.data!.plan.slug, 'pro');
    });

    test('surfaces a refused write without touching the cache', () async {
      final s = _setup();
      await s.manager.initialize();
      await s.api().load();
      s.routes['POST /v1/billing/subscription/cancel'] = (_) => _json({'error': 'nope'}, 403);
      await expectLater(s.api().cancel(), throwsA(isA<AuthClientException>()));
      expect(s.api().snapshot.subscription.data!.cancelAt, isNull);
    });

    test('cancels at the period end by default and resumes', () async {
      final s = _setup();
      await s.manager.initialize();
      await s.api().load();
      s.routes['POST /v1/billing/subscription/cancel'] = (req) {
        expect(jsonDecode(req.body), {'when': 'period_end'});
        return _json({'subscription': subJson({'cancel_at': '2026-12-01T00:00:00Z'})});
      };
      s.routes['POST /v1/billing/subscription/resume'] = (_) => _json({'subscription': subJson()});
      expect((await s.api().cancel()).willCancel, isTrue);
      expect((await s.api().resume()).willCancel, isFalse);
    });

    test('emits a snapshot on every change', () async {
      final s = _setup();
      await s.manager.initialize();
      final seen = <SubscriptionSnapshot>[];
      s.api().changes.listen(seen.add);
      await s.api().load();
      expect(seen, isNotEmpty);
    });
  });
}
