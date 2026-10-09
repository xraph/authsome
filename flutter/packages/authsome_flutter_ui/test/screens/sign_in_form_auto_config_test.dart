/// Pins how [SignInForm] follows the client config: password toggle,
/// magic link, SSO, captcha and branding. Mirrors the React tests under
/// `ui/packages/components/src/components/sign-in-form.*.test.tsx`.
library;

import 'package:authsome_flutter_ui/authsome_flutter_ui.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mocktail/mocktail.dart';

import '../_helpers/mock_auth_notifier.dart';
import '../_helpers/pump_auth_some_app.dart';

Future<void> _continueWithEmail(WidgetTester tester, String email) async {
  await tester.enterText(find.widgetWithText(TextField, 'Email'), email);
  await tester.tap(find.widgetWithText(FilledButton, 'Continue'));
  await tester.pumpAndSettle();
}

/// A stand-in captcha that hands out a token when tapped.
Widget _fakeCaptcha(
  BuildContext context,
  CaptchaConfig config,
  ValueChanged<String?> onToken,
) {
  return TextButton(
    onPressed: () => onToken('captcha-tok'),
    child: const Text('Solve captcha'),
  );
}

void main() {
  setUpAll(registerAuthFallbacks);

  group('password toggle', () {
    testWidgets('password disabled with nothing else shows the empty notice',
        (tester) async {
      final mockAuth = buildIdleMock(
        clientConfig: const ClientConfig(password: PasswordConfig(enabled: false)),
      );

      await pumpAuthSomeApp(tester, child: SignInForm(auth: mockAuth));

      expect(find.widgetWithText(TextField, 'Email'), findsNothing);
      expect(find.textContaining('No sign-in methods'), findsOneWidget);
    });

    testWidgets('password disabled keeps social buttons and drops the email',
        (tester) async {
      final mockAuth = buildIdleMock(
        clientConfig: const ClientConfig(
          password: PasswordConfig(enabled: false),
          social: SocialConfig(
            enabled: true,
            providers: [SocialProviderConfig(id: 'google', name: 'Google')],
          ),
        ),
      );

      await pumpAuthSomeApp(tester, child: SignInForm(auth: mockAuth));

      expect(find.byType(SocialButtons), findsOneWidget);
      expect(find.widgetWithText(TextField, 'Email'), findsNothing);
      expect(find.byType(OrDivider), findsNothing);
      expect(find.textContaining('No sign-in methods'), findsNothing);
    });
  });

  group('magic link', () {
    testWidgets('password step offers the link only when enabled',
        (tester) async {
      final mockAuth = buildIdleMock(
        clientConfig: const ClientConfig(magiclink: MagicLinkConfig(enabled: true)),
      );

      await pumpAuthSomeApp(tester, child: SignInForm(auth: mockAuth));
      await _continueWithEmail(tester, 'user@example.com');

      await tester.tap(find.text('Email me a sign-in link instead'));
      await tester.pumpAndSettle();

      verify(() => mockAuth.sendMagicLink('user@example.com')).called(1);
      expect(find.text('Check your inbox'), findsOneWidget);
      expect(find.textContaining('user@example.com'), findsWidgets);
    });

    testWidgets('showMagicLink:false beats magiclink.enabled', (tester) async {
      final mockAuth = buildIdleMock(
        clientConfig: const ClientConfig(magiclink: MagicLinkConfig(enabled: true)),
      );

      await pumpAuthSomeApp(
        tester,
        child: SignInForm(auth: mockAuth, showMagicLink: false),
      );
      await _continueWithEmail(tester, 'user@example.com');

      expect(find.text('Email me a sign-in link instead'), findsNothing);
    });

    testWidgets('magic link is hidden when the config leaves it off',
        (tester) async {
      final mockAuth = buildIdleMock(clientConfig: const ClientConfig());

      await pumpAuthSomeApp(tester, child: SignInForm(auth: mockAuth));
      await _continueWithEmail(tester, 'user@example.com');

      expect(find.text('Email me a sign-in link instead'), findsNothing);
    });

    testWidgets('with passwords off, the email field sends a magic link',
        (tester) async {
      final mockAuth = buildIdleMock(
        clientConfig: const ClientConfig(
          password: PasswordConfig(enabled: false),
          magiclink: MagicLinkConfig(enabled: true),
        ),
      );

      await pumpAuthSomeApp(tester, child: SignInForm(auth: mockAuth));

      expect(find.widgetWithText(FilledButton, 'Continue'), findsNothing);
      await tester.enterText(
        find.widgetWithText(TextField, 'Email'),
        'user@example.com',
      );
      await tester.tap(
        find.widgetWithText(FilledButton, 'Email me a sign-in link'),
      );
      await tester.pumpAndSettle();

      verify(() => mockAuth.sendMagicLink('user@example.com')).called(1);
      verifyNever(() => mockAuth.signIn(any(), any(),
          captchaToken: any(named: 'captchaToken')));
      expect(find.text('Check your inbox'), findsOneWidget);
    });
  });

  group('SSO', () {
    const ssoConfig = ClientConfig(
      sso: SsoConfig(
        enabled: true,
        connections: [SSOConnectionConfig(id: 'okta', name: 'Okta')],
      ),
    );

    testWidgets('renders a button per connection and calls onSSOLogin',
        (tester) async {
      final mockAuth = buildIdleMock(clientConfig: ssoConfig);
      final tapped = <String>[];

      await pumpAuthSomeApp(
        tester,
        child: SignInForm(auth: mockAuth, onSSOLogin: tapped.add),
      );

      await tester.tap(find.text('Continue with Okta'));
      await tester.pump();

      expect(tapped, ['okta']);
      verifyNever(() =>
          mockAuth.startSSOLogin(any(), returnUrl: any(named: 'returnUrl')));
    });

    testWidgets('hides the buttons when sso.enabled is false', (tester) async {
      final mockAuth = buildIdleMock(
        clientConfig: const ClientConfig(
          sso: SsoConfig(
            enabled: false,
            connections: [SSOConnectionConfig(id: 'okta', name: 'Okta')],
          ),
        ),
      );

      await pumpAuthSomeApp(
        tester,
        child: SignInForm(auth: mockAuth, onSSOLogin: (_) {}),
      );

      expect(find.text('Continue with Okta'), findsNothing);
    });

    testWidgets(
        'off the Web, buttons need an onSSOLogin since there is no redirect',
        (tester) async {
      final mockAuth = buildIdleMock(clientConfig: ssoConfig);

      await pumpAuthSomeApp(tester, child: SignInForm(auth: mockAuth));

      expect(find.byType(SSOButtons), findsNothing);
    });

    testWidgets('ssoConnections overrides the config', (tester) async {
      final mockAuth = buildIdleMock(clientConfig: ssoConfig);

      await pumpAuthSomeApp(
        tester,
        child: SignInForm(
          auth: mockAuth,
          onSSOLogin: (_) {},
          ssoConnections: const [
            SSOConnectionConfig(id: 'entra', name: 'Microsoft Entra'),
          ],
        ),
      );

      expect(find.text('Continue with Microsoft Entra'), findsOneWidget);
      expect(find.text('Continue with Okta'), findsNothing);
    });
  });

  group('captcha', () {
    const captchaConfig = ClientConfig(
      captcha: CaptchaConfig(
        required: true,
        provider: 'turnstile',
        siteKey: '0x4AAA',
      ),
    );

    testWidgets('blocks sign-in until a token arrives, then sends it',
        (tester) async {
      final mockAuth = buildIdleMock(clientConfig: captchaConfig);

      await pumpAuthSomeApp(
        tester,
        child: SignInForm(auth: mockAuth, captchaBuilder: _fakeCaptcha),
      );
      await _continueWithEmail(tester, 'user@example.com');
      await tester.enterText(find.byType(TextField).last, 'hunter2');
      await tester.pump();

      final signIn = find.widgetWithText(FilledButton, 'Sign in');
      expect(tester.widget<FilledButton>(signIn).onPressed, isNull);

      await tester.tap(find.text('Solve captcha'));
      await tester.pump();
      expect(tester.widget<FilledButton>(signIn).onPressed, isNotNull);

      await tester.tap(signIn);
      await tester.pump();

      verify(() => mockAuth.signIn('user@example.com', 'hunter2',
          captchaToken: 'captcha-tok')).called(1);
    });

    testWidgets('without a builder off the Web, says the captcha cannot show',
        (tester) async {
      final mockAuth = buildIdleMock(clientConfig: captchaConfig);

      await pumpAuthSomeApp(tester, child: SignInForm(auth: mockAuth));
      await _continueWithEmail(tester, 'user@example.com');

      expect(find.textContaining('requires a captcha'), findsOneWidget);
    });

    testWidgets('no captcha when the config does not require one',
        (tester) async {
      final mockAuth = buildIdleMock(
        clientConfig: const ClientConfig(
          captcha: CaptchaConfig(required: false, siteKey: '0x4AAA'),
        ),
      );

      await pumpAuthSomeApp(
        tester,
        child: SignInForm(auth: mockAuth, captchaBuilder: _fakeCaptcha),
      );
      await _continueWithEmail(tester, 'user@example.com');

      expect(find.text('Solve captcha'), findsNothing);
    });
  });

  group('branding', () {
    testWidgets('titles the card with the app name', (tester) async {
      final mockAuth = buildIdleMock(
        clientConfig: const ClientConfig(
          branding: BrandingConfig(appName: 'Acme'),
        ),
      );

      await pumpAuthSomeApp(tester, child: SignInForm(auth: mockAuth));

      expect(find.text('Sign in to Acme'), findsOneWidget);
    });

    testWidgets('an explicit titleText wins', (tester) async {
      final mockAuth = buildIdleMock(
        clientConfig: const ClientConfig(
          branding: BrandingConfig(appName: 'Acme'),
        ),
      );

      await pumpAuthSomeApp(
        tester,
        child: SignInForm(auth: mockAuth, titleText: 'Welcome back'),
      );

      expect(find.text('Welcome back'), findsOneWidget);
      expect(find.text('Sign in to Acme'), findsNothing);
    });

    testWidgets('falls back to the branding logo', (tester) async {
      final mockAuth = buildIdleMock(
        clientConfig: const ClientConfig(
          branding: BrandingConfig(
            appName: 'Acme',
            logoUrl: 'https://example.com/logo.png',
          ),
        ),
      );

      await pumpAuthSomeApp(tester, child: SignInForm(auth: mockAuth));

      final image = tester.widget<Image>(find.byType(Image));
      expect((image.image as NetworkImage).url, 'https://example.com/logo.png');
    });
  });
}
