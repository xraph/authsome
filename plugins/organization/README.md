# organization

Multi-tenancy: organisations, members, roles, invitations and teams.

Without this plugin registered, none of that exists. The core engine knows about users and
sessions and stops there. Add this and you get the whole B2B shape, where a person belongs to
one or more organisations, holds a role in each, and can be invited into one by email.

Teams sit inside organisations for finer-grained grouping, and the plugin contributes its own
dashboard pages and admin routes.

## Use it when

- You are selling to companies, so the account that pays you is not the person who logs in
- Permissions need to be scoped per tenant, where the same user is an admin in one org and a viewer in another
- You need invitation flows with accept and decline

## Skip it when

- Every user is an island. Consumer products rarely need this and it complicates every query you write
- You have exactly one tenant, which is most internal tools
- You want organisations later. Adding them after launch means backfilling ownership onto everything you already store

## Wiring

```go
authsome.WithPlugin(organization.New()),
```

Mount the routes somewhere other than the engine base path:

```go
authsome.WithPlugin(organization.New(organization.Config{
    PathPrefix: "/v1/tenants",
})),
```

## Config

| Field | Type | Default | What it does |
|---|---|---|---|
| `PathPrefix` | `string` | engine `BasePath` | Where organisation routes are mounted |

## Endpoints

| Method | Path | Purpose |
|---|---|---|
| `GET` | `/orgs` | List the caller's organisations |
| `POST` | `/orgs` | Create an organisation |
| `GET` | `/orgs/check-slug` | Check slug availability before creating |
| `GET` | `/orgs/:orgId` | Read an organisation |
| `PATCH` | `/orgs/:orgId` | Update an organisation |
| `DELETE` | `/orgs/:orgId` | Delete an organisation |
| `GET` | `/orgs/:orgId/members` | List members |
| `POST` | `/orgs/:orgId/members` | Add a member |
| `PATCH` | `/orgs/:orgId/members/:memberId` | Change a member's role |
| `DELETE` | `/orgs/:orgId/members/:memberId` | Remove a member |
| `GET` | `/orgs/:orgId/invitations` | List pending invitations |
| `POST` | `/orgs/:orgId/invitations` | Invite someone by email |
| `POST` | `/orgs/invitations/accept` | Accept an invitation |
| `POST` | `/orgs/invitations/decline` | Decline an invitation |
| `GET` | `/orgs/:orgId/teams` | List teams |
| `POST` | `/orgs/:orgId/teams` | Create a team |
| `GET` | `/orgs/:orgId/teams/:teamId` | Read a team |
| `PATCH` | `/orgs/:orgId/teams/:teamId` | Update a team |
| `DELETE` | `/orgs/:orgId/teams/:teamId` | Delete a team |

An admin variant of the list endpoint is mounted under `{PathPrefix}/admin`.

## Lifecycle hooks

None on the sign-in path. It registers a `DataExportContributor`, so a user's organisation
memberships land in their GDPR export.

## Related

`sso` and `scim` scope their connections to an organisation. `subscription` can bill per
organisation and auto-subscribe one on creation. `consent` covers the other half of the GDPR
story.
