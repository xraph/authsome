import 'dart:async';

import 'package:authsome_flutter_ui/authsome_flutter_ui.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mocktail/mocktail.dart';

import '../_helpers/mock_auth_notifier.dart';
import '../_helpers/pump_auth_some_app.dart';

class _MockClient extends Mock implements AuthSomeClient {}

class _MockManager extends Mock implements AuthManager {}

Future<void> _continue(WidgetTester tester) async {
  await tester.enterText(
      find.widgetWithText(TextField, 'Email'), 'ada@example.com');
  await tester.tap(find.widgetWithText(FilledButton, 'Continue'));
  await tester.pumpAndSettle();
}

void main() {
  testWidgets('waits for auto-config before checking approval', (tester) async {
    final auth = buildIdleMock();
    final client = _MockClient();
    final manager = _MockManager();
    final config = Completer<ClientConfig>();
    when(() => auth.client).thenReturn(client);
    when(() => auth.manager).thenReturn(manager);
    when(() => client.publishableKey).thenReturn('pk_test');
    when(() => manager.fetchClientConfig()).thenAnswer((_) => config.future);
    when(() => client.joinWaitlistWithStatus(any()))
        .thenAnswer((_) async => WaitlistStatus.pending);
    await pumpAuthSomeApp(tester, child: SignUpForm(auth: auth));
    await tester.enterText(
        find.widgetWithText(TextField, 'Email'), 'ada@example.com');
    await tester.tap(find.text('Continue'));
    await tester.pump();
    expect(find.widgetWithText(FilledButton, 'Sign up'), findsNothing);
    verifyNever(() => client.joinWaitlistWithStatus(any()));
    config.complete(const ClientConfig(waitlist: ToggleConfig(enabled: true)));
    await tester.pumpAndSettle();
    expect(find.text("You're on the waitlist"), findsOneWidget);
    verify(() => client.joinWaitlistWithStatus('ada@example.com')).called(1);
  });

  testWidgets('waitlist message fits a narrow viewport', (tester) async {
    await tester.binding.setSurfaceSize(const Size(375, 812));
    addTearDown(() => tester.binding.setSurfaceSize(null));
    final auth = buildIdleMock(
      clientConfig: const ClientConfig(waitlist: ToggleConfig(enabled: true)),
    );
    final client = _MockClient();
    when(() => auth.client).thenReturn(client);
    when(() => client.joinWaitlistWithStatus(any()))
        .thenAnswer((_) async => WaitlistStatus.pending);
    await pumpAuthSomeApp(tester, child: SignUpForm(auth: auth));
    await _continue(tester);
    expect(find.text("You're on the waitlist"), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  for (final status in [WaitlistStatus.pending, WaitlistStatus.rejected]) {
    testWidgets('${status.name} shows a message without signup details',
        (tester) async {
      final auth = buildIdleMock(
        clientConfig: const ClientConfig(waitlist: ToggleConfig(enabled: true)),
      );
      final client = _MockClient();
      when(() => auth.client).thenReturn(client);
      when(() => client.joinWaitlistWithStatus(any()))
          .thenAnswer((_) async => status);
      await pumpAuthSomeApp(tester, child: SignUpForm(auth: auth));
      await _continue(tester);

      expect(
          find.text(status == WaitlistStatus.pending
              ? "You're on the waitlist"
              : "Signup isn't available"),
          findsOneWidget);
      expect(find.widgetWithText(FilledButton, 'Sign up'), findsNothing);
      verify(() => client.joinWaitlistWithStatus('ada@example.com')).called(1);
      verifyNever(() => auth.signUp(any(), any()));

      await tester.tap(find.text('Use another email'));
      await tester.pumpAndSettle();
      expect(find.widgetWithText(TextField, 'Email'), findsOneWidget);
    });
  }

  testWidgets('approved entries continue to details', (tester) async {
    final auth = buildIdleMock(
      clientConfig: const ClientConfig(waitlist: ToggleConfig(enabled: true)),
    );
    final client = _MockClient();
    when(() => auth.client).thenReturn(client);
    when(() => client.joinWaitlistWithStatus(any()))
        .thenAnswer((_) async => WaitlistStatus.approved);
    await pumpAuthSomeApp(tester, child: SignUpForm(auth: auth));
    await _continue(tester);
    expect(find.widgetWithText(FilledButton, 'Sign up'), findsOneWidget);
    expect(find.text('ada@example.com'), findsOneWidget);
  });

  testWidgets('pending approval can be checked again', (tester) async {
    final auth = buildIdleMock(
      clientConfig: const ClientConfig(waitlist: ToggleConfig(enabled: true)),
    );
    final client = _MockClient();
    var status = WaitlistStatus.pending;
    when(() => auth.client).thenReturn(client);
    when(() => client.joinWaitlistWithStatus(any()))
        .thenAnswer((_) async => status);
    await pumpAuthSomeApp(tester, child: SignUpForm(auth: auth));
    await _continue(tester);
    status = WaitlistStatus.approved;
    await tester.tap(find.text('Check status'));
    await tester.pumpAndSettle();
    expect(find.widgetWithText(FilledButton, 'Sign up'), findsOneWidget);
  });

  testWidgets('a failed check stays on email and permits retry',
      (tester) async {
    final auth = buildIdleMock(
      clientConfig: const ClientConfig(waitlist: ToggleConfig(enabled: true)),
    );
    final client = _MockClient();
    when(() => auth.client).thenReturn(client);
    when(() => client.joinWaitlistWithStatus(any())).thenThrow(
        const AuthClientException('Temporarily unavailable', code: 503));
    await pumpAuthSomeApp(tester, child: SignUpForm(auth: auth));
    await _continue(tester);
    expect(find.text('Temporarily unavailable'), findsOneWidget);
    expect(find.widgetWithText(FilledButton, 'Sign up'), findsNothing);
    when(() => client.joinWaitlistWithStatus(any()))
        .thenAnswer((_) async => WaitlistStatus.pending);
    await tester.tap(find.widgetWithText(FilledButton, 'Continue'));
    await tester.pumpAndSettle();
    expect(find.text("You're on the waitlist"), findsOneWidget);
  });

  testWidgets('disables input while checking and tolerates disposal',
      (tester) async {
    final auth = buildIdleMock(
      clientConfig: const ClientConfig(waitlist: ToggleConfig(enabled: true)),
    );
    final client = _MockClient();
    final result = Completer<WaitlistStatus>();
    when(() => auth.client).thenReturn(client);
    when(() => client.joinWaitlistWithStatus(any()))
        .thenAnswer((_) => result.future);
    await pumpAuthSomeApp(tester, child: SignUpForm(auth: auth));
    await tester.enterText(
        find.widgetWithText(TextField, 'Email'), 'ada@example.com');
    await tester.tap(find.text('Continue'));
    await tester.pump();
    expect(tester.widget<TextField>(find.byType(TextField)).enabled, isFalse);
    expect(tester.widget<FilledButton>(find.byType(FilledButton)).onPressed,
        isNull);
    await tester.pumpWidget(const SizedBox());
    result.complete(WaitlistStatus.approved);
    await tester.pumpAndSettle();
    expect(tester.takeException(), isNull);
  });

  for (final waitlist in [null, const ToggleConfig(enabled: false)]) {
    testWidgets('skips the waitlist when absent or disabled: $waitlist',
        (tester) async {
      final auth =
          buildIdleMock(clientConfig: ClientConfig(waitlist: waitlist));
      await pumpAuthSomeApp(tester, child: SignUpForm(auth: auth));
      await _continue(tester);
      expect(find.widgetWithText(FilledButton, 'Sign up'), findsOneWidget);
      verifyNever(() => auth.client);
    });
  }
}
