/// AuthSome authentication SDK for Flutter.
///
/// Provides [AuthProvider] for widget tree integration,
/// [AuthNotifier] for reactive state management,
/// and [SecureTokenStorage] for secure token persistence.
///
/// Usage:
/// ```dart
/// import 'package:authsome_flutter/authsome_flutter.dart';
///
/// AuthProvider(
///   config: AuthConfig(
///     baseUrl: 'https://api.example.com',
///     publishableKey: 'pk_...',
///   ),
///   child: MyApp(),
/// )
/// ```
///
/// Access auth state anywhere in the tree:
/// ```dart
/// final auth = context.auth;
/// if (auth.isAuthenticated) { ... }
/// ```
library authsome_flutter;

// Re-export core types for convenience.
export 'package:authsome_core/authsome_core.dart';

// Flutter-specific exports.
export 'src/auth_provider.dart';
export 'src/secure_token_storage.dart';
