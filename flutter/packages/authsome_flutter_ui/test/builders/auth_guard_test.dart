/// Tests that [AuthGuard] switches its rendered child based on
/// every [AuthState] variant, including the Phase 5 additions.
library;

import 'package:authsome_flutter_ui/authsome_flutter_ui.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import '../_helpers/mock_auth_notifier.dart';
import '../_helpers/pump_auth_some_app.dart';

const _childKey = Key('guard-child');
const _loadingKey = Key('guard-loading');
const _fallbackKey = Key('guard-fallback');

Future<void> _pumpGuard(WidgetTester tester, AuthState state) async {
  final mock = buildIdleMock(state: state);
  await pumpAuthSomeApp(
    tester,
    notifier: mock,
    child: const AuthGuard(
      loading: SizedBox(key: _loadingKey),
      fallback: SizedBox(key: _fallbackKey),
      child: SizedBox(key: _childKey),
    ),
  );
}

void main() {
  setUpAll(registerAuthFallbacks);

  group('AuthGuard', () {
    testWidgets('renders loading for AuthIdle', (tester) async {
      await _pumpGuard(tester, const AuthIdle());
      expect(find.byKey(_loadingKey), findsOneWidget);
      expect(find.byKey(_childKey), findsNothing);
      expect(find.byKey(_fallbackKey), findsNothing);
    });

    testWidgets('renders loading for AuthLoading', (tester) async {
      await _pumpGuard(tester, const AuthLoading());
      expect(find.byKey(_loadingKey), findsOneWidget);
    });

    testWidgets('renders child for AuthAuthenticated', (tester) async {
      await _pumpGuard(
        tester,
        const AuthAuthenticated(
          user: {'email': 'x@example.com'},
          session: Session(
            sessionToken: 't',
            refreshToken: 'r',
            expiresAt: '2099-01-01T00:00:00Z',
          ),
        ),
      );
      expect(find.byKey(_childKey), findsOneWidget);
    });

    testWidgets('renders fallback for AuthUnauthenticated', (tester) async {
      await _pumpGuard(tester, const AuthUnauthenticated());
      expect(find.byKey(_fallbackKey), findsOneWidget);
    });

    testWidgets('renders fallback for AuthError', (tester) async {
      await _pumpGuard(tester, const AuthError(error: 'boom'));
      expect(find.byKey(_fallbackKey), findsOneWidget);
    });

    testWidgets('renders fallback for AuthMfaRequired', (tester) async {
      await _pumpGuard(
        tester,
        const AuthMfaRequired(
          email: 'x@example.com',
          mfaTicket: 'tk',
          availableMethods: ['totp'],
        ),
      );
      expect(find.byKey(_fallbackKey), findsOneWidget);
    });

    testWidgets('renders fallback for AuthEmailNotVerified', (tester) async {
      await _pumpGuard(
        tester,
        const AuthEmailNotVerified(email: 'x@example.com'),
      );
      expect(find.byKey(_fallbackKey), findsOneWidget);
    });

    testWidgets('renders fallback for AuthVerificationPending',
        (tester) async {
      await _pumpGuard(
        tester,
        const AuthVerificationPending(email: 'x@example.com'),
      );
      expect(find.byKey(_fallbackKey), findsOneWidget);
    });
  });
}
