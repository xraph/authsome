import 'package:authsome_flutter_ui/authsome_flutter_ui.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import '../_helpers/mock_auth_notifier.dart';
import '../_helpers/pump_auth_some_app.dart';

void main() {
  setUpAll(registerAuthFallbacks);

  testWidgets(
    'renders inline error when no AuthProvider and no injected auth',
    (tester) async {
      final caught = <FlutterErrorDetails>[];
      final previousOnError = FlutterError.onError;
      FlutterError.onError = caught.add;
      try {
        await pumpAuthSomeApp(tester, child: const ForgotPasswordForm());
        expect(find.textContaining('AuthProvider'), findsOneWidget);
        expect(caught, isEmpty);
      } finally {
        FlutterError.onError = previousOnError;
      }
    },
  );

  testWidgets('submits an empty-email error when the field is empty',
      (tester) async {
    final mockAuth = buildIdleMock();
    await pumpAuthSomeApp(
      tester,
      child: ForgotPasswordForm(auth: mockAuth),
    );
    await tester.tap(find.widgetWithText(FilledButton, 'Send reset link'));
    await tester.pump();
    expect(find.textContaining('Please enter your email'), findsOneWidget);
  });
}
