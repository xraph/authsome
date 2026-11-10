import 'dart:async';

import 'package:authsome_core/authsome_core.dart';

import 'models.dart';

/// Where a piece of server state is. [unavailable] means the server doesn't
/// have the subscription plugin installed; treat it like a plan without
/// the feature.
enum ResourceStatus { idle, loading, ready, error, unavailable }

/// One piece of server state.
class Resource<T> {
  const Resource(this.status, this.data, [this.error]);

  final ResourceStatus status;
  final T data;
  final Object? error;

  bool get isLoading => status == ResourceStatus.loading;
}

/// Everything the plugin knows, as one immutable snapshot.
class SubscriptionSnapshot {
  const SubscriptionSnapshot({
    required this.subscription,
    required this.canManage,
    required this.plans,
    required this.entitlements,
  });

  SubscriptionSnapshot.initial()
      : subscription = const Resource(ResourceStatus.idle, null),
        canManage = false,
        plans = const Resource(ResourceStatus.idle, <Plan>[]),
        entitlements = const Resource(ResourceStatus.idle, <String, Entitlement>{});

  /// The tenant's subscription; data is null when it has none.
  final Resource<Subscription?> subscription;

  /// Whether the signed-in user may change the subscription.
  final bool canManage;

  /// Plans the tenant can buy, cheapest first.
  final Resource<List<Plan>> plans;

  /// Every feature on the current plan, checked.
  final Resource<Map<String, Entitlement>> entitlements;

  SubscriptionSnapshot copyWith({
    Resource<Subscription?>? subscription,
    bool? canManage,
    Resource<List<Plan>>? plans,
    Resource<Map<String, Entitlement>>? entitlements,
  }) =>
      SubscriptionSnapshot(
        subscription: subscription ?? this.subscription,
        canManage: canManage ?? this.canManage,
        plans: plans ?? this.plans,
        entitlements: entitlements ?? this.entitlements,
      );
}

/// What a plan or seat change did.
sealed class ChangeResult {
  const ChangeResult();
}

/// The change happened; here's the subscription now.
class SubscriptionChanged extends ChangeResult {
  const SubscriptionChanged(this.subscription);
  final Subscription subscription;
}

/// Payment comes first: send the customer to [url]. Nothing changed yet.
class CheckoutRequired extends ChangeResult {
  const CheckoutRequired(this.url);
  final Uri url;
}

/// When a cancellation takes effect.
enum CancelWhen { periodEnd, now }

/// The subscription plugin's API, from `auth.plugin<SubscriptionApi>()` or,
/// in Flutter, `context.subscription`.
class SubscriptionApi {
  SubscriptionApi(this._ctx) {
    // Follow the session: a sign-out forgets everything, and a sign-in or
    // an organization switch reloads whatever had been loaded.
    _events = _ctx.events.listen(
      (event) {
        switch (event) {
          case AuthEvent.signedOut:
            _generation++;
            _set(SubscriptionSnapshot.initial());
          case AuthEvent.signedIn || AuthEvent.orgChanged:
            _reload();
          case AuthEvent.sessionRefreshed:
            break;
        }
      },
      // The manager closes its events when it's disposed; nothing will
      // change after that, so neither does this stream.
      onDone: () => unawaited(_changes.close()),
    );
  }

  final PluginContext _ctx;
  late final StreamSubscription<AuthEvent> _events;
  final _changes = StreamController<SubscriptionSnapshot>.broadcast();
  SubscriptionSnapshot _snapshot = SubscriptionSnapshot.initial();

  // Bumped whenever the session or organization changes, so a response for
  // the one the user just left is dropped instead of shown.
  int _generation = 0;

  // The load in flight for the current generation, shared by every caller.
  ({int gen, Future<void> future})? _loading;

  /// The current snapshot.
  SubscriptionSnapshot get snapshot => _snapshot;

  /// Every new snapshot.
  Stream<SubscriptionSnapshot> get changes => _changes.stream;

  void _set(SubscriptionSnapshot next) {
    _snapshot = next;
    if (!_changes.isClosed) _changes.add(next);
  }

  bool get _unavailable => _ctx.isServerEnabled == false;
  bool get _signedIn => _ctx.state is AuthAuthenticated;

  // A new session or organization: whatever was shown belongs to the old
  // one, so clear it before anyone can act on it, then load again.
  void _reload() {
    _generation++;
    if (_snapshot.subscription.status == ResourceStatus.idle) return;
    _set(_snapshot.copyWith(
      subscription: const Resource(ResourceStatus.loading, null),
      entitlements: const Resource(ResourceStatus.loading, <String, Entitlement>{}),
      canManage: false,
    ));
    unawaited(load());
  }

  /// Loads the subscription and its entitlements. Calls made while a load
  /// is running share it.
  Future<void> load() {
    final current = _loading;
    if (current != null && current.gen == _generation) return current.future;
    final gen = _generation;
    final future = _load(gen).whenComplete(() {
      if (_loading?.gen == gen) _loading = null;
    });
    _loading = (gen: gen, future: future);
    return future;
  }

  Future<void> _load(int gen) async {
    if (_unavailable) {
      _set(_snapshot.copyWith(
        subscription: const Resource(ResourceStatus.unavailable, null),
        entitlements: const Resource(ResourceStatus.unavailable, <String, Entitlement>{}),
      ));
      return;
    }
    if (!_signedIn) return;
    _set(_snapshot.copyWith(
      subscription: Resource(ResourceStatus.loading, _snapshot.subscription.data),
    ));
    final entitlements = _loadEntitlements(gen);
    try {
      final res = await _ctx.request('GET', '/v1/billing/subscription') as Map;
      if (gen != _generation) return;
      final sub = res['subscription'] as Map?;
      _set(_snapshot.copyWith(
        subscription: Resource(
          ResourceStatus.ready,
          sub == null ? null : Subscription.fromJson(Map<String, dynamic>.from(sub)),
        ),
        canManage: res['can_manage'] as bool? ?? false,
      ));
    } catch (error) {
      if (gen != _generation) return;
      _set(_snapshot.copyWith(
        subscription: Resource(ResourceStatus.error, _snapshot.subscription.data, error),
      ));
    }
    await entitlements;
  }

  Future<void> _loadEntitlements(int gen) async {
    if (gen != _generation) return;
    _set(_snapshot.copyWith(
      entitlements: Resource(ResourceStatus.loading, _snapshot.entitlements.data),
    ));
    try {
      final res = await _ctx.request('GET', '/v1/billing/entitlements') as Map;
      if (gen != _generation) return;
      final raw = Map<String, dynamic>.from(res['entitlements'] as Map? ?? const {});
      _set(_snapshot.copyWith(
        entitlements: Resource(ResourceStatus.ready, {
          for (final e in raw.entries)
            e.key: Entitlement.fromJson(Map<String, dynamic>.from(e.value as Map)),
        }),
      ));
    } catch (error) {
      if (gen != _generation) return;
      _set(_snapshot.copyWith(
        entitlements: Resource(ResourceStatus.error, _snapshot.entitlements.data, error),
      ));
    }
  }

  /// Loads the plans the tenant can buy.
  Future<void> loadPlans() async {
    if (_unavailable) {
      _set(_snapshot.copyWith(plans: const Resource(ResourceStatus.unavailable, <Plan>[])));
      return;
    }
    // /plans is app-wide and wants the app's ID: the client config carries
    // it, and so does the signed-in user.
    final state = _ctx.state;
    final user = state is AuthAuthenticated ? state.user : null;
    final appId = _ctx.clientConfig?.appId ?? (user is User ? user.appId : null);
    final gen = _generation;
    _set(_snapshot.copyWith(plans: Resource(ResourceStatus.loading, _snapshot.plans.data)));
    try {
      final res = await _ctx.request('GET', '/v1/billing/plans', query: {'app_id': appId}) as Map;
      if (gen != _generation) return;
      _set(_snapshot.copyWith(
        plans: Resource(ResourceStatus.ready, [
          for (final p in (res['plans'] as List? ?? const []))
            Plan.fromJson(Map<String, dynamic>.from(p as Map)),
        ]),
      ));
    } catch (error) {
      if (gen != _generation) return;
      _set(_snapshot.copyWith(plans: Resource(ResourceStatus.error, _snapshot.plans.data, error)));
    }
  }

  /// Starts a subscription on a plan, by slug or ID.
  Future<ChangeResult> subscribe(String plan, {int? seats, String? returnUrl, String? cancelUrl}) =>
      _change('POST', _body({'plan': plan}, seats, returnUrl, cancelUrl));

  /// Moves the subscription to another plan, by slug or ID.
  Future<ChangeResult> changePlan(String plan, {int? seats, String? returnUrl, String? cancelUrl}) =>
      _change('PATCH', _body({'plan': plan}, seats, returnUrl, cancelUrl));

  /// Changes the seat count on the current plan.
  Future<ChangeResult> setSeats(int seats, {String? returnUrl, String? cancelUrl}) =>
      _change('PATCH', _body({}, seats, returnUrl, cancelUrl));

  /// Cancels at the end of the period (the default) or now.
  Future<Subscription> cancel({CancelWhen when = CancelWhen.periodEnd}) => _mutate(
        '/v1/billing/subscription/cancel',
        {'when': when == CancelWhen.now ? 'now' : 'period_end'},
      );

  /// Takes back a cancellation scheduled for the end of the period.
  Future<Subscription> resume() => _mutate('/v1/billing/subscription/resume', {});

  /// A link to the payment provider's billing portal.
  Future<Uri> openPortal({String? returnUrl}) async {
    final res = await _ctx.request('POST', '/v1/billing/portal',
        body: {if (returnUrl != null) 'return_url': returnUrl}) as Map;
    return Uri.parse(res['url'] as String);
  }

  /// The cached check for a feature, null when it isn't on the plan or not loaded.
  Entitlement? entitlement(String feature) => _snapshot.entitlements.data[feature];

  Map<String, Object?> _body(Map<String, Object?> base, int? seats, String? returnUrl, String? cancelUrl) => {
        ...base,
        if (seats != null) 'seats': seats,
        if (returnUrl != null) 'return_url': returnUrl,
        if (cancelUrl != null) 'cancel_url': cancelUrl,
      };

  Future<ChangeResult> _change(String method, Map<String, Object?> body) async {
    final gen = _generation;
    final res = await _ctx.request(method, '/v1/billing/subscription', body: body) as Map;
    final url = res['checkout_url'] as String?;
    if (url != null && url.isNotEmpty) return CheckoutRequired(Uri.parse(url));
    return SubscriptionChanged(_commit(res, gen));
  }

  Future<Subscription> _mutate(String path, Map<String, Object?> body) async {
    final gen = _generation;
    final res = await _ctx.request('POST', path, body: body) as Map;
    return _commit(res, gen);
  }

  // Writes answer with the new subscription. It goes straight into the
  // cache, and the entitlements are rechecked since the plan may have moved.
  // A write that lands after the user switched org or signed out still
  // returns its result, but the cache now belongs to someone else.
  Subscription _commit(Map res, int gen) {
    final json = res['subscription'] as Map?;
    if (json == null) {
      throw StateError('authsome: the server answered without a subscription');
    }
    final sub = Subscription.fromJson(Map<String, dynamic>.from(json));
    if (gen == _generation) {
      _set(_snapshot.copyWith(subscription: Resource(ResourceStatus.ready, sub)));
      unawaited(_loadEntitlements(gen));
    }
    return sub;
  }

  /// Stops following the session. The plugin's manager calls nothing on
  /// dispose today, so this is for tests and long-lived isolates.
  Future<void> close() async {
    await _events.cancel();
    if (!_changes.isClosed) await _changes.close();
  }
}

/// The subscription client plugin. Register it in [AuthConfig.plugins] and
/// read it with `auth.plugin<SubscriptionApi>()`.
class SubscriptionPlugin extends AuthPlugin<SubscriptionApi> {
  const SubscriptionPlugin();

  @override
  String get name => 'subscription';

  @override
  SubscriptionApi init(PluginContext ctx) => SubscriptionApi(ctx);
}
