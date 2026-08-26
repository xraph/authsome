# mfa

Second-factor authentication with TOTP, SMS and recovery codes.

TOTP is the default and the one to prefer: the user scans a QR code into Google Authenticator,
1Password or whatever they already use, and you're not paying per message or trusting a mobile
carrier. SMS is there because some users will not install an app. Recovery codes exist because
people lose phones.

The plugin injects a challenge after sign-in when the user has MFA enabled, so the session
is not fully issued until the second factor clears.

## Use it when

- You hold anything worth stealing and passwords alone are not enough
- Compliance asks for MFA and you'd rather not buy a separate product for it
- You want step-up on sensitive actions, driven by `riskengine`

## Skip it when

- `passkey` already covers your users. A passkey is one factor that's stronger than password plus TOTP, and stacking both is friction for no gain
- You're relying on SMS as the only option. SIM swap is a real attack and TOTP costs nothing
- Your users have no second device at all

## Wiring

```go
authsome.WithPlugin(mfa.New(mfa.Config{
    Issuer: "My App",
}))
```

`Issuer` is the label users see in their authenticator app, so make it the product name they
recognise.

## Config

| Field | Type | Default | What it does |
|---|---|---|---|
| `Issuer` | `string` | `AuthSome` | Name shown in the authenticator app |

## Settings

| Key | Default |
|---|---|
| `mfa.issuer` | `AuthSome` |

## Endpoints

Registered under `/v1/mfa`.

| Method | Path | Purpose |
|---|---|---|
| `POST` | `/enroll` | Start TOTP enrolment, returns the secret and QR payload |
| `POST` | `/verify` | Verify a TOTP code |
| `POST` | `/challenge` | Issue an MFA challenge |
| `DELETE` | `/enrollment` | Disable MFA for the user |
| `POST` | `/sms/send` | Send an SMS code |
| `POST` | `/sms/verify` | Verify an SMS code |
| `POST` | `/recovery/verify` | Redeem a recovery code |
| `POST` | `/recovery/regenerate` | Issue a fresh set of recovery codes |

## Lifecycle hooks

None on the hook bus. It's a `RouteProvider`, a `SettingsProvider` and a `MigrationProvider`
for its enrolment tables.

## Related

`phone` uses the same SMS infrastructure but as a primary method rather than a second factor.
`riskengine` can require an MFA challenge when a sign-in scores in the middle band.
