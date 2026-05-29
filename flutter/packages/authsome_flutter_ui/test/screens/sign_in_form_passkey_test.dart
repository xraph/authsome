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
}
