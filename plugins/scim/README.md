# scim

SCIM 2.0 provisioning, so a customer's identity provider can create, update and deactivate
users in your product automatically.

An enterprise admin who removes someone in Okta expects them gone from every connected app
within minutes. SCIM is how that message gets delivered. The plugin exposes the standard
`/Users` and `/Groups` resources plus the discovery endpoints identity providers probe on
setup, and authenticates callers with a bearer token you issue per connection.

It ships disabled. Turn `scim.enabled` on deliberately, because these endpoints can create and
delete users.

## Use it when

- A customer's security review asks for automated deprovisioning
- Manual user management does not scale past a few hundred seats per tenant
- You already run `sso` and users are being created on first login with no way to remove them

## Skip it when

- Your customers are small enough to manage users by hand
- You cannot honour a deactivation quickly. An admin who sees the user removed in Okta will assume they lost access to you too, and stop checking
- You have not decided whether a SCIM delete means soft-suspend or hard-delete in your data model

## Wiring

```go
authsome.WithPlugin(scim.New(scim.Config{
    BasePath:      "/scim/v2",
    MaxLogEntries: 1000,
}))
```

Then set `scim.enabled` to `true` per app once a connection is configured.

## Config

| Field | Type | Default | What it does |
|---|---|---|---|
| `BasePath` | `string` | `/scim/v2` | Where SCIM endpoints are mounted |
| `TokenLength` | `int` | `32` | Random bytes in a generated bearer token |
| `MaxLogEntries` | `int` | `1000` | Provision log entries kept per config |

## Settings

| Key | Default |
|---|---|
| `scim.enabled` | `false` |
| `scim.auto_create_users` | `true` |
| `scim.auto_suspend_users` | `true` |
| `scim.group_sync` | `false` |
| `scim.default_role` | `member` |
| `scim.token_expiry_days` | `365` |

## Endpoints

Mounted under `BasePath`, `/scim/v2` by default.

| Method | Path | Purpose |
|---|---|---|
| `GET` | `/Users` | List and filter users |
| `POST` | `/Users` | Create a user |
| `GET` | `/Users/:userId` | Read a user |
| `PUT` | `/Users/:userId` | Replace a user |
| `PATCH` | `/Users/:userId` | Partially update a user |
| `DELETE` | `/Users/:userId` | Deprovision a user |
| `GET` | `/Groups` | List groups |
| `POST` | `/Groups` | Create a group |
| `GET` | `/Groups/:groupId` | Read a group |
| `PUT` | `/Groups/:groupId` | Replace a group |
| `PATCH` | `/Groups/:groupId` | Partially update a group |
| `DELETE` | `/Groups/:groupId` | Delete a group |
| `GET` | `/ServiceProviderConfig` | Advertise supported features |
| `GET` | `/ResourceTypes` | Advertise resource types |
| `GET` | `/Schemas` | Advertise schemas |

## Lifecycle hooks

None. It is a `RouteProvider`, a `SettingsProvider` and a `MigrationProvider`.

## Related

`sso` handles authentication for the same customers. `organization` supplies the tenant that a
SCIM connection belongs to, and group sync maps onto its teams.
