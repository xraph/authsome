/// Sign-in form screen with multi-step email → password flow.
///
/// Step 1: Social and SSO buttons (auto-discovered from config), email input.
/// Step 2: Password input with forgot-password link, the captcha when the
/// app requires one, and a magic-link alternative when that's enabled.
/// Uses [AnimatedSwitcher] for smooth step transitions.
library;

import 'package:flutter/material.dart';
import 'package:authsome_flutter/authsome_flutter.dart';

import '../theme/auth_theme.dart';
import '../platform/browser.dart';
import '../widgets/auth_card.dart';
import '../widgets/captcha_field.dart';
import '../widgets/error_display.dart';
import '../widgets/passkey_login_button.dart';
import '../widgets/password_input.dart';
import '../widgets/social_buttons.dart';
import '../widgets/sso_buttons.dart';
import '../widgets/or_divider.dart';
import '../widgets/loading_indicator.dart';

/// A multi-step sign-in form wrapped in an [AuthCard].
///
/// Supports social login, SSO, email/password, magic links, and passkey
/// sign-in. Each method follows the server's [ClientConfig] (the same
/// fields React's `sign-in-form.tsx` reads) unless overridden by the
/// matching parameter:
///
/// - `password.enabled: false` drops the password step and leaves only the
///   other methods.
/// - `social` and `sso` add provider and connection buttons.
/// - `passkey.enabled` adds the passkey button.
/// - `magiclink.enabled` offers an emailed sign-in link.
/// - `captcha` adds a challenge before the password is submitted.
/// - `branding.appName` names the app in the default title.
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
  /// `clientConfig.passkey.enabled`, matching React `sign-in-form.tsx`
  /// (`showPasskeyProp ?? config?.passkey?.enabled ?? false`). Pass
  /// `true` or `false` to override. The option and its spacing are hidden
  /// when the selected authenticator is unavailable, even when set to true.
  final bool? showPasskey;

  /// Authenticator used for the passkey ceremony. Defaults to
  /// [defaultPasskeyAuthenticator] (Web-only at the moment).
  final PasskeyAuthenticator? passkeyAuthenticator;

  /// Whether to offer an emailed sign-in link.
  ///
  /// When null (the default), follows `clientConfig.magiclink.enabled`,
  /// like React's `showMagicLink ?? config?.magiclink?.enabled ?? false`.
  final bool? showMagicLink;

  /// Override the SSO connections from `clientConfig.sso`.
  final List<SSOConnectionConfig>? ssoConnections;

  /// Called with the connection id when an SSO button is tapped.
  ///
  /// When null, Flutter Web starts the login and redirects the tab to the
  /// identity provider. Other platforms can't complete that round trip on
  /// their own, so there the SSO buttons only show when this is set (open
  /// the URL from [AuthNotifier.startSSOLogin] with your browser plugin of
  /// choice and finish with [AuthNotifier.completeSSOLogin]).
  final ValueChanged<String>? onSSOLogin;

  /// Where the identity provider should send the user back to. Must be
  /// https and allowlisted on the server (localhost is always allowed).
  /// When null, the server lands on its default `/sso/callback`, where an
  /// [SSOCallbackForm] finishes the sign-in.
  final String? ssoReturnUrl;

  /// Renders the captcha on platforms or providers the built-in widget
  /// doesn't cover. See [CaptchaBuilder].
  final CaptchaBuilder? captchaBuilder;

  /// Optional logo widget displayed above the title. When null, the app's
  /// branding logo shows if one is configured.
  final Widget? logo;

  // ── Localization overrides ──

  /// Card title. Defaults to "Sign in to {app name}" when the client
  /// config carries a branding app name, otherwise "Sign in".
  final String? titleText;

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

  /// Magic-link button label on the password step
  /// (default: "Email me a sign-in link instead").
  final String magicLinkLabel;

  /// Magic-link submit label when passwords are off
  /// (default: "Email me a sign-in link").
  final String sendMagicLinkLabel;

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
    this.showMagicLink,
    this.ssoConnections,
    this.onSSOLogin,
    this.ssoReturnUrl,
    this.captchaBuilder,
    this.logo,
    this.titleText,
    this.descriptionText = 'Enter your email to continue',
    this.emailLabel = 'Email',
    this.continueLabel = 'Continue',
    this.signInLabel = 'Sign in',
    this.forgotPasswordLabel = 'Forgot password?',
    this.signUpLabel = "Don't have an account? Sign up",
    this.magicLinkLabel = 'Email me a sign-in link instead',
    this.sendMagicLinkLabel = 'Email me a sign-in link',
    this.align = AuthCardAlign.center,
    super.key,
  });

  @override
  State<SignInForm> createState() => _SignInFormState();
}

enum _SignInStep { email, password, verify, magicLinkSent }

class _SignInFormState extends State<SignInForm> {
  final _emailController = TextEditingController();
  final _passwordController = TextEditingController();
  final _passwordFocusNode = FocusNode();

  _SignInStep _step = _SignInStep.email;
  String? _error;
  String? _info;
  bool _isSubmitting = false;
  String? _captchaToken;

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
    // The button is disabled until a token arrives, but the keyboard's
    // submit action reaches here directly.
    if (CaptchaField.isRequired(_auth?.clientConfig?.captcha) &&
        _captchaToken == null) {
      setState(() => _error = 'Please complete the captcha');
      return;
    }

    setState(() {
      _error = null;
      _isSubmitting = true;
    });

    try {
      await _auth!.signIn(
        _emailController.text.trim(),
        password,
        captchaToken: _captchaToken,
      );
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

  Future<void> _onSendMagicLink() async {
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
      await _auth!.sendMagicLink(email);
      if (!mounted) return;
      setState(() {
        _step = _SignInStep.magicLinkSent;
        _isSubmitting = false;
      });
    } catch (e) {
      if (!mounted) return;
      setState(() {
        _error = e is AuthClientException ? e.message : e.toString();
        _isSubmitting = false;
      });
    }
  }

  Future<void> _onSSOTap(String connectionId) async {
    final custom = widget.onSSOLogin;
    if (custom != null) {
      custom(connectionId);
      return;
    }
    setState(() {
      _error = null;
      _isSubmitting = true;
    });
    try {
      final url = await _auth!.startSSOLogin(
        connectionId,
        returnUrl: widget.ssoReturnUrl,
      );
      redirectBrowser(url);
    } catch (e) {
      if (!mounted) return;
      setState(() {
        _error = e is AuthClientException ? e.message : e.toString();
        _isSubmitting = false;
      });
    }
  }

  List<SSOConnectionConfig> _resolveSSOConnections() {
    // Off the Web there is no default handler, so the buttons would be dead
    // without one. Hide them, the way the passkey button hides when no
    // authenticator is available.
    if (widget.onSSOLogin == null && !canRedirectBrowser) return const [];
    if (widget.ssoConnections != null) return widget.ssoConnections!;
    final sso = _auth?.clientConfig?.sso;
    if (sso?.enabled != true) return const [];
    return sso!.connections;
  }

  String _resolveTitle() {
    if (widget.titleText != null) return widget.titleText!;
    final appName = _auth?.clientConfig?.branding?.appName;
    return appName != null && appName.isNotEmpty
        ? 'Sign in to $appName'
        : 'Sign in';
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
        title: widget.titleText ?? 'Sign in',
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
    final passkeyAuthenticator =
        widget.passkeyAuthenticator ?? defaultPasskeyAuthenticator();
    final config = _auth?.clientConfig;
    final showPasskey = (widget.showPasskey ??
            config?.passkey?.enabled ??
            false) &&
        passkeyAuthenticator.isAvailable;
    final showPassword = config?.password?.enabled ?? true;
    final showMagicLink =
        widget.showMagicLink ?? config?.magiclink?.enabled ?? false;
    final ssoConnections = _resolveSSOConnections();

    return AuthCard(
      title: _resolveTitle(),
      description: widget.descriptionText,
      logo: widget.logo,
      branding: config?.branding,
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
              ssoConnections,
              showPasskey: showPasskey,
              passkeyAuthenticator: passkeyAuthenticator,
              showPassword: showPassword,
              showMagicLink: showMagicLink,
            ),
          _SignInStep.password => _buildPasswordStep(
              context,
              theme,
              colorScheme,
              showMagicLink: showMagicLink,
              captcha: config?.captcha,
            ),
          _SignInStep.verify => _buildVerifyStep(context, theme, colorScheme),
          _SignInStep.magicLinkSent =>
            _buildMagicLinkSentStep(context, theme, colorScheme),
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
    List<SocialProvider> providers,
    List<SSOConnectionConfig> ssoConnections, {
    required bool showPasskey,
    required PasskeyAuthenticator passkeyAuthenticator,
    required bool showPassword,
    required bool showMagicLink,
  }) {
    final hasSocial = providers.isNotEmpty;
    final hasSSO = ssoConnections.isNotEmpty;
    final hasAuthOptions = hasSocial || hasSSO || showPasskey;
    // With passwords off, the email field only exists to receive a magic
    // link. Without one either, the buttons above are the whole form.
    final showEmailField = showPassword || showMagicLink;
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
          if (hasSSO || showPasskey) SizedBox(height: theme.fieldSpacing),
        ],
        if (hasSSO) ...[
          SSOButtons(
            connections: ssoConnections,
            onConnectionTap: _onSSOTap,
            isLoading: _isSubmitting,
          ),
          if (showPasskey) SizedBox(height: theme.fieldSpacing),
        ],
        if (showPasskey)
          PasskeyLoginButton(
            auth: _auth,
            authenticator: passkeyAuthenticator,
            onSuccess: widget.onSuccess,
          ),
        if (hasAuthOptions && showEmailField) ...[
          SizedBox(height: theme.fieldSpacing),
          const OrDivider(),
          SizedBox(height: theme.fieldSpacing),
        ],
        if (hasAuthOptions && !showEmailField && _error != null)
          SizedBox(height: theme.fieldSpacing),
        ErrorDisplay(error: _error),
        if (_error != null && showEmailField)
          SizedBox(height: theme.fieldSpacing),
        if (showEmailField) ...[
          TextField(
            controller: _emailController,
            enabled: !_isSubmitting,
            keyboardType: TextInputType.emailAddress,
            textInputAction: TextInputAction.next,
            onSubmitted: (_) =>
                showPassword ? _onContinue() : _onSendMagicLink(),
            decoration: InputDecoration(
              labelText: widget.emailLabel,
              hintText: 'you@example.com',
              border: const OutlineInputBorder(),
            ),
          ),
          SizedBox(height: theme.fieldSpacing),
          FilledButton(
            onPressed: _isSubmitting
                ? null
                : (showPassword ? _onContinue : _onSendMagicLink),
            child: _isSubmitting
                ? const LoadingIndicator(size: LoadingSize.sm)
                : Text(showPassword
                    ? widget.continueLabel
                    : widget.sendMagicLinkLabel),
          ),
        ],
        if (!hasAuthOptions && !showEmailField)
          Padding(
            padding: const EdgeInsets.symmetric(vertical: 16),
            child: Text(
              'No sign-in methods are currently available. Please contact '
              'your administrator.',
              textAlign: TextAlign.center,
              style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                    color: colorScheme.onSurfaceVariant,
                  ),
            ),
          ),
      ],
    );
  }

  Widget _buildMagicLinkSentStep(
    BuildContext context,
    AuthThemeData theme,
    ColorScheme colorScheme,
  ) {
    return Column(
      key: const ValueKey('sign-in-magic-link-sent-step'),
      crossAxisAlignment: CrossAxisAlignment.stretch,
      mainAxisSize: MainAxisSize.min,
      children: [
        Icon(Icons.mark_email_read_outlined,
            size: 40, color: colorScheme.primary),
        SizedBox(height: theme.fieldSpacing),
        Text(
          'Check your inbox',
          textAlign: TextAlign.center,
          style: Theme.of(context).textTheme.titleMedium,
        ),
        const SizedBox(height: 8),
        Text(
          'We sent a sign-in link to ${_emailController.text.trim()}.',
          textAlign: TextAlign.center,
          style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                color: colorScheme.onSurfaceVariant,
              ),
        ),
        SizedBox(height: theme.fieldSpacing),
        TextButton.icon(
          onPressed: _goBack,
          icon: const Icon(Icons.arrow_back, size: 18),
          label: const Text('Use a different email'),
        ),
      ],
    );
  }

  Widget _buildPasswordStep(
    BuildContext context,
    AuthThemeData theme,
    ColorScheme colorScheme, {
    required bool showMagicLink,
    required CaptchaConfig? captcha,
  }) {
    final captchaRequired = CaptchaField.isRequired(captcha);
    final canSubmit =
        !_isSubmitting && (!captchaRequired || _captchaToken != null);
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
        if (captchaRequired) ...[
          SizedBox(height: theme.fieldSpacing),
          CaptchaField(
            config: captcha!,
            builder: widget.captchaBuilder,
            onToken: (token) => setState(() => _captchaToken = token),
          ),
        ],
        SizedBox(height: theme.fieldSpacing),
        FilledButton(
          onPressed: canSubmit ? _onSignIn : null,
          child: _isSubmitting
              ? const LoadingIndicator(size: LoadingSize.sm)
              : Text(widget.signInLabel),
        ),
        if (showMagicLink) ...[
          SizedBox(height: theme.fieldSpacing / 2),
          OutlinedButton(
            onPressed: _isSubmitting ? null : _onSendMagicLink,
            child: Text(widget.magicLinkLabel),
          ),
        ],
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
