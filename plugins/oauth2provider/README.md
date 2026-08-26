# oauth2provider

Turns Authsome into an OAuth2 authorization server and OpenID Connect provider.

Everything else in this repo helps you consume identity. This plugin makes you the source of
it. Third-party apps register as clients, your users approve them, and those apps get tokens
scoped to what they were granted. If you have ever added a "Sign in with GitHub" button, this
is the plugin that lets someone add "Sign in with your product" to theirs.

Supported grants: authorization code with PKCE (RFC 7636), client credentials, and device
authorization for TVs and CLIs that cannot host a browser.

## Use it when

- You are building a platform and third parties need to act on behalf of your users
- You want first-party mobile and CLI apps to authenticate through a standard flow with short-lived tokens
- Something in your stack speaks OIDC and expects a discovery document

## Skip it when

- You only need machine-to-machine calls with no delegation. `apikey` is far less work
- All your clients are first-party and trusted. Sessions already cover that
- You cannot commit to a stable issuer URL. It is baked into every token you sign

## Wiring

```go
authsome.WithPlugin(oauth2provider.New(oauth2provider.Config{
    Issuer:          "https://auth.example.com",
    AccessTokenTTL:  1 * time.Hour,
    AuthCodeTTL:     10 * time.Minute,
}))
```

Point clients at `/v1/oauth/.well-known/openid-configuration` and most libraries configure
themselves from there.

## Config

| Field | Type | Default | What it does |
|---|---|---|---|
| `Issuer` | `string` | none | Issuer URL stamped into tokens and discovery |
| `AuthCodeTTL` | `time.Duration` | `10m` | Lifetime of an authorization code |
| `AccessTokenTTL` | `time.Duration` | `1h` | Lifetime of an access token |
| `DeviceCodeTTL` | `time.Duration` | `10m` | Lifetime of a device code |
| `DeviceCodeInterval` | `int` | `5` | Minimum client polling interval, in seconds |
| `VerificationURI` | `string` | `{issuer}/v1/oauth/device` | Where users go to approve a device |

## Endpoints

| Method | Path | Purpose |
|---|---|---|
| `GET` | `/v1/oauth/authorize` | Authorization endpoint |
| `POST` | `/v1/oauth/token` | Token endpoint |
| `POST` | `/v1/oauth/revoke` | Token revocation. The caller is authenticated |
| `GET` | `/v1/oauth/userinfo` | OIDC UserInfo |
| `GET` | `/v1/oauth/.well-known/openid-configuration` | Discovery document |
| `POST` | `/v1/oauth/device/authorize` | Start the device flow |
| `POST` | `/v1/oauth/device/complete` | Complete the device flow |
| `GET` | `/v1/admin/oauth/clients` | List registered clients |
| `POST` | `/v1/admin/oauth/clients` | Register a client |
| `DELETE` | `/v1/admin/oauth/clients/:clientId` | Delete a client |

## Lifecycle hooks

None. It is a `RouteProvider` and a `MigrationProvider` for its client, code and token tables.

## Related

`consent` records what a user agreed to. `social` and `sso` are the consuming side, where you
are the client rather than the server.
