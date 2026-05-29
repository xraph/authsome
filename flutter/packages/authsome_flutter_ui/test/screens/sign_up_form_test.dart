import 'package:authsome_flutter_ui/authsome_flutter_ui.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mocktail/mocktail.dart';

import '../_helpers/mock_auth_notifier.dart';
import '../_helpers/pump_auth_some_app.dart';

void main() {
  setUpAll(registerAuthFallbacks);

  testWidgets(
    'two-step flow advances to details step then calls signUp with name+password',
    (tester) async {
      final mockAuth = buildIdleMock();

      await pumpAuthSomeApp(tester, child: SignUpForm(auth: mockAuth));

      await tester.enterText(
        find.widgetWithText(TextField, 'Email'),
        'new@example.com',
      );
      await tester.tap(find.widgetWithText(FilledButton, 'Continue'));
      await tester.pumpAndSettle();

      await tester.enterText(
        find.widgetWithText(TextField, 'Full name'),
        'Jane Doe',
      );
      await tester.enterText(find.byType(TextField).last, 'hunter2');
      await tester.tap(find.widgetWithText(FilledButton, 'Sign up'));
      await tester.pump();

      verify(() => mockAuth.signUp('new@example.com', 'hunter2',
          name: 'Jane Doe')).called(1);
    },
  );

  testWidgets('renders inline error when no AuthProvider and no injected auth',
      (tester) async {
    final caught = <FlutterErrorDetails>[];
    final previousOnError = FlutterError.onError;
    FlutterError.onError = caught.add;
    try {
      await pumpAuthSomeApp(tester, child: const SignUpForm());
      expect(find.textContaining('AuthProvider'), findsOneWidget);
      expect(caught, isEmpty);
    } finally {
      FlutterError.onError = previousOnError;
    }
  });
}
