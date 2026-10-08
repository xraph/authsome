/// Native (iOS, Android, desktop, tests) stand-in for the browser helpers.
///
/// There is no page to redirect, so redirect-based flows need a callback
/// from the app (for example one that opens the URL with url_launcher or
/// flutter_web_auth_2).
library;

/// Whether this platform can send the user to an external URL itself.
bool get canRedirectBrowser => false;

/// Unsupported off the Web. Callers check [canRedirectBrowser] first.
void redirectBrowser(String url) {
  throw UnsupportedError(
    'Browser redirects need Flutter Web. Pass an onSSOLogin callback to '
    'open $url on this platform.',
  );
}

/// No page URL off the Web.
Map<String, String> currentQueryParameters() => const {};
