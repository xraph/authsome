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

import 'package:meta/meta.dart';

import 'client.dart';
import 'exceptions.dart';
import 'passkey_authenticator.dart';
import 'plugins.dart';
import 'types.dart';
import 'webauthn.dart';

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
  String get _configCacheKey =>
      '$_configKey:${jsonEncode([_client.baseUrl, _publishableKey ?? ""])}';
  final Set<void Function(ClientConfig?)> _configListeners = {};
  Future<ClientConfig>? _configFetchFuture;

  /// Tracks the most recent sign-in email so error-state transitions
  /// ([AuthMfaRequired], [AuthEmailNotVerified]) can carry it through to
  /// the UI without callers having to pass it again.
  String? _pendingEmail;

  final Map<String, _PluginSlot> _plugins = {};
  final StreamController<AuthEvent> _events =
      StreamController<AuthEvent>.broadcast(sync: true);
  String? _activeOrgId;

  /// The active organization /v1/me reported on the last profile load. The
  /// next transition to authenticated consumes it.
  String? _reportedOrgId;

  AuthManager(AuthConfig config)
      : _client = AuthSomeClient(
          AuthClientConfig(
            baseUrl: config.baseUrl,
            publishableKey: config.publishableKey,
            httpClient: config.httpClient,
          ),
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
    _registerPlugins(config.plugins);
  }

  /// Test-only constructor that accepts a pre-built [AuthSomeClient].
  /// Lets tests inject an `http.Client` via `MockClient` and drive the
  /// state machine without touching the network.
  @visibleForTesting
  AuthManager.withClient({
    required AuthSomeClient client,
    TokenStorage? storage,
    String? publishableKey,
    ClientConfig? initialClientConfig,
    void Function(String, int?)? onError,
    void Function(AuthState)? onStateChange,
    List<AuthPlugin<Object>> plugins = const [],
  })  : _client = client,
        _storage = storage ?? MemoryTokenStorage(),
        _onError = onError,
        _publishableKey = publishableKey {
    if (initialClientConfig != null) {
      _clientConfig = initialClientConfig;
    }
    if (onStateChange != null) {
      _listeners.add(onStateChange);
    }
    _registerPlugins(plugins);
  }

  void _registerPlugins(List<AuthPlugin<Object>> plugins) {
    for (final plugin in plugins) {
      if (_plugins.containsKey(plugin.name)) {
        throw ArgumentError.value(
            plugin.name, 'plugins', 'the "${plugin.name}" plugin is registered twice');
      }
      _plugins[plugin.name] = _PluginSlot(plugin);
    }
  }

  // ── Plugins ───────────────────────────────────────

  /// The API of the registered plugin that provides [Api], built on first
  /// use: `auth.plugin<SubscriptionApi>()`. Throws
  /// [PluginNotRegisteredError] when no registered plugin provides it, and
  /// rethrows whatever the plugin's init threw.
  Api plugin<Api extends Object>() {
    final slot = _slotFor<Api>();
    if (slot == null) throw PluginNotRegisteredError(Api);
    if (!slot.built) {
      slot.built = true;
      try {
        slot.api = slot.plugin.init(_PluginContextImpl(this, slot.plugin.name));
      } catch (error, stack) {
        slot.error = error;
        slot.stack = stack;
      }
    }
    if (slot.error != null) {
      Error.throwWithStackTrace(slot.error!, slot.stack ?? StackTrace.current);
    }
    return slot.api! as Api;
  }

  /// Whether a registered plugin provides [Api].
  bool hasPlugin<Api extends Object>() => _slotFor<Api>() != null;

  _PluginSlot? _slotFor<Api extends Object>() {
    if (Api == Object) {
      throw ArgumentError(
          'authsome: plugin() needs the API type, e.g. plugin<SubscriptionApi>()');
    }
    final matches = [
      for (final slot in _plugins.values)
        if (slot.plugin is AuthPlugin<Api>) slot,
    ];
    if (matches.length > 1) {
      throw StateError('authsome: more than one registered plugin provides $Api '
          '(${matches.map((s) => s.plugin.name).join(', ')})');
    }
    return matches.isEmpty ? null : matches.single;
  }

  /// Lifecycle events: [AuthEvent.signedIn], [AuthEvent.signedOut],
  /// [AuthEvent.sessionRefreshed] and [AuthEvent.orgChanged].
  Stream<AuthEvent> get events => _events.stream;

  // ── Active organization ───────────────────────────

  /// The organization the session is working in, null when it has none.
  /// Restored from /v1/me on load, set by [switchOrg], cleared on sign-out.
  String? get activeOrgId => _activeOrgId;

  /// Switches the session into another organization. On an app that issues
  /// JWTs the server hands back a new token carrying the organization; it
  /// replaces the stored one. Fires [AuthEvent.orgChanged].
  Future<void> switchOrg(String orgId) async {
    if (_state is! AuthAuthenticated) {
      throw StateError('authsome: you have to be signed in to switch organization');
    }
    // Runs in turn with refreshes, so a refresh rotating the refresh token
    // meanwhile isn't overwritten with the spent one.
    return _exclusive(() => _applyOrgSwitch(orgId));
  }

  Future<void> _applyOrgSwitch(String orgId) async {
    final before = _state;
    if (before is! AuthAuthenticated) {
      throw StateError('authsome: you have to be signed in to switch organization');
    }
    final res = await _client.switchOrg(
      body: SwitchOrgRequest(orgId: orgId),
      token: before.session.sessionToken,
    );

    // The session may have ended, or become someone else's, while the
    // request was out. Don't bring it back.
    final now = _state;
    if (now is! AuthAuthenticated || _userId(now) != _userId(before)) {
      throw StateError('authsome: the session changed while switching organization');
    }

    final token = res.sessionToken;
    if (token != null && token.isNotEmpty) {
      final raw = await _storage.getItem(_sessionKey);
      final latest = raw == null
          ? now.session
          : Session.fromJson(jsonDecode(raw) as Map<String, dynamic>);
      final next = Session(
        sessionToken: token,
        refreshToken: latest.refreshToken,
        expiresAt: res.expiresAt ?? latest.expiresAt,
      );
      await _persistSession(next);
      _setState(AuthAuthenticated(user: now.user, session: next));
      _scheduleRefresh(next);
    }

    // An empty org id is the server saying the session has none now.
    final reported = res.orgId;
    final org = (reported != null && reported.isNotEmpty) ? reported : null;
    final changed = _activeOrgId != org;
    _activeOrgId = org;
    if (changed) _emit(AuthEvent.orgChanged);
  }

  /// Runs [op] after every operation queued before it has finished, so a
  /// refresh and an organization switch never interleave.
  Future<T> _exclusive<T>(Future<T> Function() op) {
    final result = _serial.then((_) => op());
    _serial = result.then((_) {}, onError: (Object _) {});
    return result;
  }

  Future<void> _serial = Future<void>.value();

  /// The signed-in user's id, null when unknown (an offline restore).
  static String? _userId(AuthState state) {
    if (state is! AuthAuthenticated) return null;
    final user = state.user;
    if (user is User) return user.id;
    if (user is Map) return user['id'] as String?;
    return null;
  }

  Future<User> _loadUser(String token) async {
    final me = await _client.getMe(token: token);
    _reportedOrgId = me.activeOrgId;
    return User.fromJson(me.toJson());
  }

  void _emit(AuthEvent event) {
    if (!_events.isClosed) _events.add(event);
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
  void Function() subscribeConfig(void Function(ClientConfig?) listener) {
    _configListeners.add(listener);
    return () => _configListeners.remove(listener);
  }

  /// Initialize by hydrating the session from storage.
  /// When a publishableKey is set, also fetches client config in parallel.
  /// Call this once on app start.
  ///
  /// Defensive: never destroys the session on errors during initialization.
  /// The server may be temporarily unreachable (404/500/network) or the
  /// token might need refreshing. Only an explicit [signOut] clears storage.
  /// Mirrors React `auth.ts` initialize (lines 134–147).
  Future<void> initialize() async {
    // Kick off config fetch in parallel (non-blocking).
    if (_publishableKey != null && _clientConfig == null) {
      unawaited(fetchClientConfig().catchError((Object error) {
        // Continue can retry a failed configuration request.
        return const ClientConfig();
      }));
    }

    try {
      final raw = await _storage.getItem(_sessionKey);
      if (raw == null) {
        _setState(const AuthUnauthenticated());
        return;
      }

      final sessionJson = jsonDecode(raw) as Map<String, dynamic>;
      final session = Session.fromJson(sessionJson);
      final expiresAt =
          DateTime.parse(session.expiresAt).millisecondsSinceEpoch;

      if (DateTime.now().millisecondsSinceEpoch >= expiresAt) {
        // Token expired — try refresh.
        await _refreshSession(session.refreshToken);
        return;
      }

      // Token still valid — fetch user profile.
      _setState(const AuthLoading());
      final user = await _loadUser(session.sessionToken);
      _setState(AuthAuthenticated(user: user, session: session));
      _scheduleRefresh(session);
    } catch (_) {
      // Defensive fallback: preserve whatever session we have and schedule a
      // refresh attempt rather than wiping the user out on a transient error.
      try {
        final stored = await _storage.getItem(_sessionKey);
        if (stored != null) {
          final session = Session.fromJson(
            jsonDecode(stored) as Map<String, dynamic>,
          );
          _setState(AuthAuthenticated(user: null, session: session));
          _scheduleRefresh(session);
          return;
        }
      } catch (_) {
        // Storage itself is broken — fall through to unauthenticated.
      }
      _setState(const AuthUnauthenticated());
    }
  }

  /// Sign in with email & password.
  ///
  /// Re-throws on `email_not_verified` so callers can branch into the
  /// inline verification panel without polling state — mirrors React
  /// `auth.ts` lines 197–206.
  Future<void> signIn(
    String email,
    String password, {
    String? captchaToken,
  }) async {
    _pendingEmail = email;
    // The form owns submission progress so parent loading views keep it mounted.
    try {
      final res = await _client.signInWithCredentials(
        email: email,
        password: password,
        captchaToken: captchaToken,
      );
      final session = Session(
        sessionToken: res.sessionToken,
        refreshToken: res.refreshToken,
        expiresAt: res.expiresAt,
      );
      await _handleAuthResponse(res.user, session);
    } catch (err) {
      _handleError(err);
      if (err is AuthClientException && err.isEmailNotVerified) {
        rethrow;
      }
    }
  }

  /// Sign up with email & password.
  ///
  /// [fields] holds the values of the app's configured signup fields; see
  /// [AuthSomeClient.signUpWithCredentials] for how they map onto the
  /// request.
  ///
  /// When the client config says email verification is required, the
  /// session the server returns can't be used until the address is
  /// verified, so it is discarded and the state moves to
  /// [AuthVerificationPending] instead of [AuthAuthenticated]. React's
  /// `signUp` does the same.
  Future<void> signUp(
    String email,
    String password, {
    String? name,
    Map<String, String>? fields,
    String? captchaToken,
  }) async {
    _pendingEmail = email;

    try {
      final config = _clientConfig ??
          (_publishableKey != null ? await fetchClientConfig() : null);
      if (config?.signupEnabled == false) {
        throw StateError("Signup isn't available for this app.");
      }
      final res = await _client.signUpWithCredentials(
        email: email,
        password: password,
        name: name,
        fields: fields,
        captchaToken: captchaToken,
      );
      final verification = config?.emailVerification;
      if (verification != null &&
          (!verification.enabled || !verification.required)) {
        try {
          final user = await _client.getMeWithToken(res.sessionToken);
          final session = Session(
            sessionToken: res.sessionToken,
            refreshToken: res.refreshToken,
            expiresAt: res.expiresAt,
          );
          await _handleAuthResponse(user, session);
          return;
        } on AuthClientException catch (error) {
          if (error.code != 401 && error.code != 403) {
            rethrow;
          }
        }
      }
      _setState(AuthVerificationPending(email: email));
    } catch (err) {
      _handleError(err);
    }
  }

  /// Start an SSO login and return the IdP URL to send the user to.
  ///
  /// Doesn't touch auth state: the user leaves the app here, and the
  /// round trip ends in [completeSSOLogin] on the landing page.
  Future<String> startSSOLogin(String connectionId, {String? returnUrl}) {
    return _client.startSSOLoginUrl(connectionId, returnUrl: returnUrl);
  }

  /// Finish an SSO login by redeeming the one-time `?code=` the backend
  /// appended to the return URL. Rethrows on failure so the landing page
  /// can show why.
  Future<void> completeSSOLogin(String code) async {
    _setState(const AuthLoading());
    try {
      final res = await _client.ssoExchangeCode(code);
      final session = Session(
        sessionToken: res.sessionToken,
        refreshToken: res.refreshToken,
        expiresAt: res.expiresAt,
      );
      await _handleAuthResponse(res.user, session);
    } catch (err) {
      _handleError(err);
      rethrow;
    }
  }

  /// Email a magic sign-in link to [email].
  Future<void> sendMagicLink(String email) async {
    await _client.sendMagicLink(body: SendRequest(email: email));
  }

  /// Put [email] (and an optional [name]) on the app's waitlist.
  Future<void> joinWaitlist(String email, {String? name}) {
    return _client.joinWaitlist(email: email, name: name);
  }

  /// Approve the OAuth device authorization request identified by
  /// [userCode] (RFC 8628) as the signed-in user.
  Future<void> approveDeviceAuthorization(String userCode) async {
    final token = getSessionToken();
    if (token == null) {
      throw const AuthClientException(
        'Sign in before authorizing a device',
        code: 401,
      );
    }
    await _client.oauth2DeviceComplete(
      body: Oauth2DeviceCompleteRequest(action: 'approve', userCode: userCode),
      token: token,
    );
  }

  /// Submit an MFA challenge code using the ticket from [AuthMfaRequired].
  /// Mirrors React `auth.ts` `submitMFAChallenge(code)`.
  Future<void> submitMFAChallenge(String code) async {
    final current = _state;
    if (current is! AuthMfaRequired) {
      throw StateError('submitMFAChallenge called outside MFA flow');
    }
    try {
      final res = await _client.challengeMFA(
        body: ChallengeRequest(mfaTicket: current.mfaTicket, code: code),
      );
      final session = Session(
        sessionToken: res.sessionToken,
        refreshToken: res.refreshToken,
        expiresAt: res.expiresAt,
      );
      final user = res.user is Map
          ? User.fromJson(Map<String, dynamic>.from(res.user as Map))
          : res.user;
      await _handleAuthResponse(user, session);
    } catch (err) {
      // Keep the ticket available so a failed code can be retried.
      _onError?.call(err is AuthClientException ? err.message : err.toString(),
          err is AuthClientException ? err.code : null);
      rethrow;
    }
  }

  /// Resend the email-verification message for [email].
  Future<void> resendVerification(String email) async {
    await _client.resendEmailVerification(
      body: ResendVerificationRequest(email: email),
    );
  }

  /// Run a passkey / WebAuthn sign-in ceremony.
  ///
  /// Orchestrates the three-step flow:
  ///   1. POST `/v1/passkeys/login/begin` (with optional [email] for
  ///      discoverable-credential UX) → server returns assertion options
  ///      with base64url-encoded binary fields.
  ///   2. [authenticator.authenticate] runs the WebAuthn ceremony on the
  ///      platform (browser, iOS, Android) — exactly the
  ///      `navigator.credentials.get` step in React's
  ///      `passkey-login-button.tsx`.
  ///   3. POST `/v1/passkeys/login/finish` with the credential payload
  ///      → server returns session + user, identical to the password
  ///      sign-in response.
  Future<void> signInWithPasskey({
    required PasskeyAuthenticator authenticator,
    String? email,
  }) async {
    if (!authenticator.isAvailable) {
      throw const AuthClientException(
        'Passkeys are not available on this platform',
        code: 400,
      );
    }
    if (email != null) {
      _pendingEmail = email;
    }
    _setState(const AuthLoading());
    try {
      // No email is passed: the endpoint never read it, and the ceremony is
      // selected by whether this caller is already authenticated.
      final rawOptions = await _client.passkeyLoginBeginWithEmail();
      final prepared = prepareRequestOptions(rawOptions);
      final credential = await authenticator.authenticate(prepared);
      final res = await _client.passkeyLoginFinishWithCredential(credential);
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

  /// Verify the email-confirmation OTP.
  Future<void> verifyEmail(String token) async {
    await _client.verifyEmail(body: VerifyEmailRequest(token: token));
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
      // ignore: deprecated_member_use_from_same_package
      return (_state as AuthMfaRequired).session?.sessionToken;
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
    _events.close();
  }

  // ── Internals ─────────────────────────────────────

  Future<ClientConfig> _doFetchClientConfig() async {
    // Check storage cache.
    try {
      final cached = await _storage.getItem(_configCacheKey);
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
        _configCacheKey,
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
    // Best-effort: a bad `expires_at` from the backend must NOT undo the
    // authentication we just established. Schedule failures are silently
    // swallowed; the next manual [refreshNow] call (or app restart) will
    // try again.
    try {
      _scheduleRefresh(session);
    } catch (_) {
      // Swallowed by design — see contract above.
    }
  }

  /// Refreshes the session.
  ///
  /// Defensive: never destroys the session on failure — the refresh token
  /// may still be valid and the server could be temporarily unreachable.
  /// Schedules a 30-second retry instead. Only [signOut] clears storage.
  /// Mirrors React `auth.ts` refreshSession (lines 476–492).
  Future<void> _refreshSession(String refreshToken) =>
      _exclusive(() => _doRefreshSession(refreshToken));

  Future<void> _doRefreshSession(String refreshToken) async {
    try {
      final newSession = await _client.refreshWithToken(refreshToken);
      final user = await _loadUser(newSession.sessionToken);
      await _persistSession(newSession);
      _setState(AuthAuthenticated(user: user, session: newSession));
      _scheduleRefresh(newSession);
    } catch (_) {
      _clearRefreshTimer();
      _refreshTimer = Timer(const Duration(seconds: 30), () {
        _refreshSession(refreshToken);
      });
    }
  }

  void _scheduleRefresh(Session session) {
    _clearRefreshTimer();

    // `expires_at` is opaque-string-shaped in the API but expected to be ISO
    // 8601. Some backends emit alternative shapes (Unix epoch strings,
    // RFC 1123, etc.). Treat parse failures as "schedule retry in 5 min"
    // rather than letting the FormatException bubble up and undo a
    // successful sign-in.
    final int expiresAt;
    try {
      expiresAt = DateTime.parse(session.expiresAt).millisecondsSinceEpoch;
    } on FormatException {
      _refreshTimer = Timer(const Duration(minutes: 5), () {
        _refreshSession(session.refreshToken);
      });
      return;
    }

    final delay =
        expiresAt - DateTime.now().millisecondsSinceEpoch - _refreshBeforeMs;

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
    final prev = _state;
    _state = newState;
    final events = _transitionEvents(prev, newState);
    for (final listener in _listeners) {
      try {
        listener(newState);
      } catch (_) {
        // Listener errors should not break the state machine.
      }
    }
    for (final event in events) {
      _emit(event);
    }
  }

  /// Works out which lifecycle events a state change means, and keeps the
  /// active organization in step: /v1/me reports it on every load, and
  /// leaving the authenticated state forgets it.
  List<AuthEvent> _transitionEvents(AuthState prev, AuthState next) {
    final reported = _reportedOrgId;
    _reportedOrgId = null;

    if (prev is AuthAuthenticated && next is! AuthAuthenticated) {
      _activeOrgId = null;
      return const [AuthEvent.signedOut];
    }
    if (next is! AuthAuthenticated) return const [];
    if (prev is! AuthAuthenticated) {
      _activeOrgId = reported;
      return const [AuthEvent.signedIn];
    }

    // Signing in as someone else without signing out first is a sign-out
    // and a sign-in, so nothing of the previous user's carries over.
    final prevUser = _userId(prev);
    final nextUser = _userId(next);
    if (prevUser != null && nextUser != null && prevUser != nextUser) {
      _activeOrgId = reported;
      return const [AuthEvent.signedOut, AuthEvent.signedIn];
    }

    final events = <AuthEvent>[];
    // A refresh rotates the refresh token; switching organization reissues
    // only the access token, so it doesn't count.
    if (prev.session.refreshToken != next.session.refreshToken) {
      events.add(AuthEvent.sessionRefreshed);
    }
    if (reported != null && reported != _activeOrgId) {
      _activeOrgId = reported;
      events.add(AuthEvent.orgChanged);
    }
    return events;
  }

  void _handleError(Object err) {
    final message = err is AuthClientException
        ? err.message
        : 'An unexpected error occurred';
    final code = err is AuthClientException ? err.code : null;

    if (err is AuthClientException) {
      if (err.isMfaRequired && (err.mfaTicket?.isNotEmpty ?? false)) {
        _setState(AuthMfaRequired(
          email: err.errorEmail ?? _pendingEmail ?? '',
          mfaTicket: err.mfaTicket ?? '',
          availableMethods: err.availableMfaMethods,
        ));
        return;
      }
      if (err.isEmailNotVerified) {
        _setState(AuthEmailNotVerified(
          email: err.errorEmail ?? _pendingEmail ?? '',
        ));
        _onError?.call(message, code);
        return;
      }
    }

    _setState(AuthError(error: message));
    _onError?.call(message, code);
  }
}

/// A registered plugin and, once used, what its init built (or threw).
class _PluginSlot {
  _PluginSlot(this.plugin);
  final AuthPlugin<Object> plugin;
  bool built = false;
  Object? api;
  Object? error;
  StackTrace? stack;
}

/// The [PluginContext] one plugin sees.
class _PluginContextImpl implements PluginContext {
  _PluginContextImpl(this._manager, this._name);
  final AuthManager _manager;
  final String _name;

  @override
  Future<Object?> request(
    String method,
    String path, {
    Map<String, String?>? query,
    Object? body,
  }) {
    return _manager.client.send(
      method,
      path,
      query: query,
      body: body,
      token: _manager.getSessionToken(),
    );
  }

  @override
  AuthState get state => _manager.state;

  @override
  ClientConfig? get clientConfig => _manager.clientConfig;

  @override
  bool? get isServerEnabled {
    final supported = _manager.clientConfig?.supportedPlugins;
    return supported?.contains(_name);
  }

  @override
  String? get activeOrgId => _manager.activeOrgId;

  @override
  Stream<AuthEvent> get events => _manager.events;
}
