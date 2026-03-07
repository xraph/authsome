/// Secure token storage implementation backed by flutter_secure_storage.
///
/// Persists authentication tokens securely using platform-specific
/// keychain/keystore implementations (iOS Keychain, Android Keystore, etc.).
library;

import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:authsome_core/authsome_core.dart';

/// [TokenStorage] backed by [FlutterSecureStorage].
///
/// Uses the platform's secure storage mechanism:
/// - iOS: Keychain Services
/// - Android: Keystore / EncryptedSharedPreferences
/// - macOS: Keychain
/// - Linux: libsecret
/// - Windows: Windows Credential Manager
class SecureTokenStorage implements TokenStorage {
  final FlutterSecureStorage _storage;

  /// Creates a [SecureTokenStorage] instance.
  ///
  /// Optionally pass a pre-configured [FlutterSecureStorage] instance.
  SecureTokenStorage({FlutterSecureStorage? storage})
      : _storage = storage ?? const FlutterSecureStorage();

  @override
  Future<String?> getItem(String key) => _storage.read(key: key);

  @override
  Future<void> setItem(String key, String value) =>
      _storage.write(key: key, value: value);

  @override
  Future<void> removeItem(String key) => _storage.delete(key: key);
}
