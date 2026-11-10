/// Flutter widgets for authsome's subscription plugin.
///
/// ```dart
/// AuthProvider(
///   config: AuthConfig(baseUrl: url, plugins: const [SubscriptionPlugin()]),
///   child: const MyApp(),
/// );
///
/// const CurrentPlanCard();
/// PricingTable(onCheckout: (uri) => launchUrl(uri));
/// const EntitlementGate(feature: 'sso', fallback: UpgradePrompt(), child: SsoSettings());
/// ```
library authsome_subscription_flutter;

export 'package:authsome_flutter/authsome_flutter.dart';
export 'package:authsome_subscription/authsome_subscription.dart';

export 'src/subscription_widgets.dart';
