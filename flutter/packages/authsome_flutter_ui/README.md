# authsome_flutter_ui

Pre-built Material Design 3 authentication UI widgets for [AuthSome](https://github.com/xraph/authsome).

## Features

- Ready-to-use sign in, sign up, and password reset screens
- Material Design 3 theming
- MFA support
- Social login buttons

## Getting Started

```dart
import 'package:authsome_flutter_ui/authsome_flutter_ui.dart';

// Use pre-built auth screen
AuthScreen(
  baseUrl: 'https://your-authsome-server.com',
);
```

## AutoConfig and Continue

Set `publishableKey` on `AuthProvider` to load the app's authentication settings. Both forms wait for those settings before advancing from email. If the request fails, you can retry without losing your email. Closed signup shows a message and removes the signup link from sign-in.

Signup checks waitlist approval before asking for details. Required choices and consent fields are checked before submission. A required captcha blocks submission until a token is available, including submission from the keyboard. A failed attempt needs a fresh token.

`SignUpForm.onSuccess` and `SignInForm.onSuccess` run after authentication. Signup that requires verification opens the code form instead. Use `onVerificationComplete`, or `onSignInTap`, to return to sign-in after the email is verified. Verification itself does not create a session. When verification is explicitly optional or disabled, the SDK validates the returned signup session before completing authentication.

Supply `onSocialLogin` to show working social buttons on Flutter. Native SSO needs `onSSOLogin`; Flutter Web can redirect automatically. Native captcha integrations use `captchaBuilder`.

## Documentation

See the [AuthSome documentation](https://github.com/xraph/authsome) for more details.
