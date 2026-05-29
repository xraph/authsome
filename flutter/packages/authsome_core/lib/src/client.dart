/// AuthSome client adapter — thin wrapper over the auto-generated API client.
///
/// The generated client lives in `generated/api_client.dart` and covers all
/// core + plugin endpoints. This file extends it with convenience bridge
/// methods so that [AuthManager] (and downstream consumers) keep a simple API.
library;

import 'dart:convert';

import 'package:http/http.dart' as http;

import 'generated/api_client.dart' as generated;
import 'generated/api_client.dart' show AuthClientConfig, AuthClientException;
import 'types.dart';

// Re-exports for convenience.
export 'generated/api_client.dart' show AuthClientConfig, AuthClientException;

/// AuthSomeClient extends the auto-generated client with backward-compatible
/// convenience methods that [AuthManager] depends on.
///
/// All 80+ generated endpoints are inherited as-is. Only a few methods
/// are overridden to preserve the simpler call-site signatures used by
/// the auth state machine.
class AuthSomeClient extends generated.AuthClient {
  final String _rawBaseUrl;
  final http.Client _http;

  AuthSomeClient._(this._rawBaseUrl, this._http, AuthClientConfig config)
      : super(config);

  factory AuthSomeClient(AuthClientConfig config) {
    final baseUrl = config.baseUrl.replaceAll(RegExp(r'/+$'), '');
    final httpClient = config.httpClient ?? http.Client();
    return AuthSomeClient._(
      baseUrl,
      httpClient,
      // Forward every field — dropping `publishableKey` here was the
      // bug that made TwinOS-style backends reject sign-in with
      // "app context required: send publishable key".
      AuthClientConfig(
        baseUrl: config.baseUrl,
        httpClient: httpClient,
        publishableKey: config.publishableKey,
      ),
    );
  }

  /// Convenience constructor from base URL string.
  factory AuthSomeClient.fromUrl(String baseUrl) {
    return AuthSomeClient(AuthClientConfig(baseUrl: baseUrl));
  }

  // ── Bridge methods for AuthManager ──────────────────

  /// Get current user profile using a session token.
  Future<User> getMeWithToken(String token) {
    return super.getMe(token: token);
  }

  /// Sign in with email & password.
  Future<AuthResponse> signInWithCredentials({
    required String email,
    required String password,
  }) {
    return super.signIn(body: {'email': email, 'password': password});
  }

  /// Sign up with email & password.
  Future<AuthResponse> signUpWithCredentials({
    required String email,
    required String password,
    String? name,
  }) {
    final body = <String, dynamic>{
      'email': email,
      'password': password,
    };
    if (name != null) body['name'] = name;
    return super.signUp(body: body);
  }

  /// Refresh session tokens using a refresh token string.
  ///
  /// The codegen aliases this endpoint's response to the OAuth-shaped
  /// TokenResponse (access_token/expires_in/token_type) because of a Go
  /// struct-name collision, but the real backend returns a session-shaped
  /// payload (session_token/refresh_token/expires_at). We make a raw HTTP
  /// call and parse the real shape, mirroring [mfaChallenge].
  Future<Session> refreshWithToken(String refreshToken) async {
    final data = await _rawPost(
      '/v1/refresh',
      body: {'refresh_token': refreshToken},
    );
    return Session.fromJson(data);
  }

  /// Sign out with a token string.
  Future<void> signOutWithToken(String token) async {
    await super.signOut(body: const SignOutRequest(), token: token);
  }

  /// MFA challenge — bridge for auth_manager.dart.
  ///
  /// The server returns a full auth response (with session tokens and user)
  /// when the challenge passes, but the OpenAPI spec only models the
  /// verification-specific fields. We make a raw HTTP call to capture the
  /// full response.
  Future<AuthResponse> mfaChallenge({
    String? enrollmentId,
    required String code,
  }) async {
    final body = <String, dynamic>{'code': code};
    if (enrollmentId != null) body['enrollment_id'] = enrollmentId;
    final data = await _rawPost('/v1/mfa/challenge', body: body);
    return AuthResponse.fromJson(data);
  }

  /// Verify an MFA recovery code.
  ///
  /// Like [mfaChallenge], returns a full auth response from the server.
  Future<AuthResponse> verifyRecoveryCodeWithString(String code) async {
    final data = await _rawPost(
      '/v1/mfa/recovery/verify',
      body: {'code': code},
    );
    return AuthResponse.fromJson(data);
  }

  /// Send an SMS code for MFA.
  Future<SMSSendResponse> sendSMSCodeForMFA(String token) {
    return super.sendSMSCode(body: const SMSSendRequest(), token: token);
  }

  /// Verify an SMS code for MFA.
  ///
  /// Like [mfaChallenge], returns a full auth response from the server.
  Future<AuthResponse> verifySMSCodeForMFA(String code, String token) async {
    final data = await _rawPost(
      '/v1/mfa/sms/verify',
      body: {'code': code},
      token: token,
    );
    return AuthResponse.fromJson(data);
  }

  /// Begin a passkey login ceremony. Returns the raw options map (with
  /// base64url-encoded binary fields) as the server provides it — the
  /// caller is responsible for decoding it via [prepareRequestOptions]
  /// before handing it to a [PasskeyAuthenticator].
  Future<Map<String, dynamic>> passkeyLoginBeginWithEmail({
    String? email,
  }) async {
    final res = await super.passkeyLoginBegin(
      body: LoginBeginRequest(email: email),
    );
    return Map<String, dynamic>.from(res.options);
  }

  /// Finish a passkey login by POSTing the serialized credential to
  /// `/v1/passkeys/login/finish`. Bypasses the generated empty
  /// [LoginFinishRequest] (a codegen quirk) so the credential map
  /// actually reaches the wire.
  Future<AuthResponse> passkeyLoginFinishWithCredential(
    Map<String, dynamic> credential,
  ) async {
    final data = await _rawPost(
      '/v1/passkeys/login/finish',
      body: credential,
    );
    return AuthResponse.fromJson(data);
  }

  /// Fetch client configuration from the backend.
  ///
  /// The config describes which auth methods are enabled so SDK
  /// components can auto-configure without manual props.
  Future<ClientConfig> fetchClientConfig([String? publishableKey]) async {
    final uri = Uri.parse('$_rawBaseUrl/v1/client-config');
    final queryUri = publishableKey != null
        ? uri.replace(queryParameters: {'key': publishableKey})
        : uri;

    final response = await _http.get(
      queryUri,
      headers: {
        'Content-Type': 'application/json',
        'Accept': 'application/json',
      },
    );

    if (response.statusCode < 200 || response.statusCode >= 300) {
      throw AuthClientException(
        'Failed to fetch client config',
        code: response.statusCode,
      );
    }

    final body = jsonDecode(response.body) as Map<String, dynamic>;
    return ClientConfig.fromJson(body);
  }

  // ── Internal helpers ──────────────────────────────────

  /// Make a raw POST request and return the parsed JSON body.
  ///
  /// Used by MFA bridge methods where the server returns more data
  /// than the OpenAPI spec models.
  Future<Map<String, dynamic>> _rawPost(
    String path, {
    Map<String, dynamic>? body,
    String? token,
  }) async {
    final headers = <String, String>{
      'Content-Type': 'application/json',
      'Accept': 'application/json',
    };
    if (token != null) {
      headers['Authorization'] = 'Bearer $token';
    }
    // Mirror the publishable-key header behaviour of the generated
    // request helper — public auth endpoints (passkey login begin/finish)
    // reach the wire via _rawPost and would otherwise lose app context.
    final pk = publishableKey;
    if (pk != null && pk.isNotEmpty) {
      headers['X-Publishable-Key'] = pk;
    }

    final response = await _http.post(
      Uri.parse('$_rawBaseUrl$path'),
      headers: headers,
      body: body != null ? jsonEncode(body) : null,
    );

    if (response.statusCode < 200 || response.statusCode >= 300) {
      String errorMessage;
      try {
        final errorBody =
            jsonDecode(response.body) as Map<String, dynamic>;
        errorMessage = (errorBody['error'] as String?) ??
            'Request failed with status ${response.statusCode}';
      } catch (_) {
        errorMessage = 'Request failed with status ${response.statusCode}';
      }
      throw AuthClientException(errorMessage, code: response.statusCode);
    }

    return jsonDecode(response.body) as Map<String, dynamic>;
  }
}
