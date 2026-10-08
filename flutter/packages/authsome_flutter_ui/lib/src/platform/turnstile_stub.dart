/// Native stand-in for the Turnstile widget. Turnstile is a browser script,
/// so off the Web an app has to supply its own captcha builder (for
/// example one backed by a WebView).
library;

import 'package:flutter/widgets.dart';

/// Whether the built-in Turnstile widget can render here.
bool get turnstileSupported => false;

/// Never built off the Web: callers check [turnstileSupported] first.
class TurnstileView extends StatelessWidget {
  final String siteKey;
  final ValueChanged<String> onToken;
  final VoidCallback onExpired;
  final VoidCallback onError;

  const TurnstileView({
    required this.siteKey,
    required this.onToken,
    required this.onExpired,
    required this.onError,
    super.key,
  });

  @override
  Widget build(BuildContext context) => const SizedBox.shrink();
}
