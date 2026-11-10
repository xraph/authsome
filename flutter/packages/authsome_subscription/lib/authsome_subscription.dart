/// The client half of authsome's subscription plugin.
///
/// ```dart
/// final auth = AuthManager(AuthConfig(
///   baseUrl: 'https://api.example.com',
///   plugins: [SubscriptionPlugin()],
/// ));
/// final billing = auth.plugin<SubscriptionApi>();
/// await billing.load();
/// ```
///
/// Flutter widgets are in `authsome_subscription_flutter`.
library authsome_subscription;

export 'src/models.dart';
export 'src/subscription_api.dart';
