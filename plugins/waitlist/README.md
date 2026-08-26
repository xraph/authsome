# waitlist

Gates signup behind an approval queue, for the stretch before you open the doors.

People join with an email and land in `pending`. You approve or reject from the admin routes,
and only approved addresses can complete signup. The gate runs on `BeforeSignUp`, so it holds
regardless of which auth method someone tries.

There is a stats endpoint, which is mostly there because everyone building a waitlist wants to
know how long it is.

## Use it when

- You are pre-launch and capacity is the constraint, not demand
- You want scarcity as part of the launch, with invites going out in batches
- You are onboarding design partners by hand and a stray public signup would be awkward

## Skip it when

- You are live and growing. A gate nobody remembers turning on will quietly refuse real signups
- You already gate by email domain. `password.allowed_domains` does that with no queue to manage
- Nobody is going to work the queue. An unapproved list is worse than no list

## Wiring

```go
authsome.WithPlugin(waitlist.New()),
```

`Enabled` controls whether the gate actually blocks. Turning it off leaves the endpoints up so
you can keep collecting emails without refusing signups.

## Config

| Field | Type | Default | What it does |
|---|---|---|---|
| `Enabled` | `bool` | `false` | When true, signup requires a prior approved entry |

## Endpoints

Public routes are under `{basePath}/waitlist`, admin routes under `{basePath}/admin/waitlist`.

| Method | Path | Purpose |
|---|---|---|
| `POST` | `/join` | Join the waitlist |
| `GET` | `/status` | Check an address's position and status |
| `GET` | `` | List entries (admin) |
| `GET` | `/stats` | Counts by status (admin) |
| `POST` | `/:entryId/approve` | Approve an entry (admin) |
| `POST` | `/:entryId/reject` | Reject an entry (admin) |
| `DELETE` | `/:entryId` | Delete an entry (admin) |

## Lifecycle hooks

| Hook | What happens |
|---|---|
| `BeforeSignUp` | Refuses signup when the address has no approved entry |

It also registers a `NotificationMappingContributor`, so approval and rejection can trigger
messages through the `notification` plugin, and a `MigrationProvider` for its table.

## Related

`notification` or `email` deliver the "you're in" message. `password` can restrict by domain
if that is the only gate you need.
