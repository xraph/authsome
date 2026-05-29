import 'package:authsome_flutter/authsome_flutter.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

/// Pumps [child] under a [MaterialApp] + [Scaffold] for widget tests.
///
/// When [notifier] is non-null, the child is wrapped in [AuthProvider.test]
/// so descendants resolve the notifier via `context.auth`. When null, the
/// child renders bare so tests can drive a widget via direct injection
/// (e.g. `SignInForm(auth: mock)`).
Future<void> pumpAuthSomeApp(
  WidgetTester tester, {
  required Widget child,
  AuthNotifier? notifier,
}) async {
  final inner = MaterialApp(home: Scaffold(body: child));
  final tree = notifier == null
      ? inner
      : AuthProvider.test(notifier: notifier, child: inner);
  await tester.pumpWidget(tree);
}
