/// Pins the auto-discovery contract: when the backend's client-config
/// reports `passkey.enabled = true`, the [PasskeyLoginButton] renders
/// inside [SignInForm] without the caller having to set `showPasskey`
/// explicitly. Mirrors React `sign-in-form.tsx:96`
/// (`showPasskeyProp ?? config?.passkey?.enabled ?? false`).
library;

import 'package:authsome_flutter_ui/authsome_flutter_ui.dart';
import 'package:flutter_test/flutter_test.dart';

import '../_helpers/mock_auth_notifier.dart';
import '../_helpers/pump_auth_some_app.dart';

/// Authenticator that always reports available so the button actually
/// renders in the VM test environment. `defaultPasskeyAuthenticator()`
/// on non-Web is the stub `UnsupportedPasskeyAuthenticator`.
class _AlwaysAvailablePasskey implements PasskeyAuthenticator {
  @override
  bool get isAvailable => true;

  @override
  Future<Map<String, dynamic>> authenticate(Map<String, dynamic> options) {
    throw UnimplementedError();
  }
}

void main() {
  setUpAll(registerAuthFallbacks);

  testWidgets(
    'SignInForm renders PasskeyLoginButton when clientConfig.passkey.enabled is true',
    (tester) async {
      final mockAuth = buildIdleMock(
        clientConfig: const ClientConfig(
          passkey: PasskeyConfig(enabled: true),
        ),
      );

      await pumpAuthSomeApp(
        tester,
        child: SignInForm(
          auth: mockAuth,
          passkeyAuthenticator: _AlwaysAvailablePasskey(),
        ),
      );

      expect(find.byType(PasskeyLoginButton), findsOneWidget);
      expect(find.byType(OrDivider), findsOneWidget);
      expect(find.text('Continue with passkey'), findsOneWidget);
    },
  );

  testWidgets(
    'SignInForm does NOT render PasskeyLoginButton when passkey.enabled is false',
    (tester) async {
      final mockAuth = buildIdleMock(
        clientConfig: const ClientConfig(
          passkey: PasskeyConfig(enabled: false),
        ),
      );

      await pumpAuthSomeApp(
        tester,
        child: SignInForm(
          auth: mockAuth,
          passkeyAuthenticator: _AlwaysAvailablePasskey(),
        ),
      );

      expect(find.byType(PasskeyLoginButton), findsNothing);
    },
  );

  testWidgets(
    'explicit showPasskey:false beats clientConfig.passkey.enabled:true',
    (tester) async {
      final mockAuth = buildIdleMock(
        clientConfig: const ClientConfig(
          passkey: PasskeyConfig(enabled: true),
        ),
      );

      await pumpAuthSomeApp(
        tester,
        child: SignInForm(
          auth: mockAuth,
          showPasskey: false,
          passkeyAuthenticator: _AlwaysAvailablePasskey(),
        ),
      );

      expect(find.byType(PasskeyLoginButton), findsNothing);
    },
  );

  for (final showPasskey in <bool?>[null, true]) {
    testWidgets(
      'unavailable default authenticator leaves no divider with showPasskey=$showPasskey',
      (tester) async {
        final mockAuth = buildIdleMock(
          clientConfig: const ClientConfig(
            passkey: PasskeyConfig(enabled: true),
          ),
        );
        await pumpAuthSomeApp(
          tester,
          child: SignInForm(auth: mockAuth, showPasskey: showPasskey),
        );

        expect(find.byType(OrDivider), findsNothing);
        expect(find.byType(PasskeyLoginButton), findsNothing);
        expect(find.text('Continue'), findsOneWidget);
      },
    );
  }

  testWidgets('social login keeps its divider when passkeys are unavailable',
      (tester) async {
    final mockAuth = buildIdleMock(
      clientConfig: const ClientConfig(passkey: PasskeyConfig(enabled: true)),
    );
    await pumpAuthSomeApp(
      tester,
      child: SignInForm(
        auth: mockAuth,
        socialProviders: const [SocialProvider(id: 'google', name: 'Google')],
      ),
    );

    expect(find.byType(SocialButtons), findsOneWidget);
    expect(find.byType(OrDivider), findsOneWidget);
    expect(find.byType(PasskeyLoginButton), findsNothing);
  });

  testWidgets('replacing an available authenticator removes its divider',
      (tester) async {
    final mockAuth = buildIdleMock(
      clientConfig: const ClientConfig(passkey: PasskeyConfig(enabled: true)),
    );
    await pumpAuthSomeApp(
      tester,
      child: SignInForm(
        auth: mockAuth,
        passkeyAuthenticator: _AlwaysAvailablePasskey(),
      ),
    );
    expect(find.byType(OrDivider), findsOneWidget);
    await pumpAuthSomeApp(tester, child: SignInForm(auth: mockAuth));

    expect(find.byType(OrDivider), findsNothing);
    expect(find.byType(PasskeyLoginButton), findsNothing);
  });
}
