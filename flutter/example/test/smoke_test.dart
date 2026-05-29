import 'package:authsome_example/main.dart';
import 'package:authsome_flutter_ui/authsome_flutter_ui.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mocktail/mocktail.dart';

class _MockAuthNotifier extends Mock implements AuthNotifier {}

MockAuthNotifier _idleMock() {
  final mock = _MockAuthNotifier();
  when(() => mock.state).thenReturn(const AuthUnauthenticated());
  when(() => mock.error).thenReturn(null);
  when(() => mock.clientConfig).thenReturn(null);
  when(() => mock.isAuthenticated).thenReturn(false);
  when(() => mock.isLoading).thenReturn(false);
  when(() => mock.isMfaRequired).thenReturn(false);
  when(() => mock.isConfigLoaded).thenReturn(false);
  when(() => mock.signIn(any(), any())).thenAnswer((_) async {});
  return mock;
}

typedef MockAuthNotifier = _MockAuthNotifier;

void main() {
  testWidgets(
    'example app boots SignInForm at "/" when the notifier is unauthenticated',
    (tester) async {
      final mock = _idleMock();

      await tester.pumpWidget(AuthsomeExampleApp(authOverride: mock));
      await tester.pumpAndSettle();

      expect(find.byType(SignInForm), findsOneWidget);
    },
  );
}
