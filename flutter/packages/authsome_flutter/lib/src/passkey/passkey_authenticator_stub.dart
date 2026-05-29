/// Default authenticator used on platforms that don't have a native
/// passkey implementation wired up yet (desktop, native iOS/Android
/// when the consumer hasn't supplied a custom authenticator).
library;

import 'package:authsome_core/authsome_core.dart';

/// Re-exported under a stable name so the conditional-import factory
/// can pick the right implementation per platform.
class PlatformPasskeyAuthenticator extends UnsupportedPasskeyAuthenticator {
  const PlatformPasskeyAuthenticator();
}
