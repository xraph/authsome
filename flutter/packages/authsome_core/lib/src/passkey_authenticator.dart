/// Abstract contract for the platform half of a passkey ceremony.
///
/// Implementations live in `authsome_flutter` (Web via `dart:js_interop`,
/// iOS/Android via the `passkeys` corbado package). The core SDK only
/// knows how to call `/v1/passkeys/login/begin` and
/// `/v1/passkeys/login/finish` — it delegates the WebAuthn ceremony
/// (talking to the OS / browser to actually get a credential) to an
/// authenticator implementation that the embedding app injects.
library;

/// A platform-specific WebAuthn / passkey authenticator.
abstract class PasskeyAuthenticator {
  /// Whether the current platform has WebAuthn / passkey support
  /// available. The [PasskeyLoginButton] hides itself when this is
  /// false, so server-side auto-config never tries to surface a button
  /// on a platform that can't honour it.
  bool get isAvailable;

  /// Run the assertion ceremony for the given options.
  ///
  /// [options] is the parsed JSON map returned by
  /// `/v1/passkeys/login/begin` with all base64url binary fields already
  /// converted to [Uint8List] via `prepareRequestOptions`. The return
  /// value is the credential payload to POST to
  /// `/v1/passkeys/login/finish`, with binary fields re-encoded as
  /// base64url strings (via `serializeAssertion`).
  Future<Map<String, dynamic>> authenticate(Map<String, dynamic> options);
}

/// Default authenticator: reports `isAvailable == false` and throws on
/// every call. Used as the fallback when no platform implementation is
/// provided.
class UnsupportedPasskeyAuthenticator implements PasskeyAuthenticator {
  const UnsupportedPasskeyAuthenticator();

  @override
  bool get isAvailable => false;

  @override
  Future<Map<String, dynamic>> authenticate(Map<String, dynamic> options) {
    throw UnsupportedError(
      'No PasskeyAuthenticator is wired up. Provide one when calling '
      'AuthManager.signInWithPasskey, or pass `authenticator:` to '
      'PasskeyLoginButton.',
    );
  }
}
