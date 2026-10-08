/// Waitlist sign-up screen. Mirrors React `waitlist-form.tsx`.
library;

import 'package:flutter/material.dart';
import 'package:authsome_flutter/authsome_flutter.dart';

import '../theme/auth_theme.dart';
import '../widgets/auth_card.dart';
import '../widgets/error_display.dart';
import '../widgets/loading_indicator.dart';

/// Collects an email (and an optional name) for the app's waitlist.
///
/// Follows `clientConfig.waitlist`: when the backend reports the waitlist
/// as disabled the form renders nothing, the same as React. Without that
/// field (an older backend, or no publishable key) the form shows.
class WaitlistForm extends StatefulWidget {
  /// Optional injected [AuthNotifier]. When null, the form resolves the
  /// notifier from the surrounding [AuthProvider]. Test seam.
  final AuthNotifier? auth;

  /// Called after the email is on the list.
  final VoidCallback? onSuccess;

  /// Called when the user taps the "Sign in" link. No link when null.
  final VoidCallback? onSignInTap;

  /// Optional logo widget displayed above the title. When null, the app's
  /// branding logo shows if one is configured.
  final Widget? logo;

  // ── Localization overrides ──

  /// Card title (default: "Join the waitlist").
  final String titleText;

  /// Card description
  /// (default: "Sign up to be notified when access is available.").
  final String descriptionText;

  /// Email field label (default: "Email").
  final String emailLabel;

  /// Name field label (default: "Name (optional)").
  final String nameLabel;

  /// Submit button label (default: "Join waitlist").
  final String submitLabel;

  /// Success title (default: "You're on the list!").
  final String successTitleText;

  /// Success description
  /// (default: "We'll notify you when your spot is ready.").
  final String successDescriptionText;

  /// Sign-in link label (default: "Already have an account? Sign in").
  final String signInLabel;

  /// Title + description alignment within the [AuthCard].
  final AuthCardAlign align;

  const WaitlistForm({
    this.auth,
    this.onSuccess,
    this.onSignInTap,
    this.logo,
    this.titleText = 'Join the waitlist',
    this.descriptionText = 'Sign up to be notified when access is available.',
    this.emailLabel = 'Email',
    this.nameLabel = 'Name (optional)',
    this.submitLabel = 'Join waitlist',
    this.successTitleText = "You're on the list!",
    this.successDescriptionText = "We'll notify you when your spot is ready.",
    this.signInLabel = 'Already have an account? Sign in',
    this.align = AuthCardAlign.center,
    super.key,
  });

  @override
  State<WaitlistForm> createState() => _WaitlistFormState();
}

class _WaitlistFormState extends State<WaitlistForm> {
  final _emailController = TextEditingController();
  final _nameController = TextEditingController();

  String? _error;
  bool _isSubmitting = false;
  bool _isSuccess = false;

  AuthNotifier? _auth;
  bool _missingProvider = false;

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
      // Rebuild when the client config arrives, so a disabled waitlist
      // disappears even when the notifier was injected.
      _auth!.addListener(_onAuthChanged);
    }
  }

  @override
  void dispose() {
    _auth?.removeListener(_onAuthChanged);
    _emailController.dispose();
    _nameController.dispose();
    super.dispose();
  }

  void _onAuthChanged() {
    if (mounted) setState(() {});
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
      final name = _nameController.text.trim();
      await _auth!.joinWaitlist(email, name: name.isEmpty ? null : name);
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
            : 'Failed to join the waitlist. Please try again.';
        _isSubmitting = false;
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    if (_missingProvider) {
      return AuthCard(
        title: widget.titleText,
        description: widget.descriptionText,
        logo: widget.logo,
        align: widget.align,
        child: const ErrorDisplay(
          error: 'AuthProvider not found in widget tree. Wrap your app in '
              'AuthProvider, or pass an `auth:` notifier to WaitlistForm.',
        ),
      );
    }

    final config = _auth?.clientConfig;
    if (config?.waitlist?.enabled == false) return const SizedBox.shrink();

    final theme = AuthTheme.of(context);
    final colorScheme = Theme.of(context).colorScheme;

    if (_isSuccess) {
      return AuthCard(
        title: widget.successTitleText,
        description: widget.successDescriptionText,
        logo: widget.logo,
        branding: config?.branding,
        align: widget.align,
        footer: _buildFooter(),
        child: Icon(
          Icons.check_circle_outline,
          size: 56,
          color: colorScheme.primary,
        ),
      );
    }

    return AuthCard(
      title: widget.titleText,
      description: widget.descriptionText,
      logo: widget.logo,
      branding: config?.branding,
      align: widget.align,
      footer: _buildFooter(),
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
            textInputAction: TextInputAction.next,
            decoration: InputDecoration(
              labelText: widget.emailLabel,
              hintText: 'you@example.com',
              border: const OutlineInputBorder(),
            ),
          ),
          SizedBox(height: theme.fieldSpacing),
          TextField(
            controller: _nameController,
            enabled: !_isSubmitting,
            textInputAction: TextInputAction.done,
            onSubmitted: (_) => _onSubmit(),
            decoration: InputDecoration(
              labelText: widget.nameLabel,
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

  Widget? _buildFooter() {
    if (widget.onSignInTap == null) return null;
    return Center(
      child: TextButton(
        onPressed: widget.onSignInTap,
        child: Text(widget.signInLabel),
      ),
    );
  }
}
