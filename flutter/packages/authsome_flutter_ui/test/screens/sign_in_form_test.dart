import 'package:authsome_flutter_ui/authsome_flutter_ui.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mocktail/mocktail.dart';

import '../_helpers/mock_auth_notifier.dart';
import '../_helpers/pump_auth_some_app.dart';

void main() {
  setUpAll(registerAuthFallbacks);

  group('SignInForm', () {
    testWidgets(
      'tapping Continue then Sign in calls AuthNotifier.signIn with the entered email and password',
      (tester) async {
        final mockAuth = buildIdleMock();

        await pumpAuthSomeApp(
          tester,
          child: SignInForm(auth: mockAuth),
        );

        await tester.enterText(
          find.widgetWithText(TextField, 'Email'),
          'user@example.com',
        );
        await tester.tap(find.widgetWithText(FilledButton, 'Continue'));
        await tester.pumpAndSettle();

        // Password step is now visible.
        await tester.enterText(
          find.byType(TextField).last,
          'hunter2',
        );
        await tester.tap(find.widgetWithText(FilledButton, 'Sign in'));
        await tester.pump();

        verify(() => mockAuth.signIn('user@example.com', 'hunter2'))
            .called(1);
      },
    );

    testWidgets(
      'renders an inline missing-AuthProvider error when neither an ancestor provider nor an injected notifier is available',
      (tester) async {
        // Capture framework errors so we can assert no FlutterError was thrown.
        final caught = <FlutterErrorDetails>[];
        final previousOnError = FlutterError.onError;
        FlutterError.onError = caught.add;

        try {
          await pumpAuthSomeApp(
            tester,
            child: const SignInForm(),
          );

          expect(
            find.textContaining('AuthProvider'),
            findsOneWidget,
            reason: 'should render a developer-friendly inline error '
                'instead of throwing in didChangeDependencies',
          );
          expect(caught, isEmpty, reason: 'should not throw FlutterError');
        } finally {
          FlutterError.onError = previousOnError;
        }
      },
    );
  });
}
