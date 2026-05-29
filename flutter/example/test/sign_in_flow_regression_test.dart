/// Regression test for "form bounces back to email after pressing Sign in".
///
/// The bug: when `AuthsomeExampleApp` rebuilt the router inline on every
/// build, every `notifyListeners()` call on the auth notifier created a
/// fresh [GoRouter]. MaterialApp.router then reset its navigator,
/// remounting [SignInPage] with a fresh `_SignInFormState` whose `_step`
/// defaulted back to email.
///
/// The fix: cache the router in a [StatefulWidget] that builds it once
/// in `didChangeDependencies`. This test pins the contract — multiple
/// auth state transitions must leave the form mounted on whatever step
/// the user advanced to.
library;

import 'package:authsome_example/main.dart';
import 'package:authsome_flutter_ui/authsome_flutter_ui.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import '_helpers/fake_auth_notifier.dart';

void main() {
  testWidgets(
    'SignInForm stays on the password step after auth notify '
    '(Unauthenticated → Loading → Error) — does not bounce back to email',
    (tester) async {
      final fake = FakeAuthNotifier();

      await tester.pumpWidget(AuthsomeExampleApp(authOverride: fake));
      await tester.pumpAndSettle();

      // Step 1: confirm we landed on the sign-in route with the form
      // showing the email step.
      expect(find.byType(SignInForm), findsOneWidget);
      expect(find.widgetWithText(TextField, 'Email'), findsOneWidget);

      // Advance to the password step.
      await tester.enterText(
        find.widgetWithText(TextField, 'Email'),
        'user@example.com',
      );
      await tester.tap(find.widgetWithText(FilledButton, 'Continue'));
      await tester.pumpAndSettle();

      expect(find.widgetWithText(FilledButton, 'Sign in'), findsOneWidget,
          reason: 'should now be on the password step');

      // Simulate the AuthLoading → AuthError transition that happens
      // when the backend is unreachable (the case the user reported with
      // ERR_CONNECTION_REFUSED on POST /v1/signin).
      fake.setState(const AuthLoading());
      await tester.pump();
      fake.setState(const AuthError(error: 'Connection refused'));
      await tester.pumpAndSettle();

      // The form must still be visible AND still on the password step.
      // A bug where notifyListeners recreated the router would have
      // remounted SignInPage and reset _step back to email.
      expect(find.byType(SignInForm), findsOneWidget,
          reason: 'form must survive the state transition');
      expect(find.widgetWithText(FilledButton, 'Sign in'), findsOneWidget,
          reason: 'must still be on password step, not bounced to email');
    },
  );
}
