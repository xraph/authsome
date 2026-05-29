import 'package:authsome_flutter/authsome_flutter.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mocktail/mocktail.dart';

class _MockAuthNotifier extends Mock implements AuthNotifier {}

void main() {
  group('AuthProvider', () {
    testWidgets('maybeOf returns null when no provider is mounted',
        (tester) async {
      AuthNotifier? resolved;
      await tester.pumpWidget(
        MaterialApp(
          home: Builder(
            builder: (context) {
              resolved = AuthProvider.maybeOf(context);
              return const SizedBox.shrink();
            },
          ),
        ),
      );
      expect(resolved, isNull);
    });

    testWidgets('AuthProvider.test exposes the injected notifier via context.auth',
        (tester) async {
      final mock = _MockAuthNotifier();
      AuthNotifier? resolved;

      await tester.pumpWidget(
        AuthProvider.test(
          notifier: mock,
          child: MaterialApp(
            home: Builder(
              builder: (context) {
                resolved = AuthProvider.of(context);
                return const SizedBox.shrink();
              },
            ),
          ),
        ),
      );

      expect(identical(resolved, mock), isTrue);
    });

    testWidgets('AuthProvider.test does not dispose externally-owned notifier',
        (tester) async {
      final mock = _MockAuthNotifier();
      when(() => mock.dispose()).thenReturn(null);

      await tester.pumpWidget(
        AuthProvider.test(
          notifier: mock,
          child: const SizedBox.shrink(),
        ),
      );

      // Replace the tree to trigger dispose of the provider state.
      await tester.pumpWidget(const SizedBox.shrink());

      verifyNever(() => mock.dispose());
    });
  });
}
