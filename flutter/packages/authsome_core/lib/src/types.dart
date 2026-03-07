/// Core type definitions for the AuthSome Dart SDK.
///
/// Ports the TypeScript types from `ui/packages/core/src/types.ts`.
library;

// Re-export generated API types.
export 'generated/api_types.dart';

// ── Session ──────────────────────────────────────────

/// Session tokens returned after authentication / refresh.
class Session {
  final String sessionToken;
  final String refreshToken;
  final String expiresAt;

  const Session({
    required this.sessionToken,
    required this.refreshToken,
    required this.expiresAt,
  });

  factory Session.fromJson(Map<String, dynamic> json) {
    return Session(
      sessionToken: json['session_token'] as String,
      refreshToken: json['refresh_token'] as String,
      expiresAt: json['expires_at'] as String,
    );
  }

  Map<String, dynamic> toJson() => {
        'session_token': sessionToken,
        'refresh_token': refreshToken,
        'expires_at': expiresAt,
      };
}

// ── Auth State ───────────────────────────────────────

/// Authentication state (sealed class for exhaustive matching).
sealed class AuthState {
  const AuthState();
}

/// Initial idle state before initialization.
class AuthIdle extends AuthState {
  const AuthIdle();
}

/// Auth is loading (sign-in, sign-up, refresh, etc.).
class AuthLoading extends AuthState {
  const AuthLoading();
}

/// User is authenticated.
class AuthAuthenticated extends AuthState {
  final dynamic user; // Generated User type from api_types.dart
  final Session session;

  const AuthAuthenticated({required this.user, required this.session});
}

/// User is not authenticated.
class AuthUnauthenticated extends AuthState {
  const AuthUnauthenticated();
}

/// MFA challenge required before full authentication.
class AuthMfaRequired extends AuthState {
  final Session session;

  const AuthMfaRequired({required this.session});
}

/// An error occurred during authentication.
class AuthError extends AuthState {
  final String error;

  const AuthError({required this.error});
}

// ── Client Config (auto-discovery) ─────────────────

/// Social provider info from the backend.
class SocialProviderConfig {
  final String id;
  final String name;

  const SocialProviderConfig({required this.id, required this.name});

  factory SocialProviderConfig.fromJson(Map<String, dynamic> json) {
    return SocialProviderConfig(
      id: json['id'] as String,
      name: json['name'] as String,
    );
  }

  Map<String, dynamic> toJson() => {'id': id, 'name': name};
}

/// SSO connection info from the backend.
class SSOConnectionConfig {
  final String id;
  final String name;

  const SSOConnectionConfig({required this.id, required this.name});

  factory SSOConnectionConfig.fromJson(Map<String, dynamic> json) {
    return SSOConnectionConfig(
      id: json['id'] as String,
      name: json['name'] as String,
    );
  }

  Map<String, dynamic> toJson() => {'id': id, 'name': name};
}

/// Client configuration returned by the backend.
///
/// Describes which auth methods are enabled so SDK components
/// can auto-configure their UI without manual props.
class ClientConfig {
  final String? version;
  final String? appId;
  final BrandingConfig? branding;
  final PasswordConfig? password;
  final SocialConfig? social;
  final PasskeyConfig? passkey;
  final MfaConfig? mfa;
  final MagicLinkConfig? magiclink;
  final SsoConfig? sso;
  final List<String>? supportedPlugins;

  const ClientConfig({
    this.version,
    this.appId,
    this.branding,
    this.password,
    this.social,
    this.passkey,
    this.mfa,
    this.magiclink,
    this.sso,
    this.supportedPlugins,
  });

  factory ClientConfig.fromJson(Map<String, dynamic> json) {
    return ClientConfig(
      version: json['version'] as String?,
      appId: json['app_id'] as String?,
      branding: json['branding'] != null
          ? BrandingConfig.fromJson(
              Map<String, dynamic>.from(json['branding'] as Map))
          : null,
      password: json['password'] != null
          ? PasswordConfig.fromJson(
              Map<String, dynamic>.from(json['password'] as Map))
          : null,
      social: json['social'] != null
          ? SocialConfig.fromJson(
              Map<String, dynamic>.from(json['social'] as Map))
          : null,
      passkey: json['passkey'] != null
          ? PasskeyConfig.fromJson(
              Map<String, dynamic>.from(json['passkey'] as Map))
          : null,
      mfa: json['mfa'] != null
          ? MfaConfig.fromJson(
              Map<String, dynamic>.from(json['mfa'] as Map))
          : null,
      magiclink: json['magiclink'] != null
          ? MagicLinkConfig.fromJson(
              Map<String, dynamic>.from(json['magiclink'] as Map))
          : null,
      sso: json['sso'] != null
          ? SsoConfig.fromJson(
              Map<String, dynamic>.from(json['sso'] as Map))
          : null,
      supportedPlugins: (json['supported_plugins'] as List<dynamic>?)
          ?.map((e) => e as String)
          .toList(),
    );
  }

  Map<String, dynamic> toJson() => {
        if (version != null) 'version': version,
        if (appId != null) 'app_id': appId,
        if (branding != null) 'branding': branding!.toJson(),
        if (password != null) 'password': password!.toJson(),
        if (social != null) 'social': social!.toJson(),
        if (passkey != null) 'passkey': passkey!.toJson(),
        if (mfa != null) 'mfa': mfa!.toJson(),
        if (magiclink != null) 'magiclink': magiclink!.toJson(),
        if (sso != null) 'sso': sso!.toJson(),
        if (supportedPlugins != null) 'supported_plugins': supportedPlugins,
      };
}

class BrandingConfig {
  final String? appName;
  final String? logoUrl;

  const BrandingConfig({this.appName, this.logoUrl});

  factory BrandingConfig.fromJson(Map<String, dynamic> json) {
    return BrandingConfig(
      appName: json['app_name'] as String?,
      logoUrl: json['logo_url'] as String?,
    );
  }

  Map<String, dynamic> toJson() => {
        if (appName != null) 'app_name': appName,
        if (logoUrl != null) 'logo_url': logoUrl,
      };
}

class PasswordConfig {
  final bool enabled;

  const PasswordConfig({required this.enabled});

  factory PasswordConfig.fromJson(Map<String, dynamic> json) {
    return PasswordConfig(enabled: json['enabled'] as bool);
  }

  Map<String, dynamic> toJson() => {'enabled': enabled};
}

class SocialConfig {
  final bool enabled;
  final List<SocialProviderConfig> providers;

  const SocialConfig({required this.enabled, required this.providers});

  factory SocialConfig.fromJson(Map<String, dynamic> json) {
    return SocialConfig(
      enabled: json['enabled'] as bool,
      providers: (json['providers'] as List<dynamic>)
          .map((e) =>
              SocialProviderConfig.fromJson(Map<String, dynamic>.from(e as Map)))
          .toList(),
    );
  }

  Map<String, dynamic> toJson() => {
        'enabled': enabled,
        'providers': providers.map((p) => p.toJson()).toList(),
      };
}

class PasskeyConfig {
  final bool enabled;

  const PasskeyConfig({required this.enabled});

  factory PasskeyConfig.fromJson(Map<String, dynamic> json) {
    return PasskeyConfig(enabled: json['enabled'] as bool);
  }

  Map<String, dynamic> toJson() => {'enabled': enabled};
}

class MfaConfig {
  final bool enabled;
  final List<String> methods;

  const MfaConfig({required this.enabled, required this.methods});

  factory MfaConfig.fromJson(Map<String, dynamic> json) {
    return MfaConfig(
      enabled: json['enabled'] as bool,
      methods: (json['methods'] as List<dynamic>)
          .map((e) => e as String)
          .toList(),
    );
  }

  Map<String, dynamic> toJson() => {'enabled': enabled, 'methods': methods};
}

class MagicLinkConfig {
  final bool enabled;

  const MagicLinkConfig({required this.enabled});

  factory MagicLinkConfig.fromJson(Map<String, dynamic> json) {
    return MagicLinkConfig(enabled: json['enabled'] as bool);
  }

  Map<String, dynamic> toJson() => {'enabled': enabled};
}

class SsoConfig {
  final bool enabled;
  final List<SSOConnectionConfig> connections;

  const SsoConfig({required this.enabled, required this.connections});

  factory SsoConfig.fromJson(Map<String, dynamic> json) {
    return SsoConfig(
      enabled: json['enabled'] as bool,
      connections: (json['connections'] as List<dynamic>)
          .map((e) =>
              SSOConnectionConfig.fromJson(Map<String, dynamic>.from(e as Map)))
          .toList(),
    );
  }

  Map<String, dynamic> toJson() => {
        'enabled': enabled,
        'connections': connections.map((c) => c.toJson()).toList(),
      };
}

// ── Auth Config ──────────────────────────────────────

/// Configuration for the AuthSome client.
class AuthConfig {
  /// Base URL of the AuthSome API (e.g., "https://api.example.com").
  final String baseUrl;

  /// Publishable key for auto-discovering enabled auth methods.
  final String? publishableKey;

  /// Pre-fetched client config (useful for SSR / startup optimization).
  final ClientConfig? initialClientConfig;

  /// Storage implementation for persisting tokens.
  /// Defaults to in-memory storage if not provided.
  final TokenStorage? storage;

  /// Callback invoked when the auth state changes.
  final void Function(AuthState state)? onStateChange;

  /// Callback invoked on authentication error.
  final void Function(String error, int? code)? onError;

  const AuthConfig({
    required this.baseUrl,
    this.publishableKey,
    this.initialClientConfig,
    this.storage,
    this.onStateChange,
    this.onError,
  });
}

// ── Token Storage ────────────────────────────────────

/// Interface for persisting tokens across sessions.
abstract class TokenStorage {
  Future<String?> getItem(String key);
  Future<void> setItem(String key, String value);
  Future<void> removeItem(String key);
}

/// In-memory token storage (lost on app restart).
class MemoryTokenStorage implements TokenStorage {
  final Map<String, String> _store = {};

  @override
  Future<String?> getItem(String key) async => _store[key];

  @override
  Future<void> setItem(String key, String value) async {
    _store[key] = value;
  }

  @override
  Future<void> removeItem(String key) async {
    _store.remove(key);
  }
}

// ── SMS result type ──────────────────────────────────

/// Result of sending an SMS code for MFA.
class SmsSendResult {
  final bool sent;
  final String phoneMasked;
  final int expiresInSeconds;

  const SmsSendResult({
    required this.sent,
    required this.phoneMasked,
    required this.expiresInSeconds,
  });

  factory SmsSendResult.fromJson(Map<String, dynamic> json) {
    return SmsSendResult(
      sent: json['sent'] as bool,
      phoneMasked: json['phone_masked'] as String,
      expiresInSeconds: (json['expires_in_seconds'] as num).toInt(),
    );
  }
}
