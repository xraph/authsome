/// Forgot-password form screen.
///
/// Single-step form with email input. On success, displays a confirmation
/// message with a mail icon prompting the user to check their inbox.
library;

import 'package:flutter/material.dart';
import 'package:authsome_flutter/authsome_flutter.dart';

import '../theme/auth_theme.dart';
import '../widgets/auth_card.dart';
import '../widgets/error_display.dart';
import '../widgets/loading_indicator.dart';

/// A forgot-password form that sends a reset link to the provided email.
///
/// Calls [AuthSomeClient.forgotPassword] and shows a confirmation view
/// on success.
class ForgotPasswordForm extends StatefulWidget {
  /// Called when the reset email is sent successfully.
  final VoidCallback? onSuccess;

  /// Called when the user taps the "Back to sign in" link.
  final VoidCallback? onSignInTap;

  /// Optional logo widget displayed above the title.
  final Widget? logo;

  // ── Localization overrides ──

  /// Card title (default: "Forgot password?").
  final String titleText;

  /// Card description (default: "Enter your email and we'll send you a reset link").
  final String descriptionText;

  /// Email field label (default: "Email").
  final String emailLabel;

  /// Submit button label (default: "Send reset link").
  final String submitLabel;

  /// Sign-in link label (default: "Back to sign in").
  final String signInLabel;

  /// Success title (default: "Check your email").
  final String successTitleText;

  /// Success description (default: "We've sent a password reset link to {email}").
  final String? successDescriptionText;

  const ForgotPasswordForm({
    this.onSuccess,
    this.onSignInTap,
    this.logo,
    this.titleText = 'Forgot password?',
    this.descriptionText = "Enter your email and we'll send you a reset link",
    this.emailLabel = 'Email',
    this.submitLabel = 'Send reset link',
    this.signInLabel = 'Back to sign in',
    this.successTitleText = 'Check your email',
    this.successDescriptionText,
    super.key,
  });

  @override
  State<ForgotPasswordForm> createState() => _ForgotPasswordFormState();
}

class _ForgotPasswordFormState extends State<ForgotPasswordForm> {
  final _emailController = TextEditingController();

  String? _error;
  bool _isSubmitting = false;
  bool _isSent = false;

  @override
  void dispose() {
    _emailController.dispose();
    super.dispose();
  }

  Future<void> _onSubmit() async {
    final email = _emailController.text.trim();
    if (email.isEmpty) {
      setState(() => _error = 'Please enter your email');
      return;
    }

    setState(() {
      _error = null;
      _isSubmitting = true;
    });

    try {
      final auth = context.auth;
      await auth.client.forgotPassword(body: {'email': email});
      if (mounted) {
        setState(() {
          _isSent = true;
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

  @override
  Widget build(BuildContext context) {
    final theme = AuthTheme.of(context);
    final colorScheme = Theme.of(context).colorScheme;
    final textTheme = Theme.of(context).textTheme;

    if (_isSent) {
      return _buildSuccessView(context, theme, colorScheme, textTheme);
    }

    return AuthCard(
      title: widget.titleText,
      description: widget.descriptionText,
      logo: widget.logo,
      footer: _buildFooter(context),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        mainAxisSize: MainAxisSize.min,
        children: [
          ErrorDisplay(error: _error),
          if (_error != null) SizedBox(height: theme.fieldSpacing),
          TextField(
            controller: _emailController,
            enabled: !_isSubmitting,
            keyboardType: TextInputType.emailAddress,
            textInputAction: TextInputAction.done,
            onSubmitted: (_) => _onSubmit(),
            decoration: InputDecoration(
              labelText: widget.emailLabel,
              hintText: 'you@example.com',
              border: const OutlineInputBorder(),
            ),
          ),
          SizedBox(height: theme.fieldSpacing),
          FilledButton(
            onPressed: _isSubmitting ? null : _onSubmit,
            child: _isSubmitting
                ? const LoadingIndicator(size: LoadingSize.sm)
                : Text(widget.submitLabel),
          ),
        ],
      ),
    );
  }

  Widget _buildSuccessView(
    BuildContext context,
    AuthThemeData theme,
    ColorScheme colorScheme,
    TextTheme textTheme,
  ) {
    final email = _emailController.text.trim();
    final description = widget.successDescriptionText ??
        "We've sent a password reset link to $email";

    return AuthCard(
      title: widget.successTitleText,
      description: description,
      logo: widget.logo,
      footer: _buildFooter(context),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(
            Icons.mark_email_read_outlined,
            size: 64,
            color: colorScheme.primary,
          ),
          SizedBox(height: theme.sectionSpacing),
          Text(
            'Didn\'t receive the email? Check your spam folder.',
            style: textTheme.bodySmall?.copyWith(
              color: colorScheme.onSurfaceVariant,
            ),
            textAlign: TextAlign.center,
          ),
        ],
      ),
    );
  }

  Widget? _buildFooter(BuildContext context) {
    if (widget.onSignInTap == null) return null;

    return Center(
      child: TextButton(
        onPressed: widget.onSignInTap,
        child: Text(widget.signInLabel),
      ),
    );
  }
}
