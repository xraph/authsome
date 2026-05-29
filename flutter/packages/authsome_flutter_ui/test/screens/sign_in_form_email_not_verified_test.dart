/// Confirms the email_not_verified recovery flow in [SignInForm].
///
/// Mirrors React `sign-in-form.tsx` lines 158–179 — when the server returns
/// `type: 'email_not_verified'`, the form swaps to an inline verification
/// panel showing the email + a Resend button.
library;

import 'package:authsome_flutter_ui/authsome_flutter_ui.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mocktail/mocktail.dart';

import '../_helpers/mock_auth_notifier.dart';
import '../_helpers/pump_auth_some_app.dart';

Future<void> _advancePastSubmit(WidgetTester tester) async {
  await tester.enterText(
    find.widgetWithText(TextField, 'Email'),
    'user@example.com',
  );
  await tester.tap(find.widgetWithText(FilledButton, 'Continue'));
  await tester.pumpAndSettle();
  await tester.enterText(find.byType(TextField).last, 'wrong-pw');
  await tester.tap(find.widgetWithText(FilledButton, 'Sign in'));
  await tester.pumpAndSettle();
}

void main() {
  setUpAll(registerAuthFallbacks);

  testWidgets(
    'SignInForm swaps to the verification panel when signIn throws email_not_verified',
    (tester) async {
      final mockAuth = buildIdleMock();
      when(() => mockAuth.signIn(any(), any())).thenThrow(
        const AuthClientException(
          'Please verify your email',
          code: 403,
          type: 'email_not_verified',
        ),
      );

      await pumpAuthSomeApp(tester, child: SignInForm(auth: mockAuth));
      await _advancePastSubmit(tester);

      expect(find.textContaining('Verify your email'), findsOneWidget);
      expect(find.text('user@example.com'), findsWidgets);
    },
  );

  testWidgets(
    'tapping Resend on the verification panel calls AuthNotifier.resendVerification with the entered email',
    (tester) async {
      final mockAuth = buildIdleMock();
      when(() => mockAuth.signIn(any(), any())).thenThrow(
        const AuthClientException(
          'Please verify your email',
          code: 403,
          type: 'email_not_verified',
        ),
      );

      await pumpAuthSomeApp(tester, child: SignInForm(auth: mockAuth));
      await _advancePastSubmit(tester);

      await tester.tap(find.widgetWithText(TextButton, 'Resend'));
      await tester.pump();

      verify(() => mockAuth.resendVerification('user@example.com')).called(1);
    },
  );
}
