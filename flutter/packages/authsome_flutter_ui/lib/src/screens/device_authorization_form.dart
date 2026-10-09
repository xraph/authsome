/// OAuth device authorization screen (RFC 8628). Mirrors React
/// `device-authorization-form.tsx`.
library;

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:authsome_flutter/authsome_flutter.dart';

import '../platform/browser.dart';
import '../theme/auth_theme.dart';
import '../widgets/auth_card.dart';
import '../widgets/error_display.dart';
import '../widgets/loading_indicator.dart';

/// Lets a signed-in user approve the code shown on another device (a CLI,
/// a TV) so that device gets a session.
///
/// - The code comes from [initialCode], or on Flutter Web from the page's
///   `?user_code=` / `?code=` query parameter, and is submitted
///   automatically once the user is signed in (turn off with
///   [autoSubmit]).
/// - When `clientConfig.device_authorization.enabled` is false, the form
///   says device sign-in is unavailable instead of offering a form that
///   can only fail.
class DeviceAuthorizationForm extends StatefulWidget {
  /// Optional injected [AuthNotifier]. Test seam.
  final AuthNotifier? auth;

  /// Replaces the built-in approval call. Receives the cleaned code.
  final Future<void> Function(String code)? onSubmit;

  /// Called after the device is authorized.
  final VoidCallback? onSuccess;

  /// Called when authorization fails.
  final ValueChanged<Object>? onError;

  /// Called from the "Sign in" button shown to a signed-out user.
  final VoidCallback? onSignInTap;

  /// Number of characters in the user code (default: 8).
  final int codeLength;

  /// Pre-filled code. When null, Flutter Web reads it from the URL.
  final String? initialCode;

  /// Submit a pre-filled code without waiting for a tap (default: true).
  final bool autoSubmit;

  /// Optional logo widget displayed above the title.
  final Widget? logo;

  /// Title + description alignment within the [AuthCard].
  final AuthCardAlign align;

  const DeviceAuthorizationForm({
    this.auth,
    this.onSubmit,
    this.onSuccess,
    this.onError,
    this.onSignInTap,
    this.codeLength = 8,
    this.initialCode,
    this.autoSubmit = true,
    this.logo,
    this.align = AuthCardAlign.center,
    super.key,
  });

  @override
  State<DeviceAuthorizationForm> createState() =>
      _DeviceAuthorizationFormState();
}

/// Strips everything but letters and digits and uppercases the rest, so
/// `abcd-efgh` and `ABCDEFGH` are the same code.
String _cleanCode(String raw) =>
    raw.replaceAll(RegExp(r'[^A-Za-z0-9]'), '').toUpperCase();

class _DeviceAuthorizationFormState extends State<DeviceAuthorizationForm> {
  final _codeController = TextEditingController();

  String? _error;
  bool _isSubmitting = false;
  bool _isSuccess = false;
  bool _autoSubmitted = false;

  AuthNotifier? _auth;
  bool _missingProvider = false;

  @override
  void initState() {
    super.initState();
    final seed = widget.initialCode ??
        currentQueryParameters()['user_code'] ??
        currentQueryParameters()['code'];
    if (seed != null) _codeController.text = _cleanCode(seed);
  }

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
      _auth!.addListener(_onAuthChanged);
      WidgetsBinding.instance.addPostFrameCallback((_) => _maybeAutoSubmit());
    }
  }

  @override
  void dispose() {
    _auth?.removeListener(_onAuthChanged);
    _codeController.dispose();
    super.dispose();
  }

  void _onAuthChanged() {
    if (!mounted) return;
    setState(() {});
    _maybeAutoSubmit();
  }

  bool get _isEnabled =>
      _auth?.clientConfig?.deviceAuthorization?.enabled != false;

  bool get _isSignedIn => _auth?.state is AuthAuthenticated;

  /// Submits a pre-filled code once the user is signed in, the way React
  /// waits for the session to load before auto-submitting.
  void _maybeAutoSubmit() {
    if (!widget.autoSubmit ||
        _autoSubmitted ||
        _isSubmitting ||
        _isSuccess ||
        !_isEnabled) {
      return;
    }
    final code = _cleanCode(_codeController.text);
    if (code.length != widget.codeLength) return;
    // The built-in call needs a session; a custom onSubmit decides itself.
    if (widget.onSubmit == null && !_isSignedIn) return;
    _autoSubmitted = true;
    _submit();
  }

  Future<void> _submit() async {
    final code = _cleanCode(_codeController.text);
    if (code.length != widget.codeLength) {
      setState(() =>
          _error = 'Enter the ${widget.codeLength}-character code from your device');
      return;
    }
    if (_isSubmitting) return;

    setState(() {
      _error = null;
      _isSubmitting = true;
    });

    try {
      if (widget.onSubmit != null) {
        await widget.onSubmit!(code);
      } else {
        await _auth!.approveDeviceAuthorization(code);
      }
      if (!mounted) return;
      setState(() {
        _isSuccess = true;
        _isSubmitting = false;
      });
      widget.onSuccess?.call();
    } catch (e) {
      if (!mounted) return;
      setState(() {
        _error = e is AuthClientException
            ? e.message
            : 'Failed to authorize device';
        _isSubmitting = false;
        _codeController.clear();
      });
      widget.onError?.call(e);
    }
  }

  @override
  Widget build(BuildContext context) {
    if (_missingProvider) {
      return AuthCard(
        title: 'Authorize device',
        logo: widget.logo,
        align: widget.align,
        child: const ErrorDisplay(
          error: 'AuthProvider not found in widget tree. Wrap your app in '
              'AuthProvider, or pass an `auth:` notifier to '
              'DeviceAuthorizationForm.',
        ),
      );
    }

    final theme = AuthTheme.of(context);
    final colorScheme = Theme.of(context).colorScheme;
    final branding = _auth?.clientConfig?.branding;

    if (!_isEnabled) {
      return AuthCard(
        title: 'Device sign-in unavailable',
        description: 'Device authorization is not enabled for this app.',
        logo: widget.logo,
        branding: branding,
        align: widget.align,
        child: const SizedBox.shrink(),
      );
    }

    if (_isSuccess) {
      return AuthCard(
        title: 'Device authorized',
        description: 'Your device has been successfully authorized. '
            'You can return to it now.',
        logo: widget.logo,
        branding: branding,
        align: widget.align,
        child: Icon(
          Icons.check_circle_outline,
          size: 56,
          color: colorScheme.primary,
        ),
      );
    }

    final needsSignIn = widget.onSubmit == null && !_isSignedIn;

    return AuthCard(
      title: 'Authorize device',
      description: 'Enter the code shown on your device.',
      logo: widget.logo,
      branding: branding,
      align: widget.align,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        mainAxisSize: MainAxisSize.min,
        children: [
          ErrorDisplay(error: _error),
          if (_error != null) SizedBox(height: theme.fieldSpacing),
          TextField(
            controller: _codeController,
            enabled: !_isSubmitting,
            textAlign: TextAlign.center,
            textCapitalization: TextCapitalization.characters,
            style: Theme.of(context).textTheme.titleLarge?.copyWith(
                  letterSpacing: 4,
                ),
            inputFormatters: [
              FilteringTextInputFormatter.allow(RegExp(r'[A-Za-z0-9]')),
              LengthLimitingTextInputFormatter(widget.codeLength),
              TextInputFormatter.withFunction(
                (_, next) => next.copyWith(text: next.text.toUpperCase()),
              ),
            ],
            onChanged: (_) {
              if (_error != null) setState(() => _error = null);
            },
            onSubmitted: needsSignIn ? null : (_) => _submit(),
            decoration: const InputDecoration(
              labelText: 'Device code',
              border: OutlineInputBorder(),
            ),
          ),
          SizedBox(height: theme.fieldSpacing),
          if (needsSignIn) ...[
            Text(
              'Sign in to authorize this device.',
              textAlign: TextAlign.center,
              style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                    color: colorScheme.onSurfaceVariant,
                  ),
            ),
            if (widget.onSignInTap != null) ...[
              SizedBox(height: theme.fieldSpacing),
              FilledButton(
                onPressed: widget.onSignInTap,
                child: const Text('Sign in'),
              ),
            ],
          ] else
            FilledButton(
              onPressed: _isSubmitting ? null : _submit,
              child: _isSubmitting
                  ? const LoadingIndicator(size: LoadingSize.sm)
                  : const Text('Authorize'),
            ),
        ],
      ),
    );
  }
}
