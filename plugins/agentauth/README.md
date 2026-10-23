# agentauth

Lets a user hand an AI agent scoped, expiring, revocable access to act on their behalf.

An agent session is not a copy of the user's session. What the agent can do is the
intersection of two things: the scopes the human granted it, and what that human is allowed to
do anyway. Narrow the human's role and every agent acting for them narrows with it. Revoke the
grant and the agent stops, without touching the human's own access.

Agent sessions live 15 minutes and cannot be refreshed. `Engine.Refresh` refuses to rotate a
session whose principal kind is `agent`, on purpose, so no agent credential can outlive the
grant that authorised it. You re-issue instead.

## Read this before you wire it

Registering the plugin enforces nothing. It is a set of primitives, not a finished feature, and
a host application has to do five things before an agent session carries any real restriction.
Until all five are done, an agent that authenticates through the ordinary OAuth2 flow gets a
plain human session with that user's full access.

1. Point `oauth2provider` at the consent gate, so org policy is consulted when a user approves
   an agent. `OnInit` does this for you when an `oauth2provider` plugin is registered on the
   same engine. Wire `oauth2provider` yourself, or register it after agentauth, and you must
   call `oauth2provider.SetConsentGate` by hand.
2. Call `CreateGrant` from your own consent handler and persist the returned `AgentGrant.ID`.
   There is no consent screen in this package, only the primitive behind one.
3. Mint the credential with `IssueAgentSession`. The standard OAuth2 token endpoint knows
   nothing about this package and will hand back an ordinary human session.
4. Mount `Guard(action, resource)` on every route an agent can reach, **in addition to** your
   normal RBAC check. `middleware.RequirePermission` is not principal-aware: it resolves the
   permission of `sess.UserID`, which for an agent session is the delegating human. A route
   with `RequirePermission` alone gives the agent that human's full access. `Guard` alone gives
   it no RBAC check at all. You need both.
5. Re-issue before expiry. There is no long-lived agent credential. The grant is the durable
   thing; the session is not.

## Use it when

- You are letting agents call your API for a user and "the user's own token" is too much access
- You need to show a person which agents act for them, and give them a revoke button
- An org admin needs to approve or block agents across the whole tenant

## Skip it when

- Your agents are first-party and act as themselves, not for a user. `apikey` or OAuth2 client credentials fit that
- You cannot audit every route an agent reaches. The intersection only holds where `Guard` is mounted, and a single unguarded route undoes it
- You want it working by registering the plugin. Read the five steps above first

## Wiring

```go
agents := agentauth.New(
    agentauth.WithDefaultGrantTTL(30*24*time.Hour),
    agentauth.WithScope("calendar.read", agentauth.Permission{Action: "read", Resource: "calendar"}),
)

engine, err := authsome.NewEngine(
    authsome.WithStore(store),
    authsome.WithPlugin(oauth2provider.New(oauth2provider.Config{Issuer: "https://auth.example.com"})),
    authsome.WithPlugin(agents),
)
```

Then guard the routes:

```go
router.GET("/v1/calendar", handler,
    middleware.RequirePermission(engine, "read", "calendar"), // human RBAC
    agents.Guard("read", "calendar"),                         // agent scope intersection
)
```

## Options

| Option | What it does |
|---|---|
| `WithStore(s Store)` | Use a specific store. Defaults to the engine store |
| `WithDefaultGrantTTL(d time.Duration)` | How long a new grant lasts before it expires |
| `WithScope(scope string, p Permission)` | Map a scope string onto an action and resource pair |

## Endpoints

User surface under `/v1/me/agents`, admin surface under `/v1/admin/agents`. Both refuse
delegated sessions, agents included, so an agent cannot manage its own delegation.

| Method | Path | Purpose |
|---|---|---|
| `GET` | `/v1/me/agents` | List agents acting on my behalf |
| `DELETE` | `/v1/me/agents/:id` | Revoke an agent's delegation |
| `GET` | `/v1/admin/agents` | List registered agents |
| `POST` | `/v1/admin/agents` | Register an agent |
| `PATCH` | `/v1/admin/agents/:id/status` | Approve or block an agent |
| `GET` | `/v1/admin/agents/policy` | Read the org's delegation policy |
| `PUT` | `/v1/admin/agents/policy` | Set the org's delegation policy |

## Deny by default for delegated sessions

A delegated session acts for a user without being that user's own sign-in. An agent session is
one: it carries its delegating user's ID, so any route that only reads the user ID sees the agent
as that user. A session minted by token exchange is the other, with an actor chain naming the
principal that's really calling.

An agent's authority comes from its grant and nowhere else. So on every authsome route, core and
plugin alike, a delegated session gets a 403 on `POST`, `PUT`, `PATCH` and `DELETE` unless the
route is marked agent-safe. `GET`, `HEAD` and `OPTIONS` stay open, so an agent can still read
`GET /v1/me` and anything else its user can read. A route someone adds tomorrow is refused until
someone marks it, which is the point.

Two authsome routes are agent-safe, because the caller can only touch its own credential there:

| Route | Why it's safe |
|---|---|
| `POST /v1/signout` | Ends the caller's own session |
| `POST /v1/oauth/revoke` | Revokes a token the caller itself holds |

Everything else an agent does goes through your routes and `Guard`. Token exchange, refresh and
the OAuth2 token endpoint aren't on the list: an agent session never reaches them with a session
on the request (a workload exchanging tokens acts as itself, and `Engine.Refresh` refuses agent
sessions anyway).

To mark a write route agent-safe, add `plugin.AgentSafe()` (or `middleware.AgentSafe()`) to its
options. For a route that enforces the grant, use `GuardRoute`, which mounts `Guard` and the
marker together:

```go
router.POST("/v1/invoices/:id/approve", handler,
    append(agents.GuardRoute("approve", "invoice"),
        forge.WithMiddleware(middleware.RequirePermission(engine, "approve", "invoice")))...,
)
```

`Guard` on its own leaves a write route closed to agents. The rule lives in
`middleware.GuardDelegatedWrites`, an interceptor on a router group. The API registers its routes
through one, and the Forge extension hands one to every plugin's `RegisterRoutes`. If you mount
authsome's plugins on a router yourself, pass them `middleware.GuardDelegatedWrites(router)`.

Some reads are refused too, because what they hand out would outlive the grant or the work is
admin work: `GET /v1/oauth/authorize` and the consent page, and every route under `/v1/admin`
along with the organization, waitlist and shared-signals admin routes. The dashboard won't take a
delegated token either, in engine mode or in client mode.

In client mode, `ClientAuthMiddleware` carries introspection's `delegated` field onto the request,
and on the JWT path an `act` claim does the same, so the rule holds there as well. Introspection
reports `principal_kind` and `delegated` for every active token. A client-mode service needs an
identity server that sends the field; an older one sends nothing, and its tokens read as the
user's own.

What makes a session delegated is its shape, not its kind: an agent or grant marker, a
delegation that isn't an impersonation, an actor chain, or a non-user principal that names a
user. An admin's impersonation session isn't refused. Neither is a principal acting as itself,
whatever its kind, so a service account registered as an agent, with no user behind it, keeps
working.

The older, narrower guards stay in place underneath: `middleware.DenyDelegatedSession()` on the
credential and admin routes, `middleware.RefuseDelegated` inside the handlers that write
credentials straight to the store, and `authsome.ErrDelegatedSession` from `Engine.Impersonate`,
`Engine.ChangePassword` and `Engine.AdminCreateUser`.

## Lifecycle hooks

| Hook | What happens |
|---|---|
| `AfterUserUpdate` | Re-checks grants when the delegating human's access changes |
| `AfterUserDelete` | Tears down grants belonging to a deleted user |
| `AfterOrgDelete` | Tears down grants belonging to a deleted org |
| `BeforeMemberRemove` | Handles delegations held through an org membership being removed |

It is also a `MigrationProvider` for its agent and grant tables, and a `RouteProvider`.

## Related

`oauth2provider` is where the consent gate plugs in. `organization` supplies the org whose
policy the admin surface reads and writes. [`doc.go`](doc.go) carries the full host integration
contract, and is the authority if this README and it ever disagree.
