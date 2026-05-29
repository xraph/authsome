import 'package:authsome_flutter/authsome_flutter.dart';
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import 'pages/forgot_password_page.dart';
import 'pages/home_page.dart';
import 'pages/magic_link_page.dart';
import 'pages/mfa_challenge_page.dart';
import 'pages/profile_page.dart';
import 'pages/reset_password_page.dart';
import 'pages/sign_in_page.dart';
import 'pages/sign_up_page.dart';
import 'pages/verify_email_page.dart';

/// Builds the demo app's router.
///
/// Mirrors `apps/app-flutter/lib/router.dart` (TwinOS): a single boolean
/// `auth.isAuthenticated` drives the redirect, and the router is rebuilt
/// on every notify (we read the notifier via `AuthProvider.of(context)`,
/// which sets up the dependency). The router itself listens to the auth
/// notifier via `refreshListenable` and re-evaluates the active route on
/// every state change.
GoRouter buildRouter(BuildContext rootContext) {
  final auth = AuthProvider.of(rootContext);

  return GoRouter(
    initialLocation: '/',
    refreshListenable: auth,
    debugLogDiagnostics: true,
    redirect: (context, state) {
      final isAuthed = auth.isAuthenticated;
      final loc = state.matchedLocation;
      final isAuthRoute = loc == '/sign-in' ||
          loc == '/sign-up' ||
          loc == '/forgot-password' ||
          loc == '/reset-password' ||
          loc == '/magic-link' ||
          loc == '/verify-email' ||
          loc == '/mfa-challenge';

      if (!isAuthed && !isAuthRoute) return '/sign-in';
      if (isAuthed && isAuthRoute) return '/';
      return null;
    },
    routes: [
      GoRoute(path: '/', builder: (_, __) => const HomePage()),
      GoRoute(path: '/profile', builder: (_, __) => const ProfilePage()),
      GoRoute(path: '/sign-in', builder: (_, __) => const SignInPage()),
      GoRoute(path: '/sign-up', builder: (_, __) => const SignUpPage()),
      GoRoute(
        path: '/forgot-password',
        builder: (_, __) => const ForgotPasswordPage(),
      ),
      GoRoute(
        path: '/reset-password',
        builder: (_, state) => ResetPasswordPage(
          token: state.uri.queryParameters['token'] ?? '',
        ),
      ),
      GoRoute(path: '/magic-link', builder: (_, __) => const MagicLinkPage()),
      GoRoute(
        path: '/verify-email',
        builder: (_, state) => VerifyEmailPage(
          email: state.uri.queryParameters['email'],
        ),
      ),
      GoRoute(
        path: '/mfa-challenge',
        builder: (_, __) => const MfaChallengePage(),
      ),
    ],
    errorBuilder: (_, state) => Scaffold(
      appBar: AppBar(title: const Text('Not found')),
      body: Center(child: Text('No route for ${state.uri}')),
    ),
  );
}
