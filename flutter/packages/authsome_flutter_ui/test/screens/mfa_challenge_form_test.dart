/// Tests the ticket-based MFA flow on [MfaChallengeForm].
///
/// Mirrors React `auth.ts` `submitMFAChallenge(code)` reading the ticket
/// from the [AuthMfaRequired] state.
library;

import 'package:authsome_flutter_ui/authsome_flutter_ui.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mocktail/mocktail.dart';

import '../_helpers/mock_auth_notifier.dart';
import '../_helpers/pump_auth_some_app.dart';

void main() {
  setUpAll(registerAuthFallbacks);

  testWidgets(
    'completing the TOTP code calls submitMFAChallenge when state carries a ticket',
    (tester) async {
      final mockAuth = buildIdleMock(
        state: const AuthMfaRequired(
          email: 'user@example.com',
          mfaTicket: 'tk_abc',
          availableMethods: ['totp'],
        ),
      );

      await pumpAuthSomeApp(
        tester,
        child: MfaChallengeForm(auth: mockAuth),
      );

      // OtpInput is a single hidden TextField (length 6) that fires
      // onCompleted when the input hits the configured length.
      await tester.enterText(find.byType(TextField).first, '123456');
      // pump one frame for the onCompleted → submitMFAChallenge call.
      // (pumpAndSettle would never return because the form switches to
      // a CircularProgressIndicator that animates indefinitely.)
      await tester.pump();

      verify(() => mockAuth.submitMFAChallenge('123456')).called(1);
    },
  );

  testWidgets(
    'renders inline missing-provider error when no AuthProvider and no injected auth',
    (tester) async {
      final caught = <FlutterErrorDetails>[];
      final previousOnError = FlutterError.onError;
      FlutterError.onError = caught.add;

      try {
        await pumpAuthSomeApp(
          tester,
          child: const MfaChallengeForm(),
        );

        expect(find.textContaining('AuthProvider'), findsOneWidget);
        expect(caught, isEmpty);
      } finally {
        FlutterError.onError = previousOnError;
      }
    },
  );
}
