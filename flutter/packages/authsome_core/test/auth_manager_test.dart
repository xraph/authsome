import 'package:authsome_core/authsome_core.dart';
import 'package:test/test.dart';

void main() {
  group('AuthManager.initialize', () {
    test('transitions to AuthUnauthenticated when storage is empty', () async {
      final states = <AuthState>[];
      final manager = AuthManager(
        AuthConfig(
          baseUrl: 'http://localhost',
          storage: MemoryTokenStorage(),
          onStateChange: states.add,
        ),
      );

      await manager.initialize();

      expect(states.last, isA<AuthUnauthenticated>());
      manager.dispose();
    });
  });

  group('AuthManager.subscribe', () {
    test('returns an unsubscribe function that removes the listener',
        () async {
      final states = <AuthState>[];
      final manager = AuthManager(
        AuthConfig(
          baseUrl: 'http://localhost',
          storage: MemoryTokenStorage(),
        ),
      );

      final unsubscribe = manager.subscribe(states.add);
      unsubscribe();

      await manager.initialize();

      expect(states, isEmpty,
          reason: 'unsubscribed listener should not receive events');
      manager.dispose();
    });
  });
}
