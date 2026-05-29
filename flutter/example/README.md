# AuthSome Flutter — example app

Runnable demo for the three SDK packages (`authsome_core`, `authsome_flutter`,
`authsome_flutter_ui`). Use it to smoke-test changes end-to-end and to verify
parity with the React reference UI under `ui/`.

## Running

Start the backend (from repo root):

```bash
go run ./cmd/authsome --listen :8080
```

Then in another terminal:

```bash
cd flutter/example
flutter run -d chrome --dart-define=AUTHSOME_BASE_URL=http://localhost:8080
```

A helper script wraps the flags:

```bash
./scripts/run-local.sh chrome
```

Available devices: `macos`, `chrome`, `ios`, `android`, `linux`, `windows`.

## Routes

| Path | Page | Notes |
|---|---|---|
| `/` | Home | `AuthGuard`-protected landing |
| `/profile` | Profile | dumps the signed-in user |
| `/sign-in` | Sign in | email-first 2-step flow + social discovery |
| `/sign-up` | Sign up | email + name + password |
| `/forgot-password` | Forgot password | email → reset link |
| `/reset-password?token=...` | Reset password | deep-link friendly via query param |
| `/magic-link` | Magic link | email → magic link |
| `/verify-email?email=...` | Verify email | OTP entry |
| `/mfa-challenge` | MFA challenge | TOTP / SMS / recovery codes |

The router holds redirects until the auth state settles (past `AuthIdle`) to
avoid the splash flicker that would otherwise yank unauthenticated users away
from the home page before session hydration finishes.

## Tests

```bash
flutter test                     # widget smoke tests in test/
flutter test integration_test/   # end-to-end tests (Phase 6)
```

## Workspace

This package participates in the parent workspace (`flutter/pubspec.yaml`).
Edits to any of the three SDK packages hot-reload here without `pub get`.
