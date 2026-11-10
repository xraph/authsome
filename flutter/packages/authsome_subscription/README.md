# authsome_subscription

The client half of authsome's `subscription` plugin, for Dart and Flutter. You get the tenant's subscription, its plan, seats and feature checks as typed models, and a stream of changes. For widgets, add [`authsome_subscription_flutter`](https://pub.dev/packages/authsome_subscription_flutter).

```yaml
dependencies:
  authsome_core: ^1.6.1
  authsome_subscription: ^1.6.1
```

The server needs the plugin too: `authsome.WithPlugin(subscription.New(...))`.

## Usage

```dart
final auth = AuthManager(AuthConfig(
  baseUrl: 'https://api.example.com',
  plugins: const [SubscriptionPlugin()],
));
await auth.initialize();

final billing = auth.plugin<SubscriptionApi>();
await billing.load();

final sub = billing.snapshot.subscription.data;
if (sub != null && sub.willCancel) {
  print('${sub.plan.name} ends on ${sub.cancelAt}');
}
```

Writes update the snapshot, so anything listening to `billing.changes` sees the new state:

```dart
final result = await billing.changePlan('team');
switch (result) {
  case SubscriptionChanged(:final subscription):
    print('now on ${subscription.plan.name}');
  case CheckoutRequired(:final url):
    await launchUrl(url); // payment first; nothing changed yet
}
```

`cancel()` cancels at the end of the period unless you pass `when: CancelWhen.now`, and `resume()` takes that back. `billing.entitlement('sso')` reads the cached check for a feature.

The snapshot follows the session: signing out clears it, and signing in or switching organization reloads it. When the server doesn't have the subscription plugin, the resources come back `ResourceStatus.unavailable` without a request.
