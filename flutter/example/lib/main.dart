import 'package:authsome_flutter/authsome_flutter.dart';
import 'package:flutter/material.dart';
import 'package:flutter_dotenv/flutter_dotenv.dart';
import 'package:go_router/go_router.dart';

import 'app_config.dart';
import 'router.dart';

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();
  // Load the bundled .env asset before any AppConfig getter runs.
  // Mirrors TwinOS's pattern at apps/app-flutter/lib/main.dart.
  try {
    await dotenv.load(fileName: '.env');
  } catch (_) {
    // Missing or malformed .env — getters will fall back to dart-define
    // values and then to the localhost default.
  }
  runApp(const AuthsomeExampleApp());
}

/// Demo app for the AuthSome Flutter packages.
///
/// Canonical SDK pattern:
///   AuthProvider              ← owns the AuthNotifier
///     └ AuthRouterScope       ← builds the router ONCE, caches it
///       └ MaterialApp.router
///
/// [AuthRouterScope] is load-bearing: it stops the router from being
/// recreated on every `notifyListeners()` call. Without it, every auth
/// state transition would swap the router instance, reset the
/// navigator, and remount active pages — which causes the multi-step
/// sign-in form to snap back to the email entry after a failed signin.
class AuthsomeExampleApp extends StatelessWidget {
  /// Test-only seam: when provided, bypasses real client construction
  /// and wires the supplied notifier via [AuthProvider.test].
  final AuthNotifier? authOverride;

  const AuthsomeExampleApp({super.key, this.authOverride});

  @override
  Widget build(BuildContext context) {
    final scope = AuthRouterScope<GoRouter>(
      routerBuilder: (context, _) => buildRouter(context),
      builder: (context, router) => MaterialApp.router(
        title: 'AuthSome Demo',
        theme: ThemeData(useMaterial3: true, colorSchemeSeed: Colors.indigo),
        darkTheme: ThemeData(
          useMaterial3: true,
          brightness: Brightness.dark,
          colorSchemeSeed: Colors.indigo,
        ),
        routerConfig: router,
      ),
    );

    if (authOverride != null) {
      // ignore: invalid_use_of_visible_for_testing_member
      return AuthProvider.test(notifier: authOverride!, child: scope);
    }
    return AuthProvider(
      config: AuthConfig(
        baseUrl: AppConfig.baseUrl,
        publishableKey: AppConfig.publishableKey,
      ),
      child: scope,
    );
  }
}
