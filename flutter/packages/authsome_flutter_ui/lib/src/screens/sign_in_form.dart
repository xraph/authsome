/// Sign-in form screen with multi-step email → password flow.
///
/// Step 1: Social login buttons (auto-discovered from config), email input.
/// Step 2: Password input with forgot-password link.
/// Uses [AnimatedSwitcher] for smooth step transitions.
library;

import 'package:flutter/material.dart';
import 'package:authsome_flutter/authsome_flutter.dart';

import '../theme/auth_theme.dart';
import '../widgets/auth_card.dart';
import '../widgets/error_display.dart';
import '../widgets/password_input.dart';
import '../widgets/social_buttons.dart';
import '../widgets/or_divider.dart';
import '../widgets/loading_indicator.dart';

/// A multi-step sign-in form wrapped in an [AuthCard].
///
/// Supports social login, email/password authentication, and passkey (visual only).
/// Social providers are auto-discovered from the server's [ClientConfig] unless
/// explicitly overridden via [socialProviders].
class SignInForm extends StatefulWidget {
  /// Called when sign-in completes successfully.
  final VoidCallback? onSuccess;

  /// Called when the user taps the "Sign up" link.
  final VoidCallback? onSignUpTap;

  /// URL to navigate to for sign-up (used if [onSignUpTap] is null).
  final String? signUpUrl;

  /// Called when the user taps "Forgot password?".
  final VoidCallback? onForgotPasswordTap;

  /// URL to navigate to for forgot password (used if [onForgotPasswordTap] is null).
  final String? forgotPasswordUrl;

  /// Override auto-detected social providers.
  final List<SocialProvider>? socialProviders;

  /// Called when a social login button is tapped.
  final ValueChanged<String>? onSocialLogin;

  /// Layout for social buttons (default: [SocialButtonLayout.grid]).
  final SocialButtonLayout socialLayout;

  /// Whether to show the passkey option.
  final bool showPasskey;

  /// Optional logo widget displayed above the title.
  final Widget? logo;

  // ── Localization overrides ──

  /// Card title (default: "Sign in").
  final String titleText;

  /// Card description (default: "Enter your email to continue").
  final String descriptionText;

  /// Email field label (default: "Email").
  final String emailLabel;

  /// Continue button label (default: "Continue").
  final String continueLabel;

  /// Sign-in button label (default: "Sign in").
  final String signInLabel;

  /// Forgot-password link label (default: "Forgot password?").
  final String forgotPasswordLabel;

  /// Sign-up link label (default: "Don't have an account? Sign up").
  final String signUpLabel;

  const SignInForm({
    this.onSuccess,
    this.onSignUpTap,
    this.signUpUrl,
    this.onForgotPasswordTap,
    this.forgotPasswordUrl,
    this.socialProviders,
    this.onSocialLogin,
    this.socialLayout = SocialButtonLayout.grid,
    this.showPasskey = false,
    this.logo,
    this.titleText = 'Sign in',
    this.descriptionText = 'Enter your email to continue',
    this.emailLabel = 'Email',
    this.continueLabel = 'Continue',
    this.signInLabel = 'Sign in',
    this.forgotPasswordLabel = 'Forgot password?',
    this.signUpLabel = "Don't have an account? Sign up",
    super.key,
  });

  @override
  State<SignInForm> createState() => _SignInFormState();
}

class _SignInFormState extends State<SignInForm> {
  final _emailController = TextEditingController();
  final _passwordController = TextEditingController();
  final _passwordFocusNode = FocusNode();

  int _step = 0; // 0 = email, 1 = password
  String? _error;
  bool _isSubmitting = false;

  AuthNotifier? _auth;

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    if (_auth == null) {
      _auth = context.auth;
      _auth!.addListener(_onAuthStateChanged);
    }
  }

  @override
  void dispose() {
    _auth?.removeListener(_onAuthStateChanged);
    _emailController.dispose();
    _passwordController.dispose();
    _passwordFocusNode.dispose();
    super.dispose();
  }

  void _onAuthStateChanged() {
    if (!mounted) return;
    final auth = _auth!;

    if (auth.state is AuthAuthenticated) {
      widget.onSuccess?.call();
      return;
    }

    if (auth.error != null && mounted) {
      setState(() {
        _error = auth.error;
        _isSubmitting = false;
      });
    }
  }

  void _onContinue() {
    final email = _emailController.text.trim();
    if (email.isEmpty) {
      setState(() => _error = 'Please enter your email');
      return;
    }
    setState(() {
      _error = null;
      _step = 1;
    });
    // Focus the password field after the transition.
    Future.delayed(const Duration(milliseconds: 350), () {
      if (mounted) _passwordFocusNode.requestFocus();
    });
  }

  Future<void> _onSignIn() async {
    final password = _passwordController.text;
    if (password.isEmpty) {
      setState(() => _error = 'Please enter your password');
      return;
    }

    setState(() {
      _error = null;
      _isSubmitting = true;
    });

    try {
      await _auth!.signIn(_emailController.text.trim(), password);
    } catch (e) {
      if (mounted) {
        setState(() {
          _error = e.toString();
          _isSubmitting = false;
        });
      }
    }
  }

  void _goBack() {
    setState(() {
      _step = 0;
      _error = null;
      _passwordController.clear();
    });
  }

  List<SocialProvider> _resolveSocialProviders() {
    if (widget.socialProviders != null) return widget.socialProviders!;
    final config = _auth?.clientConfig;
    if (config?.social?.enabled != true) return const [];
    return config!.social!.providers
        .map((p) => SocialProvider(id: p.id, name: p.name))
        .toList();
  }

  @override
  Widget build(BuildContext context) {
    final theme = AuthTheme.of(context);
    final colorScheme = Theme.of(context).colorScheme;
    final providers = _resolveSocialProviders();

    return AuthCard(
      title: widget.titleText,
      description: widget.descriptionText,
      logo: widget.logo,
      footer: _buildFooter(context),
      child: AnimatedSwitcher(
        duration: const Duration(milliseconds: 300),
        switchInCurve: Curves.easeOut,
        switchOutCurve: Curves.easeIn,
        child: _step == 0
            ? _buildEmailStep(context, theme, colorScheme, providers)
            : _buildPasswordStep(context, theme, colorScheme),
      ),
    );
  }

  Widget _buildEmailStep(
    BuildContext context,
    AuthThemeData theme,
    ColorScheme colorScheme,
    List<SocialProvider> providers,
  ) {
    return Column(
      key: const ValueKey('sign-in-email-step'),
      crossAxisAlignment: CrossAxisAlignment.stretch,
      mainAxisSize: MainAxisSize.min,
      children: [
        if (providers.isNotEmpty) ...[
          SocialButtons(
            providers: providers,
            onProviderClick: (id) {
              widget.onSocialLogin?.call(id);
            },
            isLoading: _isSubmitting,
            layout: widget.socialLayout,
          ),
          SizedBox(height: theme.fieldSpacing),
          const OrDivider(),
          SizedBox(height: theme.fieldSpacing),
        ],
        ErrorDisplay(error: _error),
        if (_error != null) SizedBox(height: theme.fieldSpacing),
        TextField(
          controller: _emailController,
          enabled: !_isSubmitting,
          keyboardType: TextInputType.emailAddress,
          textInputAction: TextInputAction.next,
          onSubmitted: (_) => _onContinue(),
          decoration: InputDecoration(
            labelText: widget.emailLabel,
            hintText: 'you@example.com',
            border: const OutlineInputBorder(),
          ),
        ),
        SizedBox(height: theme.fieldSpacing),
        FilledButton(
          onPressed: _isSubmitting ? null : _onContinue,
          child: _isSubmitting
              ? const LoadingIndicator(size: LoadingSize.sm)
              : Text(widget.continueLabel),
        ),
      ],
    );
  }

  Widget _buildPasswordStep(
    BuildContext context,
    AuthThemeData theme,
    ColorScheme colorScheme,
  ) {
    return Column(
      key: const ValueKey('sign-in-password-step'),
      crossAxisAlignment: CrossAxisAlignment.stretch,
      mainAxisSize: MainAxisSize.min,
      children: [
        // Back button + email display row.
        Row(
          children: [
            IconButton(
              icon: const Icon(Icons.arrow_back, size: 20),
              onPressed: _isSubmitting ? null : _goBack,
              tooltip: 'Back',
              style: IconButton.styleFrom(
                padding: EdgeInsets.zero,
                minimumSize: const Size(36, 36),
              ),
            ),
            const SizedBox(width: 8),
            Expanded(
              child: Text(
                _emailController.text.trim(),
                style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                      color: colorScheme.onSurfaceVariant,
                    ),
                overflow: TextOverflow.ellipsis,
              ),
            ),
          ],
        ),
        SizedBox(height: theme.fieldSpacing),
        ErrorDisplay(error: _error),
        if (_error != null) SizedBox(height: theme.fieldSpacing),
        PasswordInput(
          controller: _passwordController,
          focusNode: _passwordFocusNode,
          hintText: 'Password',
          enabled: !_isSubmitting,
          textInputAction: TextInputAction.done,
          onSubmitted: _onSignIn,
        ),
        const SizedBox(height: 8),
        Align(
          alignment: Alignment.centerRight,
          child: TextButton(
            onPressed: _isSubmitting
                ? null
                : (widget.onForgotPasswordTap ?? () {}),
            child: Text(widget.forgotPasswordLabel),
          ),
        ),
        SizedBox(height: theme.fieldSpacing),
        FilledButton(
          onPressed: _isSubmitting ? null : _onSignIn,
          child: _isSubmitting
              ? const LoadingIndicator(size: LoadingSize.sm)
              : Text(widget.signInLabel),
        ),
      ],
    );
  }

  Widget? _buildFooter(BuildContext context) {
    final hasSignUp = widget.onSignUpTap != null || widget.signUpUrl != null;
    if (!hasSignUp) return null;

    return Center(
      child: TextButton(
        onPressed: widget.onSignUpTap,
        child: Text(widget.signUpLabel),
      ),
    );
  }
}
