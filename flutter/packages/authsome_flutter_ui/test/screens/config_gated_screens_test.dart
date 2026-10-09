/// Covers the screens added for config parity with React: waitlist, device
/// authorization and the SSO landing page.
library;

import 'package:authsome_flutter_ui/authsome_flutter_ui.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mocktail/mocktail.dart';

import '../_helpers/mock_auth_notifier.dart';
import '../_helpers/pump_auth_some_app.dart';

const _session = Session(
  sessionToken: 's',
  refreshToken: 'r',
  expiresAt: '2099-01-01T00:00:00Z',
);

void main() {
  setUpAll(registerAuthFallbacks);

  group('WaitlistForm', () {
    testWidgets('renders nothing when the waitlist is disabled',
        (tester) async {
      final mockAuth = buildIdleMock(
        clientConfig: const ClientConfig(waitlist: ToggleConfig(enabled: false)),
      );

      await pumpAuthSomeApp(tester, child: WaitlistForm(auth: mockAuth));

      expect(find.byType(AuthCard), findsNothing);
    });

    testWidgets('joins the waitlist and shows the success state',
        (tester) async {
      final mockAuth = buildIdleMock(
        clientConfig: const ClientConfig(waitlist: ToggleConfig(enabled: true)),
      );
      var succeeded = false;

      await pumpAuthSomeApp(
        tester,
        child: WaitlistForm(auth: mockAuth, onSuccess: () => succeeded = true),
      );
      await tester.enterText(
          find.widgetWithText(TextField, 'Email'), 'w@example.com');
      await tester.enterText(
          find.widgetWithText(TextField, 'Name (optional)'), 'Wes');
      await tester.tap(find.widgetWithText(FilledButton, 'Join waitlist'));
      await tester.pumpAndSettle();

      verify(() => mockAuth.joinWaitlist('w@example.com', name: 'Wes'))
          .called(1);
      expect(succeeded, isTrue);
      expect(find.text("You're on the list!"), findsOneWidget);
    });
  });

  group('DeviceAuthorizationForm', () {
    testWidgets('says device sign-in is unavailable when disabled',
        (tester) async {
      final mockAuth = buildIdleMock(
        clientConfig: const ClientConfig(
          deviceAuthorization: ToggleConfig(enabled: false),
        ),
      );

      await pumpAuthSomeApp(
        tester,
        child: DeviceAuthorizationForm(auth: mockAuth, initialCode: 'ABCD-EFGH'),
      );
      await tester.pumpAndSettle();

      expect(find.text('Device sign-in unavailable'), findsOneWidget);
      verifyNever(() => mockAuth.approveDeviceAuthorization(any()));
    });

    testWidgets('auto-submits a pre-filled code for a signed-in user',
        (tester) async {
      final mockAuth = buildIdleMock(
        state: const AuthAuthenticated(user: null, session: _session),
      );

      await pumpAuthSomeApp(
        tester,
        child: DeviceAuthorizationForm(auth: mockAuth, initialCode: 'abcd-efgh'),
      );
      await tester.pumpAndSettle();

      verify(() => mockAuth.approveDeviceAuthorization('ABCDEFGH')).called(1);
      expect(find.text('Device authorized'), findsOneWidget);
    });

    testWidgets('asks a signed-out user to sign in instead of submitting',
        (tester) async {
      final mockAuth = buildIdleMock(state: const AuthUnauthenticated());

      await pumpAuthSomeApp(
        tester,
        child: DeviceAuthorizationForm(auth: mockAuth, initialCode: 'ABCDEFGH'),
      );
      await tester.pumpAndSettle();

      expect(find.text('Sign in to authorize this device.'), findsOneWidget);
      verifyNever(() => mockAuth.approveDeviceAuthorization(any()));
    });
  });

  group('SSOCallbackForm', () {
    testWidgets('exchanges the code once, then calls onSuccess',
        (tester) async {
      final mockAuth = buildIdleMock();
      var succeeded = 0;

      await pumpAuthSomeApp(
        tester,
        child: SSOCallbackForm(
          auth: mockAuth,
          code: 'otc_1',
          onSuccess: () => succeeded++,
        ),
      );
      await tester.pumpAndSettle();

      verify(() => mockAuth.completeSSOLogin('otc_1')).called(1);
      expect(succeeded, 1);
    });

    testWidgets('shows the server error from sso_error', (tester) async {
      final mockAuth = buildIdleMock();

      await pumpAuthSomeApp(
        tester,
        child: SSOCallbackForm(auth: mockAuth, error: 'assertion_invalid'),
      );
      await tester.pumpAndSettle();

      expect(find.text('Single sign-on failed'), findsOneWidget);
      expect(find.text('assertion_invalid'), findsOneWidget);
      verifyNever(() => mockAuth.completeSSOLogin(any()));
    });

    testWidgets('a failed exchange shows its message', (tester) async {
      final mockAuth = buildIdleMock();
      when(() => mockAuth.completeSSOLogin(any())).thenThrow(
        const AuthClientException('invalid or expired code', code: 400),
      );

      await pumpAuthSomeApp(
        tester,
        child: SSOCallbackForm(auth: mockAuth, code: 'stale'),
      );
      await tester.pumpAndSettle();

      expect(find.text('invalid or expired code'), findsOneWidget);
    });
  });
}
