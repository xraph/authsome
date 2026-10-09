/// Pins how [SignUpForm] follows the client config: signup fields,
/// password toggle, captcha, email verification and branding. Mirrors
/// React `sign-up-form.test.tsx`.
library;

import 'package:authsome_flutter_ui/authsome_flutter_ui.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mocktail/mocktail.dart';

import '../_helpers/mock_auth_notifier.dart';
import '../_helpers/pump_auth_some_app.dart';

Future<void> _continueWithEmail(WidgetTester tester, String email) async {
  // Several dynamic fields don't fit the default 800x600 test surface, and
  // AuthCard leaves scrolling to the page that hosts it.
  tester.view.physicalSize = const Size(800, 1600);
  tester.view.devicePixelRatio = 1;
  addTearDown(tester.view.reset);
  await tester.enterText(find.widgetWithText(TextField, 'Email'), email);
  await tester.tap(find.widgetWithText(FilledButton, 'Continue'));
  await tester.pumpAndSettle();
}

const _fieldsConfig = ClientConfig(
  signupFields: [
    SignupFieldConfig(
      key: 'company',
      label: 'Company',
      type: 'text',
      order: 2,
      validation: SignupFieldValidation(required: true),
    ),
    SignupFieldConfig(
      key: 'first_name',
      label: 'First name',
      type: 'text',
      order: 1,
    ),
    SignupFieldConfig(
      key: 'plan',
      label: 'Plan',
      type: 'select',
      order: 3,
      defaultValue: 'pro',
      options: [
        SignupFieldOption(label: 'Free', value: 'free'),
        SignupFieldOption(label: 'Pro', value: 'pro'),
      ],
    ),
  ],
);

void main() {
  setUpAll(() {
    registerAuthFallbacks();
    registerFallbackValue(() {});
  });

  group('signup fields', () {
    testWidgets('replace the name field and are sent as fields',
        (tester) async {
      final mockAuth = buildIdleMock(clientConfig: _fieldsConfig);

      await pumpAuthSomeApp(tester, child: SignUpForm(auth: mockAuth));
      await _continueWithEmail(tester, 'new@example.com');

      expect(find.widgetWithText(TextField, 'Full name'), findsNothing);
      // Sorted by order: first_name (1) before company (2).
      final firstY = tester.getTopLeft(find.text('First name (optional)')).dy;
      final companyY = tester.getTopLeft(find.text('Company')).dy;
      expect(firstY, lessThan(companyY));

      await tester.enterText(
        find.widgetWithText(TextFormField, 'First name (optional)'),
        'Jane',
      );
      await tester.enterText(
        find.widgetWithText(TextFormField, 'Company'),
        'Acme',
      );
      await tester.enterText(find.byType(TextField).last, 'hunter2');
      await tester.tap(find.widgetWithText(FilledButton, 'Sign up'));
      await tester.pump();

      verify(() => mockAuth.signUp(
            'new@example.com',
            'hunter2',
            name: null,
            fields: {'first_name': 'Jane', 'company': 'Acme', 'plan': 'pro'},
            captchaToken: null,
          )).called(1);
    });

    testWidgets('a missing required field blocks sign-up', (tester) async {
      final mockAuth = buildIdleMock(clientConfig: _fieldsConfig);

      await pumpAuthSomeApp(tester, child: SignUpForm(auth: mockAuth));
      await _continueWithEmail(tester, 'new@example.com');
      await tester.enterText(find.byType(TextField).last, 'hunter2');
      await tester.tap(find.widgetWithText(FilledButton, 'Sign up'));
      await tester.pump();

      expect(find.text('Company is required'), findsOneWidget);
      verifyNever(() => mockAuth.signUp(
            any(),
            any(),
            name: any(named: 'name'),
            fields: any(named: 'fields'),
            captchaToken: any(named: 'captchaToken'),
          ));
    });
  });

  testWidgets('password disabled leaves only social sign-up', (tester) async {
    final mockAuth = buildIdleMock(
      clientConfig: const ClientConfig(
        password: PasswordConfig(enabled: false),
        social: SocialConfig(
          enabled: true,
          providers: [SocialProviderConfig(id: 'github', name: 'GitHub')],
        ),
      ),
    );

    await pumpAuthSomeApp(tester,
        child: SignUpForm(auth: mockAuth, onSocialLogin: (_) {}));

    expect(find.byType(SocialButtons), findsOneWidget);
    expect(find.widgetWithText(TextField, 'Email'), findsNothing);
  });

  testWidgets('captcha token is required and sent', (tester) async {
    final mockAuth = buildIdleMock(
      clientConfig: const ClientConfig(
        captcha: CaptchaConfig(
          required: true,
          provider: 'turnstile',
          siteKey: '0x4AAA',
        ),
      ),
    );

    await pumpAuthSomeApp(
      tester,
      child: SignUpForm(
        auth: mockAuth,
        captchaBuilder: (context, config, onToken) => TextButton(
          onPressed: () => onToken('tok'),
          child: const Text('Solve captcha'),
        ),
      ),
    );
    await _continueWithEmail(tester, 'new@example.com');
    await tester.enterText(find.byType(TextField).last, 'hunter2');
    await tester.pump();

    final submit = find.widgetWithText(FilledButton, 'Sign up');
    expect(tester.widget<FilledButton>(submit).onPressed, isNull);

    await tester.tap(find.text('Solve captcha'));
    await tester.pump();
    await tester.tap(submit);
    await tester.pump();

    verify(() => mockAuth.signUp(
          'new@example.com',
          'hunter2',
          name: null,
          fields: null,
          captchaToken: 'tok',
        )).called(1);
  });

  testWidgets('verification pending swaps to the email verification form',
      (tester) async {
    final mockAuth = buildIdleMock();
    VoidCallback? listener;
    when(() => mockAuth.addListener(any())).thenAnswer((inv) {
      listener = inv.positionalArguments.first as VoidCallback;
    });

    await pumpAuthSomeApp(tester, child: SignUpForm(auth: mockAuth));
    await _continueWithEmail(tester, 'new@example.com');

    when(() => mockAuth.state)
        .thenReturn(const AuthVerificationPending(email: 'new@example.com'));
    listener!();
    await tester.pumpAndSettle();

    expect(find.byType(EmailVerificationForm), findsOneWidget);
    expect(find.textContaining('new@example.com'), findsWidgets);
  });

  testWidgets('branded title uses the app name', (tester) async {
    final mockAuth = buildIdleMock(
      clientConfig:
          const ClientConfig(branding: BrandingConfig(appName: 'Acme')),
    );

    await pumpAuthSomeApp(tester, child: SignUpForm(auth: mockAuth));

    expect(find.text('Create your Acme account'), findsOneWidget);
  });

  group('validateSignupField', () {
    const field = SignupFieldConfig(
      key: 'age',
      label: 'Age',
      type: 'number',
      validation: SignupFieldValidation(min: 18, max: 120),
    );

    test('enforces numeric bounds', () {
      expect(validateSignupField(field, '12'), 'Age must be at least 18');
      expect(validateSignupField(field, '200'), 'Age must be at most 120');
      expect(validateSignupField(field, 'abc'), 'Age must be a number');
      expect(validateSignupField(field, '30'), isNull);
      expect(validateSignupField(field, ''), isNull,
          reason: 'an optional empty field is fine');
    });

    test('anchors pattern to the whole value like HTML does', () {
      const code = SignupFieldConfig(
        key: 'code',
        label: 'Code',
        type: 'text',
        validation: SignupFieldValidation(pattern: '[A-Z]{3}'),
      );
      expect(validateSignupField(code, 'ABC'), isNull);
      expect(validateSignupField(code, 'ABCD'), isNotNull);
    });

    test('a required checkbox must be ticked', () {
      const terms = SignupFieldConfig(
        key: 'terms',
        label: 'Terms',
        type: 'checkbox',
        validation: SignupFieldValidation(required: true),
      );
      expect(validateSignupField(terms, 'false'), 'Terms is required');
      expect(validateSignupField(terms, 'true'), isNull);
    });
  });
}
