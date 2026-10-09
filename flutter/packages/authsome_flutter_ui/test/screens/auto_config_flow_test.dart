import 'dart:async';
import 'dart:convert';
import 'package:authsome_flutter_ui/authsome_flutter_ui.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:mocktail/mocktail.dart';
import '../_helpers/mock_auth_notifier.dart';
import '../_helpers/pump_auth_some_app.dart';

http.Response jsonResponse(Object body, [int status = 200]) =>
    http.Response(jsonEncode(body), status,
        headers: {'content-type': 'application/json'});
Future<void> continueEmail(WidgetTester tester) async {
  await tester.enterText(
      find.widgetWithText(TextField, 'Email'), 'Ada@Example.com');
  await tester.tap(find.widgetWithText(FilledButton, 'Continue'));
  await tester.pumpAndSettle();
}

Future<void> enterPassword(WidgetTester tester) async {
  await tester.enterText(find.byType(TextField).last, 'password123');
}

void main() {
  testWidgets('delayed config cannot open a disabled password step',
      (tester) async {
    final config = Completer<http.Response>();
    final auth = AuthNotifier(AuthConfig(
        baseUrl: 'https://a.test',
        publishableKey: 'pk_test',
        storage: MemoryTokenStorage(),
        httpClient: MockClient((_) => config.future)));
    addTearDown(auth.dispose);
    await pumpAuthSomeApp(tester, child: SignInForm(auth: auth));
    await tester.enterText(
        find.widgetWithText(TextField, 'Email'), 'ada@example.com');
    await tester.tap(find.text('Continue'));
    await tester.pump();
    expect(find.byType(PasswordInput), findsNothing);
    config.complete(jsonResponse({
      'password': {'enabled': false}
    }));
    await tester.pumpAndSettle();
    expect(find.byType(PasswordInput), findsNothing);
  });
  testWidgets('closed signup has no form or signup footer', (tester) async {
    final auth =
        buildIdleMock(clientConfig: const ClientConfig(signupEnabled: false));
    await pumpAuthSomeApp(tester, child: SignUpForm(auth: auth));
    expect(find.text("Signup isn't available"), findsOneWidget);
    expect(find.byType(TextField), findsNothing);
    await pumpAuthSomeApp(tester,
        child: SignInForm(auth: auth, onSignUpTap: () {}));
    expect(find.text("Don't have an account? Sign up"), findsNothing);
  });
  testWidgets('invalid email stays on the first step', (tester) async {
    final auth = buildIdleMock();
    await pumpAuthSomeApp(tester, child: SignInForm(auth: auth));
    await tester.enterText(find.widgetWithText(TextField, 'Email'), 'invalid');
    await tester.tap(find.text('Continue'));
    await tester.pumpAndSettle();
    expect(find.text('Please enter a valid email address'), findsOneWidget);
    expect(find.byType(PasswordInput), findsNothing);
  });
  testWidgets('social providers without a handler have no dead buttons',
      (tester) async {
    final auth = buildIdleMock(
        clientConfig: const ClientConfig(
            social: SocialConfig(enabled: true, providers: [
      SocialProviderConfig(id: 'google', name: 'Google')
    ])));
    await pumpAuthSomeApp(tester, child: SignInForm(auth: auth));
    expect(find.byType(SocialButtons), findsNothing);
  });
  testWidgets('MFA is shown before login success at a narrow width',
      (tester) async {
    await tester.binding.setSurfaceSize(const Size(375, 812));
    addTearDown(() => tester.binding.setSurfaceSize(null));
    var successes = 0;
    final auth = AuthNotifier(AuthConfig(
        baseUrl: 'https://a.test',
        storage: MemoryTokenStorage(),
        httpClient: MockClient((_) async => jsonResponse({
              'error': 'MFA required',
              'type': 'mfa_required',
              'mfa_ticket': 'ticket',
              'available_methods': ['totp', 'sms']
            }, 403))));
    addTearDown(auth.dispose);
    await pumpAuthSomeApp(tester,
        child: SignInForm(auth: auth, onSuccess: () => successes++));
    await continueEmail(tester);
    await enterPassword(tester);
    await tester.tap(find.widgetWithText(FilledButton, 'Sign in'));
    await tester.pumpAndSettle();
    expect(find.byType(MfaChallengeForm), findsOneWidget);
    expect(find.text('Send code'), findsNothing);
    expect(successes, 0);
    expect(tester.takeException(), isNull);
  });
  testWidgets(
      'verification returns to sign-in without reporting authentication',
      (tester) async {
    var authenticated = 0;
    var verified = 0;
    final auth = AuthNotifier(AuthConfig(
      baseUrl: 'https://a.test',
      storage: MemoryTokenStorage(),
      initialClientConfig: const ClientConfig(
          emailVerification:
              EmailVerificationConfig(enabled: true, required: true)),
      httpClient: MockClient((req) async {
        if (req.url.path == '/v1/verify-email') {
          return jsonResponse({'status': 'verified'});
        }
        return jsonResponse({
          'user': {
            'id': 'u_1',
            'app_id': 'app_1',
            'env_id': 'env_1',
            'email': 'ada@example.com',
            'email_verified': false,
            'first_name': 'Ada',
            'last_name': '',
            'banned': false,
            'phone_verified': false,
            'created_at': '2024-01-01T00:00:00Z',
            'updated_at': '2024-01-01T00:00:00Z'
          },
          'session_token': 'token',
          'refresh_token': 'refresh',
          'expires_at': '2099-01-01T00:00:00Z'
        });
      }),
    ));
    addTearDown(auth.dispose);
    await pumpAuthSomeApp(tester,
        child: SignUpForm(
            auth: auth,
            onSuccess: () => authenticated++,
            onVerificationComplete: () => verified++));
    await continueEmail(tester);
    await enterPassword(tester);
    await tester.tap(find.widgetWithText(FilledButton, 'Sign up'));
    await tester.pumpAndSettle();
    expect(find.byType(EmailVerificationForm), findsOneWidget);
    await tester.enterText(find.byType(TextField), '123456');
    await tester.pumpAndSettle();
    expect(verified, 1);
    expect(authenticated, 0);
    expect(auth.session, isNull);
  });
  testWidgets('required captcha with no key still blocks keyboard submission',
      (tester) async {
    final auth = buildIdleMock(
        clientConfig: const ClientConfig(
            captcha: CaptchaConfig(required: true, provider: 'turnstile')));
    await pumpAuthSomeApp(tester, child: SignInForm(auth: auth));
    await continueEmail(tester);
    await enterPassword(tester);
    await tester.testTextInput.receiveAction(TextInputAction.done);
    await tester.pump();
    expect(find.text('Verification is unavailable. Please try again later.'),
        findsOneWidget);
    verifyNever(() =>
        auth.signIn(any(), any(), captchaToken: any(named: 'captchaToken')));
  });
  testWidgets(
      'failed sign-in consumes the captcha and permits a fresh challenge',
      (tester) async {
    final auth = buildIdleMock(
        clientConfig: const ClientConfig(
            captcha: CaptchaConfig(
                required: true, provider: 'turnstile', siteKey: 'site')));
    var challenges = 0;
    when(() =>
            auth.signIn(any(), any(), captchaToken: any(named: 'captchaToken')))
        .thenThrow(const AuthClientException('Wrong password'));
    await pumpAuthSomeApp(tester,
        child: SignInForm(
            auth: auth,
            captchaBuilder: (_, config, onToken) {
              challenges++;
              return TextButton(
                  onPressed: () => onToken('token'),
                  child: const Text('Complete captcha'));
            }));
    await continueEmail(tester);
    await enterPassword(tester);
    await tester.tap(find.text('Complete captcha'));
    await tester.pump();
    await tester.tap(find.widgetWithText(FilledButton, 'Sign in'));
    await tester.pumpAndSettle();
    expect(find.text('Wrong password'), findsOneWidget);
    final button = tester
        .widget<FilledButton>(find.widgetWithText(FilledButton, 'Sign in'));
    expect(button.onPressed, isNull);
    expect(challenges, greaterThan(1));
  });
  testWidgets('resend errors leave the code resend available', (tester) async {
    var attempts = 0;
    await pumpAuthSomeApp(tester,
        child: EmailVerificationForm(
            auth: buildIdleMock(),
            email: 'ada@example.com',
            onResend: () async {
              if (++attempts == 1) throw Exception('offline');
            }));
    await tester.tap(find.text('Resend code'));
    await tester.pump();
    expect(find.text('Exception: offline'), findsOneWidget);
    await tester.tap(find.text('Resend code'));
    await tester.pump();
    expect(attempts, 2);
    await tester.pumpWidget(const SizedBox.shrink());
  });
}
