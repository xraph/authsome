# magiclink

Passwordless sign-in over email. The user asks for a link, clicks it, and they are in.

There's no new table for this. Tokens ride on the core verification entity with
`type="magic_link"`, so enabling the plugin does not cost you a migration. You supply a
`Mailer` and the plugin handles minting, expiry and single use.

Both endpoints are rate limited, `/send` against the resend-verification budget and `/verify`
against the verify-email budget, so nobody can use your auth server to bomb an inbox.

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

## Config

| Field | Type | Default | What it does |
|---|---|---|---|
| `Mailer` | `Mailer` | none | Sends the link. Required |
| `TokenTTL` | `time.Duration` | `10m` | How long a link stays valid |
| `SessionTokenTTL` | `time.Duration` | `1h` | Lifetime of the session the link creates |
| `SessionRefreshTTL` | `time.Duration` | `30d` | Lifetime of that session's refresh token |

## Settings

| Key | Default |
|---|---|
| `magiclink.token_ttl_seconds` | `600` |
| `magiclink.session_token_ttl_seconds` | `3600` |
| `magiclink.session_refresh_ttl_seconds` | `2592000` |

## Endpoints

Registered under `/v1/magic-link`.

| Method | Path | Purpose |
|---|---|---|
| `POST` | `/send` | Mint a token and email the link |
| `POST` | `/verify` | Exchange the token for a session |

## Lifecycle hooks

None. The plugin is a `RouteProvider` and a `SettingsProvider`, and stays out of the core
sign-in path.

## Related

`passkey` is the other passwordless option, and it does not depend on your mail provider.

One thing to watch: `magiclink.Mailer` is its own one-method interface, separate from the
engine-wide `bridge.Mailer` that `email` and the core flows use. You wire it on this plugin's
config, not with `authsome.WithMailer`.
