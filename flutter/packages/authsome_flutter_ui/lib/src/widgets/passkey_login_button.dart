/// A standalone button that initiates a WebAuthn / passkey sign-in.
///
/// Mirrors React `passkey-login-button.tsx`. The button:
///   * Hides itself entirely when the configured
///     [PasskeyAuthenticator] reports `isAvailable == false`, so
///     server-side auto-config never surfaces a button on a platform
///     that can't honour it.
///   * On tap, calls [AuthNotifier.signInWithPasskey] which orchestrates
///     `/v1/passkeys/login/begin` → platform credential request →
///     `/v1/passkeys/login/finish`. The authenticated session then
///     flows through the standard [AuthState] transitions, so success
///     is observed via the surrounding [SignInForm]'s state listener.
library;

import 'package:flutter/material.dart';
import 'package:authsome_flutter/authsome_flutter.dart';

import 'error_display.dart';
import 'loading_indicator.dart';

class PasskeyLoginButton extends StatefulWidget {
  /// Optional injected notifier. Falls back to [AuthProvider.maybeOf]
  /// when null. Test seam.
  final AuthNotifier? auth;

  /// Authenticator that drives the platform-specific WebAuthn ceremony.
  /// Defaults to [defaultPasskeyAuthenticator] — Web-only at the moment.
  /// Native consumers can pass their own (e.g. backed by the `passkeys`
  /// corbado package).
  final PasskeyAuthenticator? authenticator;

  /// Optional email hint forwarded to `/passkeys/login/begin` for
  /// discoverable-credential UX.
  final String? email;

  /// Called when the passkey sign-in completes successfully. The
  /// surrounding form usually wires this to a navigation callback.
  final VoidCallback? onSuccess;

  /// Called when the passkey sign-in fails. Defaults to surfacing the
  /// error message inline.
  final ValueChanged<Object>? onError;

  /// Button label (default: "Continue with passkey").
  final String label;

  const PasskeyLoginButton({
    super.key,
    this.auth,
    this.authenticator,
    this.email,
    this.onSuccess,
    this.onError,
    this.label = 'Continue with passkey',
  });

  @override
  State<PasskeyLoginButton> createState() => _PasskeyLoginButtonState();
}

class _PasskeyLoginButtonState extends State<PasskeyLoginButton> {
  bool _isSubmitting = false;
  String? _error;
  late PasskeyAuthenticator _authenticator;

  @override
  void initState() {
    super.initState();
    _authenticator = widget.authenticator ?? defaultPasskeyAuthenticator();
  }

  @override
  void didUpdateWidget(covariant PasskeyLoginButton oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (widget.authenticator != null &&
        !identical(widget.authenticator, oldWidget.authenticator)) {
      _authenticator = widget.authenticator!;
    }
  }

  Future<void> _onTap() async {
    final auth = widget.auth ?? AuthProvider.maybeOf(context);
    if (auth == null) {
      setState(() => _error =
          'AuthProvider not found. Wrap the app in AuthProvider or pass '
          '`auth:` to PasskeyLoginButton.');
      return;
    }
    setState(() {
      _isSubmitting = true;
      _error = null;
    });
    try {
      await auth.signInWithPasskey(
        authenticator: _authenticator,
        email: widget.email,
      );
      widget.onSuccess?.call();
    } catch (e) {
      if (!mounted) return;
      setState(() {
        _error = e is AuthClientException ? e.message : e.toString();
      });
      widget.onError?.call(e);
    } finally {
      if (mounted) setState(() => _isSubmitting = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    if (!_authenticator.isAvailable) {
      // Platform can't run the WebAuthn ceremony — stay invisible so
      // the surrounding form doesn't render an OrDivider / spacing for
      // a button that would do nothing.
      return const SizedBox.shrink();
    }

    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      mainAxisSize: MainAxisSize.min,
      children: [
        OutlinedButton.icon(
          onPressed: _isSubmitting ? null : _onTap,
          icon: _isSubmitting
              ? const LoadingIndicator(size: LoadingSize.sm)
              : const Icon(Icons.fingerprint, size: 18),
          label: Text(widget.label),
        ),
        if (_error != null) ...[
          const SizedBox(height: 8),
          ErrorDisplay(error: _error),
        ],
      ],
    );
  }
}
