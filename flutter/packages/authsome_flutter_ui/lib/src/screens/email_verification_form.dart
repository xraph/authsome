/// Email verification form screen.
///
/// Displays a 6-digit OTP input for email verification with a
/// resend button that has a 60-second cooldown timer.
library;

import 'dart:async';

import 'package:flutter/material.dart';
import 'package:authsome_flutter/authsome_flutter.dart';

import '../theme/auth_theme.dart';
import '../widgets/auth_card.dart';
import '../widgets/error_display.dart';
import '../widgets/otp_input.dart';
import '../widgets/loading_indicator.dart';

/// An email-verification form with a 6-digit [OtpInput] and resend button.
///
/// Calls [AuthSomeClient.verifyEmail] with the provided [email] and
/// the entered code. Includes a 60 s cooldown on the resend button.
class EmailVerificationForm extends StatefulWidget {
  /// The email address being verified.
  final String email;

  /// Called when verification succeeds.
  final VoidCallback? onSuccess;

  /// Called when the user taps "Resend code".
  ///
  /// If null, the resend button calls [AuthSomeClient.verifyEmail] with
  /// just the email to trigger a new code.
  final VoidCallback? onResend;

  /// Optional logo widget displayed above the title.
  final Widget? logo;

  // ── Localization overrides ──

  /// Card title (default: "Verify your email").
  final String titleText;

  /// Card description (default: "Enter the 6-digit code sent to {email}").
  final String? descriptionText;

  /// Resend button label (default: "Resend code").
  final String resendLabel;

  /// Success title (default: "Email verified").
  final String successTitleText;

  /// Success description (default: "Your email has been verified successfully").
  final String successDescriptionText;

  /// Cooldown duration in seconds (default: 60).
  final int cooldownSeconds;

  const EmailVerificationForm({
    required this.email,
    this.onSuccess,
    this.onResend,
    this.logo,
    this.titleText = 'Verify your email',
    this.descriptionText,
    this.resendLabel = 'Resend code',
    this.successTitleText = 'Email verified',
    this.successDescriptionText =
        'Your email has been verified successfully',
    this.cooldownSeconds = 60,
    super.key,
  });

  @override
  State<EmailVerificationForm> createState() => _EmailVerificationFormState();
}

class _EmailVerificationFormState extends State<EmailVerificationForm> {
  String? _error;
  bool _isSubmitting = false;
  bool _isSuccess = false;

  int _resendCooldown = 0;
  Timer? _resendTimer;

  @override
  void dispose() {
    _resendTimer?.cancel();
    super.dispose();
  }

  Future<void> _onCodeCompleted(String code) async {
    setState(() {
      _error = null;
      _isSubmitting = true;
    });

    try {
      final auth = context.auth;
      await auth.client.verifyEmail(body: {
        'email': widget.email,
        'code': code,
      });
      if (mounted) {
        setState(() {
          _isSuccess = true;
          _isSubmitting = false;
        });
        widget.onSuccess?.call();
      }
    } catch (e) {
      if (mounted) {
        setState(() {
          _error = e.toString();
          _isSubmitting = false;
        });
      }
    }
  }

  Future<void> _onResend() async {
    if (_resendCooldown > 0) return;

    if (widget.onResend != null) {
      widget.onResend!.call();
    } else {
      // Default: trigger a new verification code via the client.
      final auth = context.auth;
      try {
        await auth.client.verifyEmail(body: {'email': widget.email});
      } catch (e) {
        if (mounted) {
          setState(() => _error = e.toString());
        }
      }
    }

    _startCooldown();
  }

  void _startCooldown() {
    _resendTimer?.cancel();
    setState(() => _resendCooldown = widget.cooldownSeconds);
    _resendTimer = Timer.periodic(const Duration(seconds: 1), (timer) {
      if (!mounted) {
        timer.cancel();
        return;
      }
      setState(() {
        _resendCooldown--;
        if (_resendCooldown <= 0) {
          timer.cancel();
        }
      });
    });
  }

  @override
  Widget build(BuildContext context) {
    final theme = AuthTheme.of(context);
    final colorScheme = Theme.of(context).colorScheme;

    if (_isSuccess) {
      return _buildSuccessView(context, theme, colorScheme);
    }

    final description = widget.descriptionText ??
        'Enter the 6-digit code sent to ${widget.email}';

    return AuthCard(
      title: widget.titleText,
      description: description,
      logo: widget.logo,
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          ErrorDisplay(error: _error),
          if (_error != null) SizedBox(height: theme.fieldSpacing),
          OtpInput(
            onCompleted: _onCodeCompleted,
            enabled: !_isSubmitting,
            length: 6,
          ),
          SizedBox(height: theme.fieldSpacing),
          if (_isSubmitting)
            const Center(child: LoadingIndicator(size: LoadingSize.sm))
          else
            TextButton(
              onPressed: _resendCooldown > 0 ? null : _onResend,
              child: Text(
                _resendCooldown > 0
                    ? '${widget.resendLabel} (${_resendCooldown}s)'
                    : widget.resendLabel,
              ),
            ),
        ],
      ),
    );
  }

  Widget _buildSuccessView(
    BuildContext context,
    AuthThemeData theme,
    ColorScheme colorScheme,
  ) {
    return AuthCard(
      title: widget.successTitleText,
      description: widget.successDescriptionText,
      logo: widget.logo,
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(
            Icons.check_circle_outline,
            size: 64,
            color: colorScheme.primary,
          ),
        ],
      ),
    );
  }
}
