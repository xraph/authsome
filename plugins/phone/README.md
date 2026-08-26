# phone

Sign in with a phone number and a one-time code, no password anywhere.

This is the flow most consumer apps outside the West default to, because a phone number is the
identity people actually have. The plugin reuses the SMS bridge and code generation that `mfa`
already ships, so you're not configuring two SMS paths.

By default an unrecognised number creates an account. Turn `AutoCreate` off if you want signup
to be a separate, gated step.

## Use it when

- Your users are mobile-first and a phone number is more reliable than an email address
- You're launching in a market where SMS OTP is the expected sign-in
- You want signup and sign-in to be the same two-screen flow

## Skip it when

- SIM swap is in your threat model. A phone number is a weak root of identity for anything financial
- SMS costs matter at your volume. Every sign-in is a paid message
- Your users are on desktop and will not have the phone to hand

## Wiring

```go
authsome.WithPlugin(phone.New(phone.Config{
    CodeTTL: 5 * time.Minute,
}))
```

The engine's SMS bridge is used when `SMSSender` is nil, so wire Twilio or your provider once
at the engine level and both this plugin and `mfa` pick it up.

## Config

| Field | Type | Default | What it does |
|---|---|---|---|
| `SMSSender` | `bridge.SMSSender` | engine bridge | Where codes are sent from |
| `CodeTTL` | `time.Duration` | `5m` | How long an OTP stays valid |
| `AutoCreate` | `*bool` | `true` | Create a user when the number is not known. `false` errors instead |

## Settings

| Key | Default |
|---|---|
| `phone.code_ttl_seconds` | `300` |
| `phone.auto_create` | `true` |

## Endpoints

Registered under `/v1/phone`.

| Method | Path | Purpose |
|---|---|---|
| `POST` | `/start` | Send an OTP to the number |
| `POST` | `/verify` | Exchange the OTP for a session |

## Lifecycle hooks

None. It registers as an `AuthMethodContributor` so phone shows up as a linked method on the
user record.

## Related

`mfa` shares the SMS plumbing. If you want the same low-friction flow over email instead, use
`magiclink`.
