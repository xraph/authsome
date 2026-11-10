/// Client plugins: the client side of authsome's server plugins.
///
/// A server plugin is something you opt into with `authsome.WithPlugin`. Its
/// client half is the same: a package you add to [AuthConfig.plugins], which
/// then gets an authenticated way to call its routes and hears about sign-in,
/// sign-out, refreshes and organization switches. Core knows nothing about
/// any particular plugin.
///
/// ```dart
/// final auth = AuthManager(AuthConfig(
///   baseUrl: 'https://api.example.com',
///   plugins: [SubscriptionPlugin()],
/// ));
/// final billing = auth.plugin<SubscriptionApi>();
/// ```
library;

import 'types.dart';

/// Moments in the session a plugin can react to.
enum AuthEvent {
  /// The session became authenticated: sign-in, or a session restored on load.
  signedIn,

  /// The session stopped being authenticated.
  signedOut,

  /// The session got new tokens (a refresh). Switching organization doesn't
  /// count; it fires [orgChanged].
  sessionRefreshed,

  /// The session's active organization changed.
  orgChanged,
}

/// What a plugin gets from the [AuthManager]. Kept deliberately small: once
/// plugin packages version on their own, every member here is public API.
abstract interface class PluginContext {
  /// Calls the API as the signed-in user: base URL, publishable key and the
  /// session's bearer token are filled in, and query values that are null
  /// are left out. Returns the decoded JSON body (null for an empty one).
  /// Throws [AuthClientException] on an error status.
  Future<Object?> request(
    String method,
    String path, {
    Map<String, String?>? query,
    Object? body,
  });

  /// The current auth state.
  AuthState get state;

  /// The client config, null until it has loaded.
  ClientConfig? get clientConfig;

  /// Whether the server has this plugin installed, from the client config's
  /// supported plugins. Null while that isn't known.
  bool? get isServerEnabled;

  /// The session's active organization, null when it has none.
  String? get activeOrgId;

  /// Lifecycle events.
  Stream<AuthEvent> get events;
}

/// A client plugin. [name] must be the server plugin's name, so the plugin
/// can tell from the client config whether the server side is installed.
///
/// [init] runs once, the first time the plugin is looked up with
/// [AuthManager.plugin], and returns the plugin's API.
abstract class AuthPlugin<Api extends Object> {
  const AuthPlugin();

  /// The server plugin's name, e.g. `subscription`.
  String get name;

  /// Builds the plugin's API.
  Api init(PluginContext ctx);
}

/// Thrown when code asks for a plugin that wasn't added to
/// [AuthConfig.plugins].
class PluginNotRegisteredError extends StateError {
  PluginNotRegisteredError(Type api)
      : super('authsome: no registered plugin provides $api. '
            'Add it to AuthConfig.plugins.');
}
