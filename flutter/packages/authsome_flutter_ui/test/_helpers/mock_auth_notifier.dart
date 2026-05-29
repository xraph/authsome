import 'package:authsome_flutter/authsome_flutter.dart';
import 'package:mocktail/mocktail.dart';

class MockAuthNotifier extends Mock implements AuthNotifier {}

/// Hook for tests that need to register mocktail fallback values.
/// Currently a no-op (signIn args are built-in `String` types), but kept
/// so future tests that pass complex `any()` args have a single
/// initialisation point.
void registerAuthFallbacks() {}

MockAuthNotifier buildIdleMock({
  AuthState state = const AuthIdle(),
  String? error,
  ClientConfig? clientConfig,
}) {
  final mock = MockAuthNotifier();
  when(() => mock.state).thenReturn(state);
  when(() => mock.error).thenReturn(error);
  when(() => mock.clientConfig).thenReturn(clientConfig);
  when(() => mock.isAuthenticated).thenReturn(state is AuthAuthenticated);
  when(() => mock.isLoading).thenReturn(state is AuthLoading);
  when(() => mock.isMfaRequired).thenReturn(state is AuthMfaRequired);
  when(() => mock.isConfigLoaded).thenReturn(clientConfig != null);
  when(() => mock.signIn(any(), any())).thenAnswer((_) async {});
  when(() => mock.signUp(any(), any(), name: any(named: 'name')))
      .thenAnswer((_) async {});
  when(() => mock.signOut()).thenAnswer((_) async {});
  when(() => mock.refreshNow()).thenAnswer((_) async {});
  when(() => mock.resendVerification(any())).thenAnswer((_) async {});
  when(() => mock.verifyEmail(any())).thenAnswer((_) async {});
  when(() => mock.submitMFAChallenge(any())).thenAnswer((_) async {});
  return mock;
}
