/// Change-password form screen.
///
/// Allows an authenticated user to change their password by providing
/// the current password and a new password with confirmation.
library;

import 'package:flutter/material.dart';
import 'package:authsome_flutter/authsome_flutter.dart';

import '../theme/auth_theme.dart';
import '../widgets/auth_card.dart';
import '../widgets/error_display.dart';
import '../widgets/password_input.dart';
import '../widgets/loading_indicator.dart';

/// A change-password form with current, new, and confirm password fields.
///
/// Validates that:
/// - The new password is different from the current password.
/// - The new password matches the confirmation field.
///
/// Calls [AuthSomeClient.changePassword] with the user's session token.
class ChangePasswordForm extends StatefulWidget {
  /// Called when the password is changed successfully.
  final VoidCallback? onSuccess;

  // ── Localization overrides ──

  /// Card title (default: "Change password").
  final String titleText;

  /// Card description (default: "Enter your current password and choose a new one").
  final String descriptionText;

  /// Current password field label (default: "Current password").
  final String currentPasswordLabel;

  /// New password field label (default: "New password").
  final String newPasswordLabel;

  /// Confirm password field label (default: "Confirm new password").
  final String confirmPasswordLabel;

  /// Submit button label (default: "Change password").
  final String submitLabel;

  /// Same-password error (default: "New password must be different from current password").
  final String samePasswordError;

  /// Mismatch error (default: "Passwords do not match").
  final String mismatchError;

  /// Success title (default: "Password changed").
  final String successTitleText;

  /// Success description (default: "Your password has been updated successfully").
  final String successDescriptionText;

  const ChangePasswordForm({
    this.onSuccess,
    this.titleText = 'Change password',
    this.descriptionText =
        'Enter your current password and choose a new one',
    this.currentPasswordLabel = 'Current password',
    this.newPasswordLabel = 'New password',
    this.confirmPasswordLabel = 'Confirm new password',
    this.submitLabel = 'Change password',
    this.samePasswordError =
        'New password must be different from current password',
    this.mismatchError = 'Passwords do not match',
    this.successTitleText = 'Password changed',
    this.successDescriptionText =
        'Your password has been updated successfully',
    super.key,
  });

  @override
  State<ChangePasswordForm> createState() => _ChangePasswordFormState();
}

class _ChangePasswordFormState extends State<ChangePasswordForm> {
  final _currentController = TextEditingController();
  final _newController = TextEditingController();
  final _confirmController = TextEditingController();

  String? _error;
  bool _isSubmitting = false;
  bool _isSuccess = false;

  @override
  void dispose() {
    _currentController.dispose();
    _newController.dispose();
    _confirmController.dispose();
    super.dispose();
  }

  Future<void> _onSubmit() async {
    final current = _currentController.text;
    final newPw = _newController.text;
    final confirm = _confirmController.text;

    if (current.isEmpty) {
      setState(() => _error = 'Please enter your current password');
      return;
    }
    if (newPw.isEmpty) {
      setState(() => _error = 'Please enter a new password');
      return;
    }
    if (newPw == current) {
      setState(() => _error = widget.samePasswordError);
      return;
    }
    if (newPw != confirm) {
      setState(() => _error = widget.mismatchError);
      return;
    }

    setState(() {
      _error = null;
      _isSubmitting = true;
    });

    try {
      final auth = context.auth;
      await auth.client.changePassword(
        body: {
          'current_password': current,
          'new_password': newPw,
        },
        token: auth.session?.sessionToken ?? '',
      );
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
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        mainAxisSize: MainAxisSize.min,
        children: [
          ErrorDisplay(error: _error),
          if (_error != null) SizedBox(height: theme.fieldSpacing),
          PasswordInput(
            controller: _currentController,
            hintText: widget.currentPasswordLabel,
            labelText: widget.currentPasswordLabel,
            enabled: !_isSubmitting,
            textInputAction: TextInputAction.next,
          ),
          SizedBox(height: theme.fieldSpacing),
          PasswordInput(
            controller: _newController,
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
