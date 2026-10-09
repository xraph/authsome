/// SSO landing screen. Mirrors React `sso-callback.tsx`.
library;

import 'package:flutter/material.dart';
import 'package:authsome_flutter/authsome_flutter.dart';

import '../platform/browser.dart';
import '../widgets/auth_card.dart';
import '../widgets/error_display.dart';
import '../widgets/loading_indicator.dart';

/// Finishes an SSO sign-in on the page the identity provider sends the
/// user back to.
///
/// The backend appends `?code=` on success and `?sso_error=` on failure.
/// Mount this at that return URL (the server default is `/sso/callback`).
/// On Flutter Web it reads both from the page URL. Elsewhere, or with a
/// router that already parsed them, pass [code] and [error].
///
/// The code is exchanged exactly once, then [onSuccess] runs.
class SSOCallbackForm extends StatefulWidget {
  /// Optional injected [AuthNotifier]. Test seam.
  final AuthNotifier? auth;

  /// The one-time code. When null, read from the page URL on Web.
  final String? code;

  /// The failure reason. When null, read from `?sso_error=` on Web.
  final String? error;

  /// Called once the session is established.
  final VoidCallback? onSuccess;

  /// Called when the sign-in fails.
  final ValueChanged<Object>? onError;

  /// Shows a "Back to sign in" button on the failure card.
  final VoidCallback? onSignInTap;

  /// Optional logo widget displayed above the title.
  final Widget? logo;

  /// Title + description alignment within the [AuthCard].
  final AuthCardAlign align;

  const SSOCallbackForm({
    this.auth,
    this.code,
    this.error,
    this.onSuccess,
    this.onError,
    this.onSignInTap,
    this.logo,
    this.align = AuthCardAlign.center,
    super.key,
  });

  @override
  State<SSOCallbackForm> createState() => _SSOCallbackFormState();
}

class _SSOCallbackFormState extends State<SSOCallbackForm> {
  AuthNotifier? _auth;
  bool _missingProvider = false;
  bool _started = false;
  bool _done = false;
  String? _failure;

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    if (_auth == null && !_missingProvider) {
      final injected = widget.auth ?? AuthProvider.maybeOf(context);
      if (injected == null) {
        _missingProvider = true;
        return;
      }
      _auth = injected;
      // completeSSOLogin notifies the AuthProvider, which must not happen
      // while this frame is still building.
      WidgetsBinding.instance.addPostFrameCallback((_) => _start());
    }
  }

  void _start() {
    if (_started) return;
    _started = true;

    final query = currentQueryParameters();
    final error = widget.error ?? query['sso_error'];
    final code = widget.code ?? query['code'];

    if (!mounted) return;
    if (error != null && error.isNotEmpty) {
      setState(() => _failure = error);
      widget.onError?.call(error);
      return;
    }
    if (code == null || code.isEmpty) {
      const missing = 'The sign-in response had no code.';
      setState(() => _failure = missing);
      widget.onError?.call(missing);
      return;
    }
    _exchange(code);
  }

  Future<void> _exchange(String code) async {
    try {
      await _auth!.completeSSOLogin(code);
      if (!mounted) return;
      setState(() => _done = true);
      widget.onSuccess?.call();
    } catch (e) {
      if (!mounted) return;
      setState(() {
        _failure = e is AuthClientException ? e.message : e.toString();
      });
      widget.onError?.call(e);
    }
  }

  @override
  Widget build(BuildContext context) {
    if (_missingProvider) {
      return AuthCard(
        title: 'Single sign-on',
        logo: widget.logo,
        align: widget.align,
        child: const ErrorDisplay(
          error: 'AuthProvider not found in widget tree. Wrap your app in '
              'AuthProvider, or pass an `auth:` notifier to SSOCallbackForm.',
        ),
      );
    }

    final branding = _auth?.clientConfig?.branding;

    if (_failure != null) {
      return AuthCard(
        title: 'Single sign-on failed',
        logo: widget.logo,
        branding: branding,
        align: widget.align,
        footer: widget.onSignInTap == null
            ? null
            : Center(
                child: TextButton(
                  onPressed: widget.onSignInTap,
                  child: const Text('Back to sign in'),
                ),
              ),
        child: ErrorDisplay(error: _failure),
      );
    }

    if (_done) {
      return AuthCard(
        title: "You're signed in",
        logo: widget.logo,
        branding: branding,
        align: widget.align,
        child: Icon(
          Icons.check_circle_outline,
          size: 56,
          color: Theme.of(context).colorScheme.primary,
        ),
      );
    }

    return AuthCard(
      title: 'Signing you in',
      logo: widget.logo,
      branding: branding,
      align: widget.align,
      child: const Center(child: LoadingIndicator()),
    );
  }
}
