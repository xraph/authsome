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

  /// Submit MFA code.
  Future<void> submitMFACode(String enrollmentId, String code) =>
      _manager.submitMFACode(enrollmentId, code);

  /// Submit MFA recovery code.
  Future<void> submitRecoveryCode(String code) =>
      _manager.submitRecoveryCode(code);

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
  /// Auth configuration.
  final AuthConfig config;

  /// Child widget.
  final Widget child;

  const AuthProvider({
    required this.config,
    required this.child,
    super.key,
  });

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

  @override
  void initState() {
    super.initState();
    _notifier = AuthNotifier(widget.config);
    // Hydrate from storage on mount (matches React useEffect).
    _notifier.initialize();
  }

  @override
  void dispose() {
    _notifier.dispose();
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
