/// Sign-up form screen with multi-step email → details flow.
///
/// Step 1: Social login buttons (auto-discovered from config), email input.
/// Step 2: The app's configured signup fields (or a name field when there
/// are none), password, and the captcha when one is required.
/// Uses [AnimatedSwitcher] for smooth step transitions.
library;

import 'package:flutter/material.dart';
import 'package:authsome_flutter/authsome_flutter.dart';

import '../theme/auth_theme.dart';
import '../widgets/form_validation.dart';
import '../widgets/auth_card.dart';
import '../widgets/captcha_field.dart';
import '../widgets/error_display.dart';
import '../widgets/password_input.dart';
import '../widgets/signup_field_input.dart';
import 'email_verification_form.dart';
import '../widgets/social_buttons.dart';
import '../widgets/or_divider.dart';
import '../widgets/loading_indicator.dart';

/// A multi-step sign-up form wrapped in an [AuthCard].
///
/// Supports social signup and email/password registration. Follows the
/// server's [ClientConfig] the way React's `sign-up-form.tsx` does:
///
/// - `password.enabled: false` leaves only social sign-up.
/// - `signup_fields` replaces the name field with the app's own fields.
/// - `captcha` adds a challenge before the account is created.
/// - `waitlist.enabled` checks approval when you continue from email.
/// - `email_verification.required` swaps to an [EmailVerificationForm]
///   after sign-up instead of signing the user in.
/// - `branding.appName` names the app in the default title.
class SignUpForm extends StatefulWidget {
  /// Called when sign-up completes successfully.
  final VoidCallback? onSuccess;

  /// Called after email verification, when the user can sign in.
  final VoidCallback? onVerificationComplete;

  /// Called when the user taps the "Sign in" link.
  final VoidCallback? onSignInTap;

  /// Override auto-detected social providers.
  final List<SocialProvider>? socialProviders;

  /// Called when a social login button is tapped.
  final ValueChanged<String>? onSocialLogin;

  /// Layout for social buttons (default: [SocialButtonLayout.grid]).
  final SocialButtonLayout socialLayout;

  /// Renders the captcha on platforms or providers the built-in widget
  /// doesn't cover. See [CaptchaBuilder].
  final CaptchaBuilder? captchaBuilder;

  /// Optional logo widget displayed above the title. When null, the app's
  /// branding logo shows if one is configured.
  final Widget? logo;

  // ── Localization overrides ──

  /// Card title. Defaults to "Create your {app name} account" when the
  /// client config carries a branding app name, otherwise
  /// "Create an account".
  final String? titleText;

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

  /// Optional injected [AuthNotifier]. When null, the form resolves the
  /// notifier from the surrounding [AuthProvider]. Test seam.
  final AuthNotifier? auth;

  /// Title + description text alignment within the [AuthCard]. Defaults to
  /// [AuthCardAlign.center]; pass [AuthCardAlign.left] for a flush-left
  /// layout that matches a product-style sign-up.
  final AuthCardAlign align;

  const SignUpForm({
    this.auth,
    this.onSuccess,
    this.onVerificationComplete,
    this.onSignInTap,
    this.socialProviders,
    this.onSocialLogin,
    this.socialLayout = SocialButtonLayout.grid,
    this.captchaBuilder,
    this.logo,
    this.titleText,
    this.descriptionText = 'Enter your email to get started',
    this.emailLabel = 'Email',
    this.nameLabel = 'Full name',
    this.continueLabel = 'Continue',
    this.signUpLabel = 'Sign up',
    this.signInLabel = 'Already have an account? Sign in',
    this.align = AuthCardAlign.center,
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
  WaitlistStatus? _waitlistStatus;
  String? _captchaToken;
  int _captchaAttempt = 0;
  bool _successReported = false;

  /// Email awaiting verification, set once sign-up lands in
  /// [AuthVerificationPending].
  String? _verifyEmail;

  /// Values of the configured signup fields, keyed by field key.
  final Map<String, String> _fieldValues = {};
  final Map<String, String> _fieldErrors = {};

  /// Field list the defaults were last seeded from.
  List<SignupFieldConfig>? _seededFields;

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
    _nameController.dispose();
    _passwordController.dispose();
    _nameFocusNode.dispose();
    super.dispose();
  }

  void _onAuthStateChanged() {
    if (!mounted) return;
    final auth = _auth!;

    if (auth.state is AuthAuthenticated) {
      if (!_successReported) {
        _successReported = true;
        widget.onSuccess?.call();
      }
      return;
    }

    final state = auth.state;
    if (state is AuthVerificationPending && mounted) {
      setState(() {
        _verifyEmail = state.email;
        _isSubmitting = false;
      });
      return;
    }

    if (auth.error != null && mounted) {
      setState(() {
        _error = auth.error;
        _isSubmitting = false;
      });
      return;
    }
    setState(() {});
  }

  Future<void> _onContinue() async {
    if (_isSubmitting) return;
    final email = _emailController.text.trim().toLowerCase();
    if (!validEmail(email)) {
      setState(() => _error = 'Please enter a valid email address');
      return;
    }
    setState(() {
      _error = null;
      _isSubmitting = true;
    });

    try {
      var config = _auth?.clientConfig;
      if (config == null && _auth!.client.publishableKey != null) {
        config = await _auth!.manager.fetchClientConfig();
        if (!mounted) return;
      }
      if (config?.signupEnabled == false) {
        setState(() => _error = "Signup isn't available for this app.");
        return;
      }
      if (config?.password?.enabled == false) return;
      if (config?.waitlist?.enabled == true) {
        final status = await _auth!.client.joinWaitlistWithStatus(email);
        if (!mounted) return;
        setState(() => _waitlistStatus = status);
        if (status != WaitlistStatus.approved) return;
      }
      if (!mounted) return;
      setState(() {
        _emailController.text = email;
        _waitlistStatus = null;
        _step = 1;
      });
    } catch (e) {
      if (!mounted) return;
      setState(() {
        _error = e is AuthClientException
            ? e.message
            : 'Unable to check waitlist status. Please try again.';
      });
      return;
    } finally {
      if (mounted) setState(() => _isSubmitting = false);
    }
    if (!mounted) return;
    Future.delayed(const Duration(milliseconds: 350), () {
      if (mounted) _nameFocusNode.requestFocus();
    });
  }

  Future<void> _onSignUp() async {
    if (_isSubmitting ||
        _auth?.clientConfig?.signupEnabled == false ||
        _auth?.clientConfig?.password?.enabled == false) {
      return;
    }
    _successReported = false;
    final name = _nameController.text.trim();
    final password = _passwordController.text;
    final fields = _signupFields();

    final errors = <String, String>{};
    for (final field in fields ?? const <SignupFieldConfig>[]) {
      final problem = validateSignupField(field, _fieldValues[field.key] ?? '');
      if (problem != null) errors[field.key] = problem;
    }
    if (errors.isNotEmpty) {
      setState(() {
        _fieldErrors
          ..clear()
          ..addAll(errors);
        _error = null;
      });
      return;
    }

    if (password.isEmpty) {
      setState(() {
        _fieldErrors.clear();
        _error = 'Please enter a password';
      });
      return;
    }

    if (CaptchaField.isRequired(_auth?.clientConfig?.captcha) &&
        _captchaToken == null) {
      setState(() => _error = 'Please complete the captcha');
      return;
    }

    setState(() {
      _error = null;
      _fieldErrors.clear();
      _isSubmitting = true;
    });

    try {
      final values = <String, String>{
        for (final field in fields ?? const <SignupFieldConfig>[])
          if ((_fieldValues[field.key] ?? '').isNotEmpty)
            field.key: _fieldValues[field.key]!,
      };
      await _auth!.signUp(
        _emailController.text.trim(),
        password,
        // The name field only shows when no signup fields are configured.
        name: fields == null && name.isNotEmpty ? name : null,
        fields: fields != null && values.isNotEmpty ? values : null,
        captchaToken: _captchaToken,
      );
    } catch (e) {
      if (mounted) {
        setState(() {
          _error = e.toString();
          _isSubmitting = false;
        });
      }
    } finally {
      if (mounted) {
        setState(() {
          _isSubmitting = false;
          _captchaToken = null;
          _captchaAttempt++;
        });
      }
    }
  }

  void _goBack() {
    if (_isSubmitting) return;
    setState(() {
      _step = 0;
      _error = null;
      _nameController.clear();
      _passwordController.clear();
      _captchaToken = null;
      _captchaAttempt++;
    });
  }

  /// The configured signup fields sorted by order, or null when there are
  /// none (the form then falls back to a single name field).
  List<SignupFieldConfig>? _signupFields() {
    final fields = _auth?.clientConfig?.signupFields;
    if (fields == null || fields.isEmpty) return null;
    final sorted = [...fields]..sort((a, b) => a.order.compareTo(b.order));
    // Seed configured defaults the first time a field list is seen, and
    // again if the config changes. A default only fills a field the user
    // hasn't touched.
    if (!identical(fields, _seededFields)) {
      _seededFields = fields;
      for (final f in sorted) {
        final d = f.defaultValue;
        if (d != null && d.isNotEmpty) _fieldValues.putIfAbsent(f.key, () => d);
      }
    }
    return sorted;
  }

  String _resolveTitle() {
    if (widget.titleText != null) return widget.titleText!;
    final appName = _auth?.clientConfig?.branding?.appName;
    return appName != null && appName.isNotEmpty
        ? 'Create your $appName account'
        : 'Create an account';
  }

  List<SocialProvider> _resolveSocialProviders() {
    if (widget.onSocialLogin == null) return const [];
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
        title: widget.titleText ?? 'Create an account',
        description: widget.descriptionText,
        logo: widget.logo,
        align: widget.align,
        child: const ErrorDisplay(
          error: 'AuthProvider not found in widget tree. Wrap your app in '
              'AuthProvider, or pass an `auth:` notifier to SignUpForm.',
        ),
      );
    }

    final verifyEmail = _verifyEmail;
    if (verifyEmail != null) {
      return EmailVerificationForm(
        auth: _auth,
        email: verifyEmail,
        logo: widget.logo,
        align: widget.align,
        onSuccess: widget.onVerificationComplete ?? widget.onSignInTap,
      );
    }

    if (_auth?.clientConfig?.signupEnabled == false) {
      return AuthCard(
          title: "Signup isn't available",
          description: "New accounts aren't available for this app.",
          logo: widget.logo,
          align: widget.align,
          footer: _buildFooter(context),
          child: const SizedBox.shrink());
    }

    final theme = AuthTheme.of(context);
    final colorScheme = Theme.of(context).colorScheme;
    final providers = _resolveSocialProviders();
    final config = _auth?.clientConfig;
    final showPassword = config?.password?.enabled ?? true;
    final waiting = _waitlistStatus != null;
    final rejected = _waitlistStatus == WaitlistStatus.rejected;

    return AuthCard(
      title: waiting
          ? rejected
              ? "Signup isn't available"
              : "You're on the waitlist"
          : _resolveTitle(),
      description: waiting
          ? rejected
              ? "Access hasn't been approved for this email address."
              : "We'll email ${_emailController.text.trim()} when access is approved."
          : widget.descriptionText,
      logo: widget.logo,
      branding: config?.branding,
      align: widget.align,
      footer: _buildFooter(context),
      child: AnimatedSwitcher(
        duration: const Duration(milliseconds: 300),
        switchInCurve: Curves.easeOut,
        switchOutCurve: Curves.easeIn,
        child: waiting
            ? _buildWaitlistStep(theme)
            : !showPassword
                ? _buildSocialOnly(context, theme, colorScheme, providers)
                : _step == 0
                    ? _buildEmailStep(context, theme, colorScheme, providers)
                    : _buildDetailsStep(context, theme, colorScheme),
      ),
    );
  }

  Widget _buildWaitlistStep(AuthThemeData theme) {
    return Column(
      key: const ValueKey('sign-up-waitlist-step'),
      crossAxisAlignment: CrossAxisAlignment.stretch,
      mainAxisSize: MainAxisSize.min,
      children: [
        ErrorDisplay(error: _error),
        if (_error != null) SizedBox(height: theme.fieldSpacing),
        FilledButton(
          onPressed: _isSubmitting ? null : _onContinue,
          child: _isSubmitting
              ? const LoadingIndicator(size: LoadingSize.sm)
              : const Text('Check status'),
        ),
        TextButton(
          onPressed: _isSubmitting
              ? null
              : () => setState(() {
                    _waitlistStatus = null;
                    _error = null;
                    _step = 0;
                  }),
          child: const Text('Use another email'),
        ),
      ],
    );
  }

  /// Passwords are off: social sign-up is the only way in, as in React.
  Widget _buildSocialOnly(
    BuildContext context,
    AuthThemeData theme,
    ColorScheme colorScheme,
    List<SocialProvider> providers,
  ) {
    return Column(
      key: const ValueKey('sign-up-social-only'),
      crossAxisAlignment: CrossAxisAlignment.stretch,
      mainAxisSize: MainAxisSize.min,
      children: [
        if (providers.isNotEmpty)
          SocialButtons(
            providers: providers,
            onProviderClick: (id) => widget.onSocialLogin?.call(id),
            isLoading: _isSubmitting,
            layout: widget.socialLayout,
          )
        else
          Padding(
            padding: const EdgeInsets.symmetric(vertical: 16),
            child: Text(
              'No sign-up methods are currently available. Please contact '
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
    final fields = _signupFields();
    final captcha = _auth?.clientConfig?.captcha;
    final captchaRequired = CaptchaField.isRequired(captcha);
    final canSubmit =
        !_isSubmitting && (!captchaRequired || _captchaToken != null);
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
        if (fields == null) ...[
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
        ] else
          for (final field in fields) ...[
            SignupFieldInput(
              field: field,
              value: _fieldValues[field.key] ?? '',
              enabled: !_isSubmitting,
              errorText: _fieldErrors[field.key],
              onChanged: (v) => setState(() {
                _fieldValues[field.key] = v;
                _fieldErrors.remove(field.key);
              }),
            ),
            SizedBox(height: theme.fieldSpacing),
          ],
        PasswordInput(
          controller: _passwordController,
          hintText: 'Create a password',
          enabled: !_isSubmitting,
          textInputAction: TextInputAction.done,
          onSubmitted: _onSignUp,
        ),
        if (captchaRequired) ...[
          SizedBox(height: theme.fieldSpacing),
          CaptchaField(
            key: ValueKey(
                '${captcha?.provider}:${captcha?.siteKey}:$_captchaAttempt'),
            config: captcha!,
            builder: widget.captchaBuilder,
            onToken: (token) {
              if (mounted) setState(() => _captchaToken = token);
            },
          ),
        ],
        SizedBox(height: theme.fieldSpacing),
        FilledButton(
          onPressed: canSubmit ? _onSignUp : null,
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
