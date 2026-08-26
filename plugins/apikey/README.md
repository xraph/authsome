# apikey

Long-lived keys for machine-to-machine calls, scoped to a user and revocable one at a time.

Keys are hashed with SHA-256 before storage, so a database dump does not hand anyone working
credentials. A short visible prefix is kept in the clear purely so lookup can find the right
row without scanning every hash. The auth strategy checks both the `Authorization` header and
`X-API-Key`, so callers can use whichever their HTTP client makes easy.

The plaintext key is returned exactly once, at creation.

## Use it when

- A service, a CI job or a CLI needs to call your API without a browser
- You want per-integration credentials that can be revoked without touching anyone's password
- You're building a public API and users need to generate their own keys

## Skip it when

- The caller is a browser with a logged-in human. Sessions are the right shape for that
- You need short-lived, scoped, audience-bound tokens. That's `oauth2provider` with the client credentials grant
- You have no way to rotate keys. A key that never expires is a credential you cannot take back once it leaks

## Wiring

```go
authsome.WithPlugin(apikey.New(apikey.Config{
    MaxKeysPerUser: 10,
    DefaultExpiry:  90 * 24 * time.Hour,
}))
```

Set `DefaultExpiry`. Keys that never expire are the ones still working three years after the
person who made them left.

## Config

| Field | Type | Default | What it does |
|---|---|---|---|
| `PathPrefix` | `string` | `/v1/keys` | Where the management routes are mounted |
| `MaxKeysPerUser` | `int` | `0` | Cap on active keys per user. `0` is unlimited |
| `DefaultExpiry` | `time.Duration` | `0` | Default TTL for a new key. Zero means it never expires |

## Settings

| Key | Default |
|---|---|
| `apikey.max_keys_per_user` | `0` |
| `apikey.default_expiry_seconds` | `0` |

## Endpoints

Registered under `PathPrefix`, `/v1/keys` by default.

| Method | Path | Purpose |
|---|---|---|
| `POST` | `` | Create a key. The plaintext is in this response and nowhere else |
| `GET` | `` | List the caller's keys, metadata only |
| `DELETE` | `/:keyId` | Revoke a key |

## Lifecycle hooks

None. It registers a `StrategyProvider` that authenticates requests carrying a key.

## Related

`oauth2provider` if you need real OAuth2 client credentials with scopes and expiry rather than
a static secret.
