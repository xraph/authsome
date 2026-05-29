/// Factory that selects the right [PasskeyAuthenticator] for the
/// running platform via conditional imports.
///
/// Default branch is the Web implementation; native (`dart:io`) falls
/// back to the stub. **Do not** flip these to `dart.library.js_interop`
/// — that condition resolves `true` on both Web and native in Dart
/// 3.5+, so the Web impl would attempt to call browser APIs from the
/// VM and crash. `dart.library.io` is unambiguous: present on every
/// native platform (iOS, Android, macOS, Windows, Linux, Dart VM
/// tests) and absent on Web.
///
/// - Flutter Web: uses `passkey_authenticator_web.dart` driving
///   `navigator.credentials.get`.
/// - Everywhere else: [UnsupportedPasskeyAuthenticator] (reports
///   `isAvailable == false`). iOS/Android consumers can pass their own
///   implementation (e.g. backed by the `passkeys` corbado package)
///   to [SignInForm] / [AuthNotifier.signInWithPasskey].
library;

import 'package:authsome_core/authsome_core.dart';

import 'passkey_authenticator_web.dart'
    if (dart.library.io) 'passkey_authenticator_stub.dart';

/// Returns the platform-appropriate [PasskeyAuthenticator]. Cached at
/// the call site is fine — implementations are stateless.
PasskeyAuthenticator defaultPasskeyAuthenticator() =>
    const PlatformPasskeyAuthenticator();
