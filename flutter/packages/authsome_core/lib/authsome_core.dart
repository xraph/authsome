/// AuthSome authentication SDK for Dart.
///
/// Framework-agnostic core with auto-config support.
/// Use [AuthManager] to manage authentication state,
/// and [ClientConfig] for auto-discovery of enabled auth methods.
///
/// For Flutter apps, use the `authsome_flutter` package which provides
/// [AuthProvider] and [AuthNotifier] for reactive state management.
library authsome_core;

export 'src/types.dart';
export 'src/client.dart';
export 'src/auth_manager.dart';
export 'src/generated/api_client.dart'
    show AuthClient, AuthClientConfig, AuthClientException;
export 'src/generated/api_types.dart';
