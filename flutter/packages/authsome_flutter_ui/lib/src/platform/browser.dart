/// Browser access for the flows that leave the app: SSO redirects and the
/// landing pages that read `?code=` back out of the URL.
///
/// Same conditional-import rule as `default_passkey_authenticator.dart` in
/// authsome_flutter: the Web file is the default and `dart.library.io`
/// picks the stub. `dart.library.js_interop` would resolve true on native
/// too and pull browser calls into the VM.
library;

export 'browser_web.dart' if (dart.library.io) 'browser_stub.dart';
