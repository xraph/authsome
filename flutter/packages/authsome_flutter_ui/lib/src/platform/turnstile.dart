/// Cloudflare Turnstile for Flutter Web, and a stub everywhere else.
///
/// See `browser.dart` for why the condition is `dart.library.io`.
library;

export 'turnstile_web.dart' if (dart.library.io) 'turnstile_stub.dart';
