/// Flutter authentication provider — wraps [AuthManager] with reactive state.
///
/// Mirrors the React AuthProvider pattern from `ui/packages/react/src/context.tsx`.
library;

import 'package:flutter/widgets.dart';
import 'package:authsome_core/authsome_core.dart';

import 'secure_token_storage.dart';

// ── AuthNotifier ─────────────────────────────────────

/// [AuthNotifier] wraps [AuthManager] as a [ChangeNotifier] for Flutter.
///
/// This is the core reactive bridge between the framework-agnostic
/// [AuthManager] and Flutter's widget tree.
class AuthNotifier extends ChangeNotifier {
  final AuthManager _manager;
  late AuthState _state;
  ClientConfig? _clientConfig;
  bool _isConfigLoaded = false;
  void Function()? _unsubscribeState;
  void Function()? _unsubscribeConfig;

  /// Creates an [AuthNotifier] with the given [AuthConfig].
  ///
  /// If no [TokenStorage] is provided in the config, defaults to
  /// [SecureTokenStorage] for secure persistence.
  AuthNotifier(AuthConfig config)
      : _manager = AuthManager(
          AuthConfig(
            baseUrl: config.baseUrl,
            publishableKey: config.publishableKey,
            initialClientConfig: config.initialClientConfig,
            storage: config.storage ?? SecureTokenStorage(),
            onStateChange: config.onStateChange,
            onError: config.onError,
          ),
        ) {
    _state = _manager.state;
    _clientConfig = _manager.clientConfig;
    _isConfigLoaded = _clientConfig != null;

    _unsubscribeState = _manager.subscribe((newState) {
      _state = newState;
      notifyListeners();
    });

    _unsubscribeConfig = _manager.subscribeConfig((config) {
      _clientConfig = config;
      _isConfigLoaded = true;
      notifyListeners();
    });
  }

  // ── State Getters ──────────────────────────────────

  /// Current authentication state.
  AuthState get state => _state;

  /// The underlying [AuthManager] instance (for advanced usage).
  AuthManager get manager => _manager;

  /// The HTTP API client.
  AuthSomeClient get client => _manager.client;

  /// Auto-discovered client configuration (null until loaded).
  ClientConfig? get clientConfig => _clientConfig;

  /// Whether the client config has been loaded from the server.
  bool get isConfigLoaded => _isConfigLoaded;

  /// Convenience: whether the user is authenticated.
  bool get isAuthenticated => _state is AuthAuthenticated;

  /// Convenience: whether auth is loading.
  bool get isLoading => _state is AuthLoading;

  /// Convenience: whether MFA is required.
  bool get isMfaRequired => _state is AuthMfaRequired;

  /// Convenience: current user or null.
  dynamic get user {
    if (_state is AuthAuthenticated) {
      return (_state as AuthAuthenticated).user;
    }
    return null;
  }

  /// Convenience: current session or null.
  Session? get session {
    if (_state is AuthAuthenticated) {
      return (_state as AuthAuthenticated).session;
    }
    if (_state is AuthMfaRequired) {
      // ignore: deprecated_member_use
      return (_state as AuthMfaRequired).session;
    }
    return null;
  }

  /// Error message, if in error state.
  String? get error {
    if (_state is AuthError) {
      return (_state as AuthError).error;
    }
    return null;
  }

  // ── Auth Operations ────────────────────────────────

  /// Initialize by hydrating session from storage.
  /// Call this once on app start (e.g., in [AuthProvider]).
  Future<void> initialize() => _manager.initialize();

  /// Sign in with email & password.
  Future<void> signIn(String email, String password) =>
      _manager.signIn(email, password);

  /// Sign up with email & password.
  Future<void> signUp(String email, String password, {String? name}) =>
      _manager.signUp(email, password, name: name);

  /// Sign out.
  Future<void> signOut() => _manager.signOut();

  /// Submit MFA code using the ticket carried by [AuthMfaRequired].
  /// Mirrors React `auth.ts` `submitMFAChallenge(code)`.
  Future<void> submitMFAChallenge(String code) =>
      _manager.submitMFAChallenge(code);

  /// Legacy MFA submission keyed by enrollment ID.
  @Deprecated('Use submitMFAChallenge(code) which reads the ticket from state')
  Future<void> submitMFACode(String enrollmentId, String code) =>
      _manager.submitMFACode(enrollmentId, code);

  /// Submit MFA recovery code.
  Future<void> submitRecoveryCode(String code) =>
      _manager.submitRecoveryCode(code);

  /// Resend the email-verification message for [email].
  Future<void> resendVerification(String email) =>
      _manager.resendVerification(email);

  /// Verify the email-confirmation OTP / token.
  Future<void> verifyEmail(String token) => _manager.verifyEmail(token);

  /// Run a passkey sign-in ceremony. The default
  /// [PasskeyAuthenticator] is web-only; on iOS/Android consumers can
  /// pass a custom one (e.g. backed by the `passkeys` corbado package).
  Future<void> signInWithPasskey({
    required PasskeyAuthenticator authenticator,
    String? email,
  }) =>
      _manager.signInWithPasskey(
        authenticator: authenticator,
        email: email,
      );

  /// Send SMS code for MFA verification.
  Future<SmsSendResult> sendSMSCode() => _manager.sendSMSCode();

  /// Submit SMS verification code for MFA.
  Future<void> submitSMSCode(String code) => _manager.submitSMSCode(code);

  /// Refresh the session manually.
  Future<void> refreshNow() => _manager.refreshNow();

  /// Fetch client config from the backend.
  Future<ClientConfig> fetchClientConfig() => _manager.fetchClientConfig();

  @override
  void dispose() {
    _unsubscribeState?.call();
    _unsubscribeConfig?.call();
    _manager.dispose();
    super.dispose();
  }
}

// ── AuthProvider Widget ──────────────────────────────

/// [AuthProvider] provides authentication state to the widget tree.
///
/// Wrap your app with this widget to enable auth throughout:
///
/// ```dart
/// AuthProvider(
///   config: AuthConfig(
///     baseUrl: 'https://api.example.com',
///     publishableKey: 'pk_...',
///   ),
///   child: MyApp(),
/// )
/// ```
class AuthProvider extends StatefulWidget {
  /// Auth configuration. Null when the widget was constructed via
  /// [AuthProvider.test] with a pre-built notifier.
  final AuthConfig? config;

  /// Externally supplied notifier (test seam). When non-null, the provider
  /// uses it directly instead of constructing one from [config] and will
  /// NOT dispose it (ownership remains with the test).
  final AuthNotifier? injectedNotifier;

  /// Child widget.
  final Widget child;

  const AuthProvider({
    required AuthConfig this.config,
    required this.child,
    super.key,
  }) : injectedNotifier = null;

  /// Test-only constructor that wires a pre-built [AuthNotifier] directly
  /// into the inherited scope. Lets widget tests assert on resolution via
  /// [BuildContext.auth] without spinning up a real HTTP client.
  @visibleForTesting
  const AuthProvider.test({
    required AuthNotifier notifier,
    required this.child,
    super.key,
  })  : config = null,
        injectedNotifier = notifier;

  /// Get the [AuthNotifier] from the widget tree.
  ///
  /// Throws if no [AuthProvider] is found in the ancestor tree.
  static AuthNotifier of(BuildContext context) {
    final scope = context.dependOnInheritedWidgetOfExactType<_AuthScope>();
    if (scope == null) {
      throw FlutterError(
        'AuthProvider.of() was called with a context that does not '
        'contain an AuthProvider.\n'
        'Make sure to wrap your app with AuthProvider.',
      );
    }
    return scope.notifier!;
  }

  /// Try to get the [AuthNotifier] from the widget tree.
  ///
  /// Returns null if no [AuthProvider] is found.
  static AuthNotifier? maybeOf(BuildContext context) {
    final scope = context.dependOnInheritedWidgetOfExactType<_AuthScope>();
    return scope?.notifier;
  }

  @override
  State<AuthProvider> createState() => _AuthProviderState();
}

class _AuthProviderState extends State<AuthProvider> {
  late AuthNotifier _notifier;
  bool _ownsNotifier = true;

  @override
  void initState() {
    super.initState();
    final injected = widget.injectedNotifier;
    if (injected != null) {
      _notifier = injected;
      _ownsNotifier = false;
    } else {
      _notifier = AuthNotifier(widget.config!);
      // Hydrate from storage on mount (matches React useEffect).
      _notifier.initialize();
    }
  }

  @override
  void dispose() {
    if (_ownsNotifier) _notifier.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return _AuthScope(
      notifier: _notifier,
      child: widget.child,
    );
  }
}

/// Internal [InheritedNotifier] that rebuilds dependents on state changes.
class _AuthScope extends InheritedNotifier<AuthNotifier> {
  const _AuthScope({
    required AuthNotifier notifier,
    required super.child,
  }) : super(notifier: notifier);
}

// ── Convenience Extension ────────────────────────────

/// Extension for convenient access to auth state from [BuildContext].
///
/// Usage:
/// ```dart
/// final auth = context.auth;
/// if (auth.isAuthenticated) { ... }
/// ```
extension AuthContext on BuildContext {
  /// Get the [AuthNotifier] from the widget tree.
  AuthNotifier get auth => AuthProvider.of(this);
}
