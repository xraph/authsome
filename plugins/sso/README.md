# sso

SAML 2.0 and OIDC single sign-on, configured per organisation.

This is the plugin enterprise deals ask for. Each tenant points at its own identity provider,
Okta, Entra, Ping, whatever they run, and their staff sign in with corporate credentials that
you never see. Users get provisioned automatically on first login.

Domain-based routing means someone can type their work email on your sign-in page and get sent
to the right identity provider without picking anything from a list.

## Use it when

- You are selling to companies with an identity team and an SSO line item in the contract
- Offboarding has to be instant. Revoking access in the identity provider should end access to you
- Each tenant needs its own identity provider, which is why connections are scoped per organisation

## Skip it when

- Your users are individuals. `social` is the right shape for personal accounts
- You cannot serve a stable public HTTPS base URL. SAML entity IDs and ACS URLs derive from it
- You have not decided what happens to an SSO user when their organisation is deleted

## Wiring

```go
authsome.WithPlugin(sso.New(sso.Config{
    PublicBaseURL:        "https://auth.example.com",
    AllowedReturnOrigins: []string{"https://app.example.com"},
}))
```

`PublicBaseURL` is required for SAML. `AllowedReturnOrigins` is the open-redirect guard on the
post-ACS landing, so treat it as a security control and not a convenience.

## Config

| Field | Type | Default | What it does |
|---|---|---|---|
| `Providers` | `[]Provider` | empty | Providers configured at startup |
| `PublicBaseURL` | `string` | none | Externally reachable base URL. Required for SAML |
| `AllowedReturnOrigins` | `[]string` | localhost | Origins the browser may be redirected back to |
| `SessionTokenTTL` | `time.Duration` | `1h` | Lifetime of a session created by SSO |
| `SessionRefreshTTL` | `time.Duration` | `30d` | Lifetime of that session's refresh token |

## Settings

| Key | Default |
|---|---|
| `sso.session_token_ttl_seconds` | `3600` |
| `sso.session_refresh_ttl_seconds` | `2592000` |

## Endpoints

| Method | Path | Purpose |
|---|---|---|
| `POST` | `/v1/sso/login` | Start SSO by email domain |
| `POST` | `/v1/sso/:provider/login` | Start SSO against a named provider |
| `POST` | `/v1/sso/:provider/acs` | SAML assertion consumer service |
| `GET` | `/v1/sso/:provider/metadata` | SAML service provider metadata |
| `GET` | `/v1/sso/:provider/callback` | OIDC redirect landing |
| `POST` | `/v1/sso/exchange` | Trade the one-time code for a session |
| `GET` | `/v1/admin/sso/connections` | List connections for an app |
| `POST` | `/v1/admin/sso/connections` | Create a connection |
| `GET` | `/v1/admin/sso/connections/:connectionId` | Read a connection |
| `PUT` | `/v1/admin/sso/connections/:connectionId` | Update a connection |
| `DELETE` | `/v1/admin/sso/connections/:connectionId` | Delete a connection |

## Lifecycle hooks

None. It is a `RouteProvider`, a `SettingsProvider` and a `MigrationProvider` for its
connection tables.

## Related

`scim` is the other half of the enterprise story: SSO handles who can get in, SCIM keeps the
user list in sync. `organization` provides the tenants that connections are scoped to.
