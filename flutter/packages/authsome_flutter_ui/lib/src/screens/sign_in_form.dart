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
import '../widgets/passkey_login_button.dart';
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
  /// Optional injected [AuthNotifier]. When null, the form resolves the
  /// notifier from the surrounding [AuthProvider]. Primarily a testing seam
  /// so widget tests can drive the form without mounting an [AuthProvider].
  final AuthNotifier? auth;

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
  ///
  /// When null (the default), the value is auto-derived from
  /// `clientConfig.passkey.enabled` — mirrors React `sign-in-form.tsx`
  /// (`showPasskeyProp ?? config?.passkey?.enabled ?? false`). Pass
  /// `true` or `false` to override.
  final bool? showPasskey;

  /// Authenticator used for the passkey ceremony. Defaults to
  /// [defaultPasskeyAuthenticator] (Web-only at the moment).
  final PasskeyAuthenticator? passkeyAuthenticator;

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

  /// Title + description text alignment within the [AuthCard]. Defaults to
  /// [AuthCardAlign.center]; pass [AuthCardAlign.left] for a flush-left
  /// layout that matches a product-style sign-in (e.g. shadcn).
  final AuthCardAlign align;

  const SignInForm({
    this.auth,
    this.onSuccess,
    this.onSignUpTap,
    this.signUpUrl,
    this.onForgotPasswordTap,
    this.forgotPasswordUrl,
    this.socialProviders,
    this.onSocialLogin,
    this.socialLayout = SocialButtonLayout.grid,
    this.showPasskey,
    this.passkeyAuthenticator,
    this.logo,
    this.titleText = 'Sign in',
    this.descriptionText = 'Enter your email to continue',
    this.emailLabel = 'Email',
    this.continueLabel = 'Continue',
    this.signInLabel = 'Sign in',
    this.forgotPasswordLabel = 'Forgot password?',
    this.signUpLabel = "Don't have an account? Sign up",
    this.align = AuthCardAlign.center,
    super.key,
  });

  @override
  State<SignInForm> createState() => _SignInFormState();
}

enum _SignInStep { email, password, verify }

class _SignInFormState extends State<SignInForm> {
  final _emailController = TextEditingController();
  final _passwordController = TextEditingController();
  final _passwordFocusNode = FocusNode();

  _SignInStep _step = _SignInStep.email;
  String? _error;
  String? _info;
  bool _isSubmitting = false;

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
      _step = _SignInStep.password;
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
    } on AuthClientException catch (e) {
      if (!mounted) return;
      if (e.isEmailNotVerified) {
        setState(() {
          _step = _SignInStep.verify;
          _error = null;
          _info = null;
          _isSubmitting = false;
        });
        return;
      }
      setState(() {
        _error = e.message;
        _isSubmitting = false;
      });
    } catch (e) {
      if (mounted) {
        setState(() {
          _error = e.toString();
          _isSubmitting = false;
        });
      }
    }
  }

  Future<void> _onResendVerification() async {
    if (_isSubmitting) return;
    setState(() {
      _isSubmitting = true;
      _error = null;
      _info = null;
    });
    try {
      await _auth!.resendVerification(_emailController.text.trim());
      if (mounted) {
        setState(() {
          _info = 'Verification email sent. Check your inbox.';
          _isSubmitting = false;
        });
      }
    } catch (e) {
      if (mounted) {
        setState(() {
          _error = e is AuthClientException ? e.message : e.toString();
          _isSubmitting = false;
        });
      }
    }
  }

  void _goBack() {
    setState(() {
      _step = _SignInStep.email;
      _error = null;
      _info = null;
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
    if (_missingProvider) {
      return AuthCard(
        title: widget.titleText,
        description: widget.descriptionText,
        logo: widget.logo,
        align: widget.align,
        child: const ErrorDisplay(
          error:
              'AuthProvider not found in widget tree. Wrap your app in '
              'AuthProvider, or pass an `auth:` notifier to SignInForm.',
        ),
      );
    }

    final theme = AuthTheme.of(context);
    final colorScheme = Theme.of(context).colorScheme;
    final providers = _resolveSocialProviders();
    // Auto-derive passkey visibility from client config when the
    // caller hasn't pinned a value — mirrors React `sign-in-form.tsx`
    // line 96: `showPasskeyProp ?? config?.passkey?.enabled ?? false`.
    final showPasskey = widget.showPasskey ??
        _auth?.clientConfig?.passkey?.enabled ??
        false;

    return AuthCard(
      title: widget.titleText,
      description: widget.descriptionText,
      logo: widget.logo,
      align: widget.align,
      footer: _buildFooter(context),
      child: AnimatedSwitcher(
        duration: const Duration(milliseconds: 300),
        switchInCurve: Curves.easeOut,
        switchOutCurve: Curves.easeIn,
        child: switch (_step) {
          _SignInStep.email => _buildEmailStep(
              context,
              theme,
              colorScheme,
              providers,
              showPasskey: showPasskey,
            ),
          _SignInStep.password =>
            _buildPasswordStep(context, theme, colorScheme),
          _SignInStep.verify => _buildVerifyStep(context, theme, colorScheme),
        },
      ),
    );
  }

  Widget _buildVerifyStep(
    BuildContext context,
    AuthThemeData theme,
    ColorScheme colorScheme,
  ) {
    return Column(
      key: const ValueKey('sign-in-verify-step'),
      crossAxisAlignment: CrossAxisAlignment.stretch,
      mainAxisSize: MainAxisSize.min,
      children: [
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
        Text(
          'Verify your email',
          style: Theme.of(context).textTheme.titleMedium,
        ),
        const SizedBox(height: 8),
        Text(
          'Please verify your email address before signing in. Check your inbox for a verification link.',
          style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                color: colorScheme.onSurfaceVariant,
              ),
        ),
        SizedBox(height: theme.fieldSpacing),
        ErrorDisplay(error: _error),
        if (_info != null) ...[
          Container(
            width: double.infinity,
            padding:
                const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
            decoration: BoxDecoration(
              color: colorScheme.secondaryContainer,
              borderRadius: BorderRadius.circular(8),
            ),
            child: Text(
              _info!,
              style: Theme.of(context).textTheme.bodySmall?.copyWith(
                    color: colorScheme.onSecondaryContainer,
                  ),
            ),
          ),
          SizedBox(height: theme.fieldSpacing),
        ],
        Align(
          alignment: Alignment.centerLeft,
          child: TextButton(
            onPressed: _isSubmitting ? null : _onResendVerification,
            child: const Text('Resend'),
          ),
        ),
      ],
    );
  }

  Widget _buildEmailStep(
    BuildContext context,
    AuthThemeData theme,
    ColorScheme colorScheme,
    List<SocialProvider> providers, {
    required bool showPasskey,
  }) {
    final hasSocial = providers.isNotEmpty;
    final hasAuthOptions = hasSocial || showPasskey;
    return Column(
      key: const ValueKey('sign-in-email-step'),
      crossAxisAlignment: CrossAxisAlignment.stretch,
      mainAxisSize: MainAxisSize.min,
      children: [
        if (hasSocial) ...[
          SocialButtons(
            providers: providers,
            onProviderClick: (id) {
              widget.onSocialLogin?.call(id);
            },
            isLoading: _isSubmitting,
            layout: widget.socialLayout,
          ),
          if (showPasskey) SizedBox(height: theme.fieldSpacing),
        ],
        if (showPasskey)
          PasskeyLoginButton(
            auth: _auth,
            authenticator: widget.passkeyAuthenticator,
            onSuccess: widget.onSuccess,
          ),
        if (hasAuthOptions) ...[
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
