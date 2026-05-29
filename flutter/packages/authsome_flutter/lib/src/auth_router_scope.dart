/// [AuthRouterScope] — build your router once and keep it stable across
/// auth state changes.
///
/// The footgun this widget exists to close:
///
/// ```dart
/// // ❌ DON'T. Calls AuthProvider.of(context) inside build, so this
/// // widget re-runs on every notifyListeners(). MaterialApp.router
/// // receives a fresh GoRouter each time, swaps its navigator, and
/// // remounts active pages — multi-step sign-in forms snap back to
/// // step 1 after a failed network request.
/// class MyApp extends StatelessWidget {
///   Widget build(BuildContext context) {
///     final router = buildRouter(AuthProvider.of(context));
///     return MaterialApp.router(routerConfig: router);
///   }
/// }
/// ```
///
/// The fix is to construct the router exactly once and cache it, then
/// re-use the cached instance on every rebuild. [GoRouter]'s own
/// `refreshListenable` already takes care of re-evaluating redirects on
/// auth state changes — recreating the router itself is what breaks the
/// navigator. [AuthRouterScope] does the caching for you:
///
/// ```dart
/// // ✅ DO. AuthRouterScope builds the router once, hands the cached
/// // instance to every rebuild.
/// AuthRouterScope<GoRouter>(
///   routerBuilder: (context, auth) => buildRouter(auth),
///   builder: (context, router) => MaterialApp.router(
///     routerConfig: router,
///   ),
/// )
/// ```
library;

import 'package:flutter/widgets.dart';

import 'auth_provider.dart';

/// Caches a value (typically a router) built from the resolved
/// [AuthNotifier]. The cached value's identity is stable across the
/// widget's lifetime, even when the notifier fires `notifyListeners()`.
///
/// Generic in [T] so it works with any router framework — [GoRouter],
/// `BeamerDelegate`, a hand-rolled [RouterDelegate], etc.
class AuthRouterScope<T extends Object> extends StatefulWidget {
  /// Called exactly once, the first time the [AuthNotifier] becomes
  /// available via the surrounding [AuthProvider]. The returned value is
  /// cached for the widget's lifetime.
  ///
  /// Use it to construct your router so it keeps a stable identity
  /// across auth state notifications.
  final T Function(BuildContext context, AuthNotifier auth) routerBuilder;

  /// Renders the rest of the app, receiving the cached router on every
  /// rebuild. Typically wraps the value in `MaterialApp.router`.
  final Widget Function(BuildContext context, T router) builder;

  const AuthRouterScope({
    super.key,
    required this.routerBuilder,
    required this.builder,
  });

  @override
  State<AuthRouterScope<T>> createState() => _AuthRouterScopeState<T>();
}

class _AuthRouterScopeState<T extends Object>
    extends State<AuthRouterScope<T>> {
  T? _router;

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    _router ??= widget.routerBuilder(context, AuthProvider.of(context));
  }

  @override
  Widget build(BuildContext context) {
    final router = _router;
    if (router == null) {
      // didChangeDependencies runs before build, so this branch is
      // defensive only.
      return const SizedBox.shrink();
    }
    return widget.builder(context, router);
  }
}
