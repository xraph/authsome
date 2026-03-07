/// Framework-agnostic authentication state machine.
///
/// Manages sign-in/sign-up flows, token persistence, automatic refresh,
/// and MFA challenges. Framework adapters (Flutter, etc.) wrap this
/// class and expose reactive state.
///
/// Ported from `ui/packages/core/src/auth.ts`.
library;

import 'dart:async';
import 'dart:convert';

import 'client.dart';
import 'types.dart';

const _sessionKey = 'authsome:session';
const _configKey = 'authsome:client_config';
const _refreshBeforeMs = 60000; // Refresh 60 s before expiry.
const _configTtlMs = 5 * 60000; // Cache client config for 5 minutes.

/// AuthManager is the core state machine that drives authentication.
///
/// Usage:
/// ```dart
/// final auth = AuthManager(AuthConfig(baseUrl: 'https://api.example.com'));
/// auth.subscribe((state) => print(state));
/// await auth.initialize();
/// await auth.signIn('email@example.com', 'password');
/// ```
class AuthManager {
  final AuthSomeClient _client;
  final TokenStorage _storage;
  AuthState _state = const AuthIdle();
  final Set<void Function(AuthState)> _listeners = {};
  Timer? _refreshTimer;
  final void Function(String error, int? code)? _onError;

  final String? _publishableKey;
  ClientConfig? _clientConfig;
  final Set<void Function(ClientConfig?)> _configListeners = {};
  Future<ClientConfig>? _configFetchFuture;

  AuthManager(AuthConfig config)
      : _client = AuthSomeClient(
          AuthClientConfig(baseUrl: config.baseUrl),
        ),
        _storage = config.storage ?? MemoryTokenStorage(),
        _onError = config.onError,
        _publishableKey = config.publishableKey {
    if (config.initialClientConfig != null) {
      _clientConfig = config.initialClientConfig;
    }

    if (config.onStateChange != null) {
      _listeners.add(config.onStateChange!);
    }
  }

  // ── Public API ────────────────────────────────────

  /// Current auth state (snapshot).
  AuthState get state => _state;

  /// Access the underlying HTTP client.
  AuthSomeClient get client => _client;

  /// Get the cached client config (or null if not yet fetched).
  ClientConfig? get clientConfig => _clientConfig;

  /// Subscribe to state changes. Returns an unsubscribe function.
  void Function() subscribe(void Function(AuthState) listener) {
    _listeners.add(listener);
    return () => _listeners.remove(listener);
  }

  /// Subscribe to client config changes. Returns an unsubscribe function.
  void Function() subscribeConfig(
      void Function(ClientConfig?) listener) {
    _configListeners.add(listener);
    return () => _configListeners.remove(listener);
  }

  /// Initialize by hydrating the session from storage.
  /// When a publishableKey is set, also fetches client config in parallel.
  /// Call this once on app start.
  Future<void> initialize() async {
    // Kick off config fetch in parallel (non-blocking).
    if (_publishableKey != null && _clientConfig == null) {
      // ignore: unawaited_futures
      fetchClientConfig();
    }

    try {
      final raw = await _storage.getItem(_sessionKey);
      if (raw == null) {
        _setState(const AuthUnauthenticated());
        return;
      }

      final sessionJson =
          jsonDecode(raw) as Map<String, dynamic>;
      final session = Session.fromJson(sessionJson);
      final expiresAt = DateTime.parse(session.expiresAt).millisecondsSinceEpoch;

      if (DateTime.now().millisecondsSinceEpoch >= expiresAt) {
        // Token expired — try refresh.
        await _refreshSession(session.refreshToken);
        return;
      }

      // Token still valid — fetch user profile.
      _setState(const AuthLoading());
      final user = await _client.getMeWithToken(session.sessionToken);
      _setState(AuthAuthenticated(user: user, session: session));
      _scheduleRefresh(session);
    } catch (_) {
      await _clearSession();
      _setState(const AuthUnauthenticated());
    }
  }

  /// Sign in with email & password.
  Future<void> signIn(String email, String password) async {
    _setState(const AuthLoading());
    try {
      final res = await _client.signInWithCredentials(
        email: email,
        password: password,
      );
      final session = Session(
        sessionToken: res.sessionToken,
        refreshToken: res.refreshToken,
        expiresAt: res.expiresAt,
      );
      await _handleAuthResponse(res.user, session);
    } catch (err) {
      _handleError(err);
    }
  }

  /// Sign up with email & password.
  Future<void> signUp(String email, String password, {String? name}) async {
    _setState(const AuthLoading());
    try {
      final res = await _client.signUpWithCredentials(
        email: email,
        password: password,
        name: name,
      );
      final session = Session(
        sessionToken: res.sessionToken,
        refreshToken: res.refreshToken,
        expiresAt: res.expiresAt,
      );
      await _handleAuthResponse(res.user, session);
    } catch (err) {
      _handleError(err);
    }
  }

  /// Sign out and clear the session.
  Future<void> signOut() async {
    final token = getSessionToken();
    if (token != null) {
      try {
        await _client.signOutWithToken(token);
      } catch (_) {
        // Best-effort server sign-out.
      }
    }
    _clearRefreshTimer();
    await _clearSession();
    _setState(const AuthUnauthenticated());
  }

  /// Submit an MFA challenge code.
  Future<void> submitMFACode(String enrollmentId, String code) async {
    _setState(const AuthLoading());
    try {
      final res = await _client.mfaChallenge(
        enrollmentId: enrollmentId,
        code: code,
      );
      final session = Session(
        sessionToken: res.sessionToken,
        refreshToken: res.refreshToken,
        expiresAt: res.expiresAt,
      );
      await _handleAuthResponse(res.user, session);
    } catch (err) {
      _handleError(err);
    }
  }

  /// Submit an MFA recovery code.
  Future<void> submitRecoveryCode(String code) async {
    _setState(const AuthLoading());
    try {
      final res = await _client.verifyRecoveryCodeWithString(code);
      final session = Session(
        sessionToken: res.sessionToken,
        refreshToken: res.refreshToken,
        expiresAt: res.expiresAt,
      );
      await _handleAuthResponse(res.user, session);
    } catch (err) {
      _handleError(err);
    }
  }

  /// Send an SMS code for MFA verification. Returns masked phone + expiry info.
  Future<SmsSendResult> sendSMSCode() async {
    final token = getSessionToken();
    if (token == null) {
      throw StateError('No session token available');
    }
    final res = await _client.sendSMSCodeForMFA(token);
    return SmsSendResult(
      sent: res.sent,
      phoneMasked: res.phoneMasked,
      expiresInSeconds: res.expiresInSeconds,
    );
  }

  /// Submit an SMS verification code during MFA challenge.
  Future<void> submitSMSCode(String code) async {
    _setState(const AuthLoading());
    try {
      final token = getSessionToken();
      if (token == null) {
        throw StateError('No session token available');
      }
      final res = await _client.verifySMSCodeForMFA(code, token);
      final session = Session(
        sessionToken: res.sessionToken,
        refreshToken: res.refreshToken,
        expiresAt: res.expiresAt,
      );
      await _handleAuthResponse(res.user, session);
    } catch (err) {
      _handleError(err);
    }
  }

  /// Refresh the current session manually.
  Future<void> refreshNow() async {
    if (_state is! AuthAuthenticated) return;
    final session = (_state as AuthAuthenticated).session;
    await _refreshSession(session.refreshToken);
  }

  /// Get the current session token (if authenticated or MFA required).
  String? getSessionToken() {
    if (_state is AuthAuthenticated) {
      return (_state as AuthAuthenticated).session.sessionToken;
    }
    if (_state is AuthMfaRequired) {
      return (_state as AuthMfaRequired).session.sessionToken;
    }
    return null;
  }

  /// Get the current user (if authenticated).
  dynamic getUser() {
    if (_state is AuthAuthenticated) {
      return (_state as AuthAuthenticated).user;
    }
    return null;
  }

  // ── Client Config API ────────────────────────────

  /// Fetch client config from the backend.
  /// Deduplicates concurrent calls and caches the result with a TTL.
  Future<ClientConfig> fetchClientConfig() {
    // Deduplicate concurrent fetches.
    if (_configFetchFuture != null) {
      return _configFetchFuture!;
    }

    _configFetchFuture = _doFetchClientConfig().whenComplete(() {
      _configFetchFuture = null;
    });

    return _configFetchFuture!;
  }

  /// Tear down: clear timers and listeners.
  void dispose() {
    _clearRefreshTimer();
    _listeners.clear();
    _configListeners.clear();
  }

  // ── Internals ─────────────────────────────────────

  Future<ClientConfig> _doFetchClientConfig() async {
    // Check storage cache.
    try {
      final cached = await _storage.getItem(_configKey);
      if (cached != null) {
        final parsed = jsonDecode(cached) as Map<String, dynamic>;
        final config =
            ClientConfig.fromJson(parsed['config'] as Map<String, dynamic>);
        final fetchedAt = (parsed['fetchedAt'] as num).toInt();
        if (DateTime.now().millisecondsSinceEpoch - fetchedAt < _configTtlMs) {
          _setClientConfig(config);
          return config;
        }
      }
    } catch (_) {
      // Cache miss or parse error — fetch fresh.
    }

    final config = await _client.fetchClientConfig(_publishableKey);
    _setClientConfig(config);

    // Cache with timestamp.
    try {
      await _storage.setItem(
        _configKey,
        jsonEncode({
          'config': config.toJson(),
          'fetchedAt': DateTime.now().millisecondsSinceEpoch,
        }),
      );
    } catch (_) {
      // Storage write failure is non-fatal.
    }

    return config;
  }

  Future<void> _handleAuthResponse(dynamic user, Session session) async {
    await _persistSession(session);
    _setState(AuthAuthenticated(user: user, session: session));
    _scheduleRefresh(session);
  }

  Future<void> _refreshSession(String refreshToken) async {
    try {
      final res = await _client.refreshWithToken(refreshToken);
      final newSession = Session(
        sessionToken: res.sessionToken,
        refreshToken: res.refreshToken,
        expiresAt: res.expiresAt,
      );
      final user = await _client.getMeWithToken(newSession.sessionToken);
      await _persistSession(newSession);
      _setState(AuthAuthenticated(user: user, session: newSession));
      _scheduleRefresh(newSession);
    } catch (_) {
      await _clearSession();
      _setState(const AuthUnauthenticated());
    }
  }

  void _scheduleRefresh(Session session) {
    _clearRefreshTimer();
    final expiresAt =
        DateTime.parse(session.expiresAt).millisecondsSinceEpoch;
    final delay = expiresAt -
        DateTime.now().millisecondsSinceEpoch -
        _refreshBeforeMs;

    if (delay <= 0) {
      // Already near expiry — refresh immediately.
      _refreshSession(session.refreshToken);
      return;
    }

    _refreshTimer = Timer(Duration(milliseconds: delay), () {
      _refreshSession(session.refreshToken);
    });
  }

  void _clearRefreshTimer() {
    _refreshTimer?.cancel();
    _refreshTimer = null;
  }

  Future<void> _persistSession(Session session) async {
    await _storage.setItem(_sessionKey, jsonEncode(session.toJson()));
  }

  Future<void> _clearSession() async {
    await _storage.removeItem(_sessionKey);
  }

  void _setClientConfig(ClientConfig config) {
    _clientConfig = config;
    for (final listener in _configListeners) {
      try {
        listener(config);
      } catch (_) {
        // Listener errors should not break the config flow.
      }
    }
  }

  void _setState(AuthState newState) {
    _state = newState;
    for (final listener in _listeners) {
      try {
        listener(newState);
      } catch (_) {
        // Listener errors should not break the state machine.
      }
    }
  }

  void _handleError(Object err) {
    final message = err is AuthClientException
        ? err.message
        : 'An unexpected error occurred';
    final code = err is AuthClientException ? err.code : null;

    // MFA required is returned as a specific error code.
    if (code == 403 && message.toLowerCase().contains('mfa')) {
      final token = getSessionToken();
      if (token != null) {
        _setState(AuthMfaRequired(
          session: Session(
            sessionToken: token,
            refreshToken: '',
            expiresAt: DateTime.now()
                .add(const Duration(minutes: 5))
                .toIso8601String(),
          ),
        ));
        return;
      }
    }

    _setState(AuthError(error: message));
    _onError?.call(message, code);
  }
}
