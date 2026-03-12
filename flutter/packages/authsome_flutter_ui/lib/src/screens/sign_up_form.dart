/// Sign-up form screen with multi-step email → details flow.
///
/// Step 1: Social login buttons (auto-discovered from config), email input.
/// Step 2: Name and password inputs with sign-up submission.
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

/// A multi-step sign-up form wrapped in an [AuthCard].
///
/// Supports social signup, and email/password registration with an
/// optional name field. Social providers are auto-discovered from
/// the server's [ClientConfig] unless explicitly overridden.
class SignUpForm extends StatefulWidget {
  /// Called when sign-up completes successfully.
  final VoidCallback? onSuccess;

  /// Called when the user taps the "Sign in" link.
  final VoidCallback? onSignInTap;

  /// Override auto-detected social providers.
  final List<SocialProvider>? socialProviders;

  /// Called when a social login button is tapped.
  final ValueChanged<String>? onSocialLogin;

  /// Layout for social buttons (default: [SocialButtonLayout.grid]).
  final SocialButtonLayout socialLayout;

  /// Optional logo widget displayed above the title.
  final Widget? logo;

  // ── Localization overrides ──

  /// Card title (default: "Create an account").
  final String titleText;

  /// Card description (default: "Enter your email to get started").
  final String descriptionText;

  /// Email field label (default: "Email").
  final String emailLabel;

  /// Name field label (default: "Full name").
  final String nameLabel;

  /// Continue button label (default: "Continue").
  final String continueLabel;

  /// Sign-up button label (default: "Sign up").
  final String signUpLabel;

  /// Sign-in link label (default: "Already have an account? Sign in").
  final String signInLabel;

  const SignUpForm({
    this.onSuccess,
    this.onSignInTap,
    this.socialProviders,
    this.onSocialLogin,
    this.socialLayout = SocialButtonLayout.grid,
    this.logo,
    this.titleText = 'Create an account',
    this.descriptionText = 'Enter your email to get started',
    this.emailLabel = 'Email',
    this.nameLabel = 'Full name',
    this.continueLabel = 'Continue',
    this.signUpLabel = 'Sign up',
    this.signInLabel = 'Already have an account? Sign in',
    super.key,
  });

  @override
  State<SignUpForm> createState() => _SignUpFormState();
}

class _SignUpFormState extends State<SignUpForm> {
  final _emailController = TextEditingController();
  final _nameController = TextEditingController();
  final _passwordController = TextEditingController();
  final _nameFocusNode = FocusNode();

  int _step = 0; // 0 = email, 1 = details
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
    _nameController.dispose();
    _passwordController.dispose();
    _nameFocusNode.dispose();
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
    Future.delayed(const Duration(milliseconds: 350), () {
      if (mounted) _nameFocusNode.requestFocus();
    });
  }

  Future<void> _onSignUp() async {
    final name = _nameController.text.trim();
    final password = _passwordController.text;

    if (password.isEmpty) {
      setState(() => _error = 'Please enter a password');
      return;
    }

    setState(() {
      _error = null;
      _isSubmitting = true;
    });

    try {
      await _auth!.signUp(
        _emailController.text.trim(),
        password,
        name: name.isNotEmpty ? name : null,
      );
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
      _nameController.clear();
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
            : _buildDetailsStep(context, theme, colorScheme),
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
      key: const ValueKey('sign-up-email-step'),
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

  Widget _buildDetailsStep(
    BuildContext context,
    AuthThemeData theme,
    ColorScheme colorScheme,
  ) {
    return Column(
      key: const ValueKey('sign-up-details-step'),
      crossAxisAlignment: CrossAxisAlignment.stretch,
      mainAxisSize: MainAxisSize.min,
      children: [
        // Back button + email display.
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
        TextField(
          controller: _nameController,
          focusNode: _nameFocusNode,
          enabled: !_isSubmitting,
          textInputAction: TextInputAction.next,
          decoration: InputDecoration(
            labelText: widget.nameLabel,
            border: const OutlineInputBorder(),
          ),
        ),
        SizedBox(height: theme.fieldSpacing),
        PasswordInput(
          controller: _passwordController,
          hintText: 'Create a password',
          enabled: !_isSubmitting,
          textInputAction: TextInputAction.done,
          onSubmitted: _onSignUp,
        ),
        SizedBox(height: theme.fieldSpacing),
        FilledButton(
          onPressed: _isSubmitting ? null : _onSignUp,
          child: _isSubmitting
              ? const LoadingIndicator(size: LoadingSize.sm)
              : Text(widget.signUpLabel),
        ),
      ],
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
