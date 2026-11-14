# magiclink

Passwordless sign-in over email. The user asks for a link, clicks it, and they are in. Turn
on the email code and the same email also carries a 6-digit code they can type instead. Turn
on sign-up and a new address gets an account with no password.

There's no new table for this. Tokens ride on the core verification entity with
`type="magic_link"`, so enabling the plugin does not cost you a migration. You supply a
`Mailer` and the plugin handles minting, expiry and single use.

The code is not a new table either. It lives in the ceremony store (the shared KV table),
hashed and tied to the link it was sent with.

Every endpoint is rate limited, `/send` against the resend-verification budget and `/verify`
and `/verify-code` against the verify-email budget, so nobody can use your auth server to bomb
an inbox.

## Use it when

- You want signup with as little friction as possible, and no password to reset later
- Support cost from forgotten passwords is a real line item for you
- Email is already the identity of record for your users

## Skip it when

- You have no reliable transactional email. This plugin is exactly as available as your mail provider
- Your users share an inbox, which turns a magic link into a shared credential
- Sign-in has to work offline or on a device with no mail client. `passkey` handles that better

## Wiring

```go
authsome.WithPlugin(magiclink.New(magiclink.Config{
    Mailer:   myMailer, // implements magiclink.Mailer
    TokenTTL: 10 * time.Minute,
}))
```

`Mailer` is required. Without it the plugin has no way to deliver anything.

For the code, sign-up, or a different email per purpose, implement `CodeMailer` too:

```go
type CodeMailer interface {
    Mailer
    SendMagicLinkWithCode(ctx context.Context, msg magiclink.MagicLinkMessage) error
}

type MagicLinkMessage struct {
    AppID     string
    Email     string
    Token     string    // goes in the link; the client posts it to /verify
    Code      string    // 6 digits, empty unless the email code is on
    Purpose   Purpose   // "signin" or "signup"
    ExpiresAt time.Time // when both the link and the code stop working
    Name      string    // the user's name, when the account has one
}
```

When the `Mailer` implements it, the plugin calls `SendMagicLinkWithCode` and never
`SendMagicLink`. The code is only minted for a mailer that implements it.

## Config

| Field | Type | Default | What it does |
|---|---|---|---|
| `Mailer` | `Mailer` | none | Sends the link. Required |
| `TokenTTL` | `time.Duration` | `10m` | How long a link stays valid |
| `SessionTokenTTL` | `time.Duration` | `1h` | Lifetime of the session the link creates |
| `SessionRefreshTTL` | `time.Duration` | `30d` | Lifetime of that session's refresh token |
| `EmailCode` | `bool` | `false` | Also send a 6-digit code. Needs a `CodeMailer` |
| `AllowSignUp` | `bool` | `false` | Let `/send` with `create_user` create a passwordless account |

## Settings

| Key | Default |
|---|---|
| `magiclink.token_ttl_seconds` | `600` |
| `magiclink.session_token_ttl_seconds` | `3600` |
| `magiclink.session_refresh_ttl_seconds` | `2592000` |
| `magiclink.email_code` | `false` (unset, `Config.EmailCode` decides) |
| `magiclink.allow_signup` | `false` (unset, `Config.AllowSignUp` decides) |

## Endpoints

Registered under `/v1/magic-link`.

| Method | Path | Purpose |
|---|---|---|
| `POST` | `/send` | Mint a token (and code) and email them |
| `POST` | `/verify` | Exchange the token for a session |
| `POST` | `/verify-code` | Exchange the emailed code for a session |

`/send` takes `{"email", "app_id"?, "create_user"?, "name"?, "captcha_token"?}` and answers
`{"status": "magic link sent", "expires_in": 600, "code_length": 6}`. `code_length` is there
only when a code is sent. The answer is the same whether or not the address has an account,
and nothing is sent to an address with none.

`/verify` takes `{"token"}` and `/verify-code` takes `{"email", "code"}`. Both answer
`{"user", "session_token", "refresh_token", "expires_at"}`, or the usual 403 with
`"type": "mfa_required"` and an `mfa_ticket` when the user has a second factor. Using either
one proves the inbox, so the email is marked verified.

The link and the code are one credential: they share one lifetime, and whichever is used first
uses up the other. A code allows five tries, right or wrong, and a resend replaces it. An
unknown address, a wrong code, a used one and an expired one all get the same 401.

## Passwordless sign-up

With `AllowSignUp` (or `magiclink.allow_signup`) on and sign-up enabled for the app, a send
with `"create_user": true` creates an account with no password for an address nobody has, and
the email goes out with purpose `signup`. An address that has an account gets an ordinary
sign-in link and the same answer.

The account goes through the engine's sign-up path (`Engine.SignUpPasswordless`), so the
waitlist and any other `BeforeSignUp` hook, the app's sign-up switch, custom form rules, email
normalization and the per-address send limit apply as they do for `/v1/signup`. The sign-up
captcha is asked for too, from the `X-Captcha-Token` header or `captcha_token` in the body,
and only for a send with `create_user`. A sign-up that is off or refused is a 403, decided
before the address is looked up.

Such an account can't sign in at `/v1/signin` with any password until forgot and reset
password give it one.

## Lifecycle hooks

None. The plugin is a `RouteProvider` and a `SettingsProvider`, and stays out of the core
sign-in path.

## Related

`passkey` is the other passwordless option, and it does not depend on your mail provider.

One thing to watch: `magiclink.Mailer` is its own one-method interface, separate from the
engine-wide `bridge.Mailer` that `email` and the core flows use. You wire it on this plugin's
config, not with `authsome.WithMailer`.
