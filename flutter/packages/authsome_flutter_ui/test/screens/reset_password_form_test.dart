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
        await pumpAuthSomeApp(
          tester,
          child: const ResetPasswordForm(token: 't_123'),
        );
        expect(find.textContaining('AuthProvider'), findsOneWidget);
        expect(caught, isEmpty);
      } finally {
        FlutterError.onError = previousOnError;
      }
    },
  );

  testWidgets('surfaces a mismatch error when passwords do not match',
      (tester) async {
    final mockAuth = buildIdleMock();
    await pumpAuthSomeApp(
      tester,
      child: ResetPasswordForm(token: 't_123', auth: mockAuth),
    );
    await tester.enterText(find.byType(TextField).first, 'one');
    await tester.enterText(find.byType(TextField).last, 'two');
    await tester.tap(find.widgetWithText(FilledButton, 'Reset password'));
    await tester.pump();
    expect(find.textContaining('do not match'), findsOneWidget);
  });
}
