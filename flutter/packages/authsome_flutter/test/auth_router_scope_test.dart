/// Pins the contract that [AuthRouterScope] builds its value exactly
/// once and keeps it stable across auth notifier state changes — the
/// invariant that prevents `MaterialApp.router` from resetting its
/// navigator and remounting active pages on every `notifyListeners()`.
library;

import 'package:authsome_flutter/authsome_flutter.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_test/flutter_test.dart';

class _ProbeNotifier extends ChangeNotifier implements AuthNotifier {
  AuthState _state = const AuthUnauthenticated();

  void emit(AuthState next) {
    _state = next;
    notifyListeners();
  }

  @override
  AuthState get state => _state;
  @override
  bool get isAuthenticated => _state is AuthAuthenticated;
  @override
  bool get isLoading => _state is AuthLoading;
  @override
  bool get isMfaRequired => _state is AuthMfaRequired;
  @override
  bool get isConfigLoaded => false;
  @override
  ClientConfig? get clientConfig => null;
  @override
  String? get error =>
      _state is AuthError ? (_state as AuthError).error : null;
  @override
  dynamic get user => null;
  @override
  Session? get session => null;

  @override
  dynamic noSuchMethod(Invocation invocation) => null;
}

void main() {
  testWidgets(
    'routerBuilder is called exactly once even after many notify cycles',
    (tester) async {
      final notifier = _ProbeNotifier();
      int builds = 0;

      await tester.pumpWidget(
        AuthProvider.test(
          notifier: notifier,
          child: AuthRouterScope<String>(
            routerBuilder: (_, __) {
              builds++;
              return 'router-instance-$builds';
            },
            builder: (_, router) =>
                Text(router, textDirection: TextDirection.ltr),
          ),
        ),
      );

      expect(find.text('router-instance-1'), findsOneWidget);
      expect(builds, 1);

      // Fire several state transitions. Each notify rebuilds the
      // InheritedNotifier dependents, including AuthRouterScope. The
      // cached value must NOT be recomputed.
      notifier.emit(const AuthLoading());
      await tester.pump();
      notifier.emit(const AuthError(error: 'oops'));
      await tester.pump();
      notifier.emit(const AuthUnauthenticated());
      await tester.pump();

      expect(builds, 1,
          reason: 'cached router must survive every notifyListeners()');
      expect(find.text('router-instance-1'), findsOneWidget);
    },
  );

  testWidgets(
    'builder receives the same router instance across rebuilds',
    (tester) async {
      final notifier = _ProbeNotifier();
      final builderReceived = <Object>[];

      await tester.pumpWidget(
        AuthProvider.test(
          notifier: notifier,
          child: AuthRouterScope<Object>(
            routerBuilder: (_, __) => Object(),
            builder: (_, router) {
              builderReceived.add(router);
              return const SizedBox.shrink();
            },
          ),
        ),
      );

      notifier.emit(const AuthLoading());
      await tester.pump();
      notifier.emit(const AuthError(error: 'x'));
      await tester.pump();

      expect(builderReceived.length, greaterThanOrEqualTo(2),
          reason: 'builder rebuilds on each notify');
      expect(
        builderReceived.every((r) => identical(r, builderReceived.first)),
        isTrue,
        reason: 'every rebuild must receive the exact same router instance',
      );
    },
  );
}
