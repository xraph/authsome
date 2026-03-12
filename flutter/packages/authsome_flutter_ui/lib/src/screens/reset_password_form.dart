/// Reset-password form screen.
///
/// Accepts a reset token and allows the user to set a new password
/// with confirmation. Shows a success state on completion.
library;

import 'package:flutter/material.dart';
import 'package:authsome_flutter/authsome_flutter.dart';

import '../theme/auth_theme.dart';
import '../widgets/auth_card.dart';
import '../widgets/error_display.dart';
import '../widgets/password_input.dart';
import '../widgets/loading_indicator.dart';

/// A reset-password form that takes a [token] and submits a new password.
///
/// Validates that the new password and confirmation match before
/// calling [AuthSomeClient.resetPassword].
class ResetPasswordForm extends StatefulWidget {
  /// The password-reset token (typically from a deep link or URL param).
  final String token;

  /// Called when the password is reset successfully.
  final VoidCallback? onSuccess;

  /// Optional logo widget displayed above the title.
  final Widget? logo;

  // ── Localization overrides ──

  /// Card title (default: "Reset password").
  final String titleText;

  /// Card description (default: "Enter your new password").
  final String descriptionText;

  /// New password field label (default: "New password").
  final String newPasswordLabel;

  /// Confirm password field label (default: "Confirm password").
  final String confirmPasswordLabel;

  /// Submit button label (default: "Reset password").
  final String submitLabel;

  /// Success title (default: "Password reset").
  final String successTitleText;

  /// Success description (default: "Your password has been reset successfully").
  final String successDescriptionText;

  /// Mismatch error (default: "Passwords do not match").
  final String mismatchError;

  const ResetPasswordForm({
    required this.token,
    this.onSuccess,
    this.logo,
    this.titleText = 'Reset password',
    this.descriptionText = 'Enter your new password',
    this.newPasswordLabel = 'New password',
    this.confirmPasswordLabel = 'Confirm password',
    this.submitLabel = 'Reset password',
    this.successTitleText = 'Password reset',
    this.successDescriptionText =
        'Your password has been reset successfully',
    this.mismatchError = 'Passwords do not match',
    super.key,
  });

  @override
  State<ResetPasswordForm> createState() => _ResetPasswordFormState();
}

class _ResetPasswordFormState extends State<ResetPasswordForm> {
  final _passwordController = TextEditingController();
  final _confirmController = TextEditingController();

  String? _error;
  bool _isSubmitting = false;
  bool _isSuccess = false;

  @override
  void dispose() {
    _passwordController.dispose();
    _confirmController.dispose();
    super.dispose();
  }

  Future<void> _onSubmit() async {
    final password = _passwordController.text;
    final confirm = _confirmController.text;

    if (password.isEmpty) {
      setState(() => _error = 'Please enter a new password');
      return;
    }
    if (password != confirm) {
      setState(() => _error = widget.mismatchError);
      return;
    }

    setState(() {
      _error = null;
      _isSubmitting = true;
    });

    try {
      final auth = context.auth;
      await auth.client.resetPassword(body: {
        'token': widget.token,
        'password': password,
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

  @override
  Widget build(BuildContext context) {
    final theme = AuthTheme.of(context);
    final colorScheme = Theme.of(context).colorScheme;

    if (_isSuccess) {
      return _buildSuccessView(context, theme, colorScheme);
    }

    return AuthCard(
      title: widget.titleText,
      description: widget.descriptionText,
      logo: widget.logo,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        mainAxisSize: MainAxisSize.min,
        children: [
          ErrorDisplay(error: _error),
          if (_error != null) SizedBox(height: theme.fieldSpacing),
          PasswordInput(
            controller: _passwordController,
            hintText: widget.newPasswordLabel,
            labelText: widget.newPasswordLabel,
            enabled: !_isSubmitting,
            textInputAction: TextInputAction.next,
          ),
          SizedBox(height: theme.fieldSpacing),
          PasswordInput(
            controller: _confirmController,
            hintText: widget.confirmPasswordLabel,
            labelText: widget.confirmPasswordLabel,
            enabled: !_isSubmitting,
            textInputAction: TextInputAction.done,
            onSubmitted: _onSubmit,
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
