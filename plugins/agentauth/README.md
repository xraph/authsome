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
agent and other delegated sessions, so an agent cannot manage its own delegation.

| Method | Path | Purpose |
|---|---|---|
| `GET` | `/v1/me/agents` | List agents acting on my behalf |
| `DELETE` | `/v1/me/agents/:id` | Revoke an agent's delegation |
| `GET` | `/v1/admin/agents` | List registered agents |
| `POST` | `/v1/admin/agents` | Register an agent |
| `PATCH` | `/v1/admin/agents/:id/status` | Approve or block an agent |
| `GET` | `/v1/admin/agents/policy` | Read the org's delegation policy |
| `PUT` | `/v1/admin/agents/policy` | Set the org's delegation policy |

## What an agent session can't do

An agent session carries its delegating user's ID, so any route that only reads the user ID sees
the agent as that user. That's why step 4 matters for your own routes. Authsome's own routes
that mint something in the user's name refuse an agent session outright with a 403, whatever the
user is allowed to do, because what they mint would outlive the grant:

| Route | What it would have handed the agent |
|---|---|
| `POST /v1/keys` | A durable API key owned by the user |
| `GET /v1/oauth/authorize`, `GET` and `POST /v1/oauth/consent`, `POST /v1/oauth/device/complete` | An ordinary user token for a client the agent controls, with no `pk` claim |
| `/v1/admin/oauth/*` | A client secret |
| `POST /v1/admin/impersonate/:userId` | A session as any user in the app |
| `POST /v1/admin/service-accounts/:id/api-keys` | A service-account key |
| `POST /v1/admin/users/create` | A new account with a password the agent picked |
| `POST /v1/change-password` | A password it knows |
| `POST /v1/mfa/enroll`, `DELETE /v1/mfa/enrollment`, `POST /v1/mfa/recovery/regenerate` | A second factor or recovery codes, or MFA turned off |
| `POST /v1/passkeys/register/begin` and `/finish` | A passkey that signs in as the user |
| `POST /v1/me/switch-org` | A token for another org the user belongs to |
| `/v1/me/agents`, `/v1/admin/agents` | Control over its own delegation |

The dashboard won't accept an agent token either. The same refusal covers a session minted by
token exchange, whose actor chain names the principal that's really calling. An admin's
impersonation session isn't refused, and neither is a service account acting as itself.

`middleware.DenyDelegatedSession()` is the guard most of these use (org switch checks the session in the engine). Mount it on a route of your own
that mints a credential, after the middleware that resolves the session. `Engine.Impersonate` and
`Engine.ChangePassword` return `authsome.ErrDelegatedSession` when the context carries such a
session, for callers that don't come in over HTTP.

Everything else under `/v1` still treats the agent as its user, including the rest of
`/v1/admin`. If your users delegate to agents, keep that in mind when you hand someone an admin
role.

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
