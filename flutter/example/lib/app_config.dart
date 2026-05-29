import 'package:flutter_dotenv/flutter_dotenv.dart';

/// Runtime configuration for the demo app.
///
/// Each getter consults three sources in order, mirroring the TwinOS
/// `AppEnv` pattern (apps/app-flutter/lib/config/env.dart):
///   1. A loaded `.env` file (via [dotenv]) — bundled as an asset and
///      loaded in `main()`.
///   2. A `--dart-define` value baked at compile time.
///   3. A hardcoded fallback default.
///
/// dart-defines win over `.env` only when explicitly set at build time;
/// an empty dart-define falls through to the `.env` value. Edit
/// `flutter/example/.env` for your local machine.
class AppConfig {
  static String _read(
    String key,
    String dartDefine, {
    String defaultValue = '',
  }) {
    String? fromEnv;
    try {
      fromEnv = dotenv.isInitialized ? dotenv.maybeGet(key) : null;
    } catch (_) {
      fromEnv = null;
    }
    if (fromEnv != null && fromEnv.isNotEmpty) return fromEnv;
    if (dartDefine.isNotEmpty) return dartDefine;
    return defaultValue;
  }

  /// Base URL of the AuthSome HTTP API. For TwinOS-style deployments this
  /// is the studio gateway with `/identity/authsome` appended, e.g.
  /// `http://192.168.4.153:7903/identity/authsome`.
  static String get baseUrl => _read(
        'AUTHSOME_BASE_URL',
        const String.fromEnvironment('AUTHSOME_BASE_URL'),
        defaultValue: 'http://localhost:8080',
      );

  /// Publishable key (`pk_live_…` / `pk_test_…`). When set, AuthManager
  /// fetches `/v1/client-config` on startup to auto-discover enabled
  /// social/passkey/captcha methods.
  static String? get publishableKey {
    final v = _read(
      'AUTHSOME_PUBLISHABLE_KEY',
      const String.fromEnvironment('AUTHSOME_PUBLISHABLE_KEY'),
    );
    return v.isEmpty ? null : v;
  }
}
