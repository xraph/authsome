# authsome_subscription_flutter

Material 3 widgets for authsome's `subscription` plugin: a pricing table, the current plan, usage, an entitlement gate and a billing portal button.

```yaml
dependencies:
  authsome_flutter: ^1.6.1
  authsome_subscription_flutter: ^1.6.1
```

## Set it up

```dart
AuthProvider(
  config: AuthConfig(
    baseUrl: 'https://api.example.com',
    plugins: const [SubscriptionPlugin()],
  ),
  child: const MyApp(),
);
```

## Widgets

```dart
const CurrentPlanCard();

PricingTable(
  highlight: 'team',
  onCheckout: (uri) => launchUrl(uri),
);

const EntitlementGate(
  feature: 'sso',
  fallback: UpgradePrompt(),
  child: SsoSettings(),
);

const UsageMeter(feature: 'api_calls', label: 'API calls this month');

ManageBillingButton(onOpen: (uri) => launchUrl(uri));
```

`PricingTable` and `CurrentPlanCard` show plain words when a change fails, such as "Couldn't change your plan. Try again.", and pass the server's error to `onError` if you want to log it. Leave out `onCheckout` and a paid plan can't be bought: the table says so and hands `onError` a `StateError`.

`EntitlementGate` only shows its child when the plan includes the feature. While the check loads, on an error, or when the server doesn't have the plugin, you get the fallback.

Opening a payment or portal page is left to you, so the package doesn't pull in a URL launcher. Pass `launchUrl` from url_launcher, or whatever you already use.

For your own UI, `context.subscription` is the plugin's API and `SubscriptionBuilder` rebuilds from its snapshot:

```dart
SubscriptionBuilder(
  builder: (context, snap) {
    final sub = snap.subscription.data;
    if (snap.subscription.isLoading) return const CircularProgressIndicator();
    return Text(sub?.plan.name ?? 'Free');
  },
);
```
