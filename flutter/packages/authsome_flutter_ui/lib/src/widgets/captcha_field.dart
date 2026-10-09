/// Captcha challenge driven by the client config's `captcha` section.
library;

import 'package:flutter/material.dart';
import 'package:authsome_flutter/authsome_flutter.dart';

import '../platform/turnstile.dart';
import 'error_display.dart';

/// Builds a captcha challenge for [config] and reports the token through
/// [onToken] (null when a token expires or the challenge fails).
///
/// Pass one to [SignInForm] / [SignUpForm] to show a captcha the built-in
/// widget can't, such as Turnstile inside a WebView on iOS or Android.
typedef CaptchaBuilder = Widget Function(
  BuildContext context,
  CaptchaConfig config,
  ValueChanged<String?> onToken,
);

/// Renders the captcha the backend asks for.
///
/// Uses [builder] when given. Otherwise renders Cloudflare Turnstile on
/// Flutter Web, the only provider and platform the built-in widget covers
/// (React's `TurnstileWidget` has the same scope). Anywhere else it shows
/// an error instead of an empty gap, since the server rejects the request
/// without a token anyway.
class CaptchaField extends StatelessWidget {
  /// The captcha section of the client config.
  final CaptchaConfig config;

  /// Receives the current token, or null once it is no longer valid.
  final ValueChanged<String?> onToken;

  /// Optional override for platforms or providers the built-in widget
  /// doesn't cover.
  final CaptchaBuilder? builder;

  const CaptchaField({
    required this.config,
    required this.onToken,
    this.builder,
    super.key,
  });

  /// Whether [config] asks the form to collect a token before submitting.
  static bool isRequired(CaptchaConfig? config) =>
      config != null && config.required;

  @override
  Widget build(BuildContext context) {
    if (builder != null) return builder!(context, config, onToken);
    if (config.provider == 'turnstile' &&
        turnstileSupported &&
        (config.siteKey?.isNotEmpty ?? false)) {
      return Center(
        child: TurnstileView(
          siteKey: config.siteKey!,
          onToken: onToken,
          onExpired: () => onToken(null),
          onError: () => onToken(null),
        ),
      );
    }
    return const ErrorDisplay(
      error: 'Verification is unavailable. Please try again later.',
    );
  }
}
