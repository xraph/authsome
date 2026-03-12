/// Auth guard widget — conditionally renders children based on auth state.
///
/// A headless widget that shows [child] when the user is authenticated,
/// a [loading] widget during initialization, and a [fallback] widget
/// when unauthenticated. Uses the sealed [AuthState] for exhaustive
/// pattern matching.
///
/// ```dart
/// AuthGuard(
///   loading: const Center(child: CircularProgressIndicator()),
///   fallback: const LoginScreen(),
///   child: const HomeScreen(),
/// )
/// ```
library;

import 'package:flutter/material.dart';
import 'package:authsome_flutter/authsome_flutter.dart';

/// A widget that gates access to [child] based on the current auth state.
///
/// - When the user is authenticated, [child] is rendered.
/// - During loading or initial idle state, [loading] is rendered
///   (defaults to a centered [CircularProgressIndicator]).
/// - In all other states (unauthenticated, error, MFA required),
///   [fallback] is rendered (defaults to [SizedBox.shrink]).
class AuthGuard extends StatelessWidget {
  /// The widget to display when the user is authenticated.
  final Widget child;

  /// The widget to display when the user is not authenticated.
  ///
  /// Defaults to [SizedBox.shrink] if not provided.
  final Widget? fallback;

  /// The widget to display while auth state is loading or idle.
  ///
  /// Defaults to a centered [CircularProgressIndicator] if not provided.
  final Widget? loading;

  /// Creates an [AuthGuard].
  const AuthGuard({
    required this.child,
    this.fallback,
    this.loading,
    super.key,
  });

  @override
  Widget build(BuildContext context) {
    final auth = context.auth;
    final state = auth.state;

    return switch (state) {
      AuthAuthenticated() => child,
      AuthLoading() || AuthIdle() =>
        loading ?? const Center(child: CircularProgressIndicator()),
      AuthUnauthenticated() || AuthError() || AuthMfaRequired() =>
        fallback ?? const SizedBox.shrink(),
    };
  }
}
