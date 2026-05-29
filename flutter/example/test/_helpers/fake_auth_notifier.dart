import 'package:authsome_flutter/authsome_flutter.dart';
import 'package:flutter/foundation.dart';

/// Hand-rolled stand-in for [AuthNotifier]. Extends [ChangeNotifier] so
/// `notifyListeners()` actually fires registered subscribers (mocktail
/// mocks stub `notifyListeners` and silently swallow the call, which is
/// useless for tests that need to drive state transitions).
///
/// `noSuchMethod` returns a sensible default for every unstubbed member
/// of the [AuthNotifier] surface — only the bits the example app calls
/// (`state`, `error`, `clientConfig`, `isAuthenticated`, `signIn`, …)
/// need explicit handling.
class FakeAuthNotifier extends ChangeNotifier implements AuthNotifier {
  AuthState _state;
  ClientConfig? _clientConfig;

  /// Records every signIn(email, password) call, so tests can assert on
  /// the order and arguments without using a mock framework.
  final List<({String email, String password})> signInCalls = [];

  FakeAuthNotifier({
    AuthState state = const AuthUnauthenticated(),
    ClientConfig? clientConfig,
  })  : _state = state,
        _clientConfig = clientConfig;

  /// Test seam: update the state and notify listeners in one call.
  void setState(AuthState next) {
    _state = next;
    notifyListeners();
  }

  @override
  AuthState get state => _state;

  @override
  ClientConfig? get clientConfig => _clientConfig;

  @override
  bool get isAuthenticated => _state is AuthAuthenticated;

  @override
  bool get isLoading => _state is AuthLoading;

  @override
  bool get isMfaRequired => _state is AuthMfaRequired;

  @override
  bool get isConfigLoaded => _clientConfig != null;

  @override
  String? get error =>
      _state is AuthError ? (_state as AuthError).error : null;

  @override
  dynamic get user =>
      _state is AuthAuthenticated ? (_state as AuthAuthenticated).user : null;

  @override
  Session? get session {
    if (_state is AuthAuthenticated) {
      return (_state as AuthAuthenticated).session;
    }
    // ignore: deprecated_member_use
    if (_state is AuthMfaRequired) return (_state as AuthMfaRequired).session;
    return null;
  }

  @override
  Future<void> signIn(String email, String password) async {
    signInCalls.add((email: email, password: password));
    setState(const AuthLoading());
  }

  // Everything else falls through to a no-op. `dynamic` return so the
  // analyzer doesn't complain about non-nullable getter types we don't
  // exercise; tests that need a specific value should subclass.
  @override
  dynamic noSuchMethod(Invocation invocation) => null;
}
