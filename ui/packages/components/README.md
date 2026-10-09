# AuthSome UI components

With `publishableKey` on `AuthProvider`, the sign-in and signup forms load the app's AutoConfig. Continue waits for that configuration before choosing a step. Failed configuration requests leave the email in place so you can retry. When `signup_enabled` is false, signup shows a message and sign-in hides the signup link.

Signup checks waitlist approval before showing details. Required choices and consent fields are validated before submission. Required captcha checks also apply to keyboard submission; failed attempts need a fresh token.

`SignInForm.onSuccess` and `SignUpForm.onSuccess` run after authentication. MFA opens the challenge form before sign-in completes. For signup that needs email verification, `onVerificationRequired(email)` lets you route to your verification page. Without that callback, the form shows the code input and Resend action inline. After verification, the user signs in to create a session. Optional or disabled verification completes signup only after the returned session is validated.

The `SignUp` routing component carries the email and `redirect` query into its verification route, then returns to sign-in after verification. `EmailVerificationForm` submits the email with the OTP code and sends resend requests by default. If you supply `onResend`, return its promise so the form can handle failures and start the cooldown only after success.
