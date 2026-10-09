/// Flutter Web implementation of the browser helpers.
library;

import 'package:web/web.dart' as web;

/// Whether this platform can send the user to an external URL itself.
bool get canRedirectBrowser => true;

/// Navigates the current tab to [url].
void redirectBrowser(String url) {
  web.window.location.assign(url);
}

/// The query parameters of the page the app is running on.
Map<String, String> currentQueryParameters() {
  return Uri.parse(web.window.location.href).queryParameters;
}
