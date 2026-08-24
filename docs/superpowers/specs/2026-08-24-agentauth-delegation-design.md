# Agent delegation (agentauth): design

**Status:** draft
**Date:** 2026-08-24
**Owner:** Rex Raphael

## Context

Someone points an AI agent at your product and tells it to go do something on their behalf. Today authsome gives you two ways to allow that, and both are wrong.

Hand the agent a static API key and you have a long-lived bearer secret with no owner, no consent record and no expiry story. It is worse than it sounds, because API key traffic never reaches the risk plugins, so you can have `riskengine`, `impossibletravel`, `ipreputation`, `anomaly`, `geofence` and `vpndetect` all switched on and paying for themselves while every one of them sits there watching a credential it cannot see. Your other option is to mint the agent a session as the user, at which point it can do anything that user can do for as long as the session lives, your user has no way to tell it exists, and you cannot switch off that one agent without ending every session they have.

Neither answers the four questions a security review will ask you: who authorized this, on whose behalf, scoped to what, and until when.

Most of the parts needed to answer them are already in the tree.

## What already exists

Worth stating plainly, because it is easy to get wrong and the whole design leans on getting it right.

Authsome already has a non-human principal. `serviceaccount.ServiceAccount` is described in its own package doc as "a first-class alternative to impersonating fake user rows", `session.Session` carries a `PrincipalKind` field taking `"user"` or `"service_account"` with an empty value meaning `"user"` so that old rows keep working, `apikey.APIKey` carries a `ServiceAccountID`, and there is a TODO sitting at `rbac/warden_store.go:326` that reserves the idea of a Warden service-account subject kind for whenever somebody needs it.

So the extension point is built. What you do not get is delegation. A service account has an `AppID` and no owner, which means "acting on behalf of user U" cannot be expressed at all, it authorizes by scope alone and never reaches Warden, it has no expiry and no consent record and nowhere for a user to go and look at it, and the only way to bring one into existence is an admin API call.

That gap is what this spec fills.

## Goals

- An agent principal that always has a human behind it, so every agent action is attributable to a person.
- An agent can never do more than the user who authorized it, and loses access the moment that user does.
- A user can see every agent acting for them and revoke any of them from one place.
- Org admins can decide whether their members may authorize outside agents at all.
- Grants expire on a hard cap and die when the person who made them is offboarded.
- Agent traffic is visible to the risk plugins, unlike API key traffic today.

## Non-goals

- Autonomous agents with no human behind them. Those stay service accounts. Keeping the two apart is deliberate and is what lets the type system enforce "an agent always has an owner".
- Resolving the Warden service-account TODO. This design does not touch `rbac/warden_store.go`.
- Instance-level delegation scopes. `invoices:read` means invoices, not customer 42's invoices. More on this below.
- Replacing or reimplementing anything in `oauth2provider`. This plugin sits on top of it.

## Decisions taken during brainstorming

Three questions were settled up front and everything else follows from them.

Registration is open but consent is gated per org. Anybody can self-register an agent, and whether a given org's members may then authorize it is that org's call, across three modes: open, allowlist, blocked.

Authorization is strict intersection, always. What an agent can do equals what its delegating user can do, intersected with the scopes that user granted, with no exceptions and no autonomous path.

Grants work offline, with a capped TTL and a hard offboard. An agent keeps working while its user is asleep, every grant carries a maximum lifetime, and deactivating a user or removing them from an org kills the grants they issued immediately.

## Architecture

An agent is an OAuth client. A grant is an authorization-code grant with delegation metadata hung off it. Agent tokens are OAuth access tokens.

That reuse is the whole point. `OAuth2Client` in `plugins/oauth2provider/models.go` is most of an agent record already, `AuthorizationCode` already carries a `UserID` and a scope list, and between the two of them you get PKCE, the device grant and metadata discovery without writing a line of any of it. It also means the RFC work happening in parallel, which is 7591 dynamic client registration, 8707 resource indicators, 8693 token exchange and DPoP, composes with this plugin instead of duplicating it.

What agentauth adds on top is the delegation semantics: the grant record, the org policy gate, intersection enforcement, the lifecycle, and a surface where users can see and revoke.

The plugin keeps its own tables and joins to `oauth2provider` by `ClientID`. It does not migrate that plugin's schema. Partly that's clean separation, and partly it's practical, because several other workstreams are editing those files right now.

## Data model

Three new entities, all in `plugins/agentauth/`.

```go
// Agent is a non-human principal that always acts for a human.
type Agent struct {
    ID          id.AgentID
    AppID       id.AppID
    OrgID       id.OrgID     // owning org; zero when self-registered
    ClientID    string       // soft FK to oauth2provider OAuth2Client.ClientID
    Name        string
    Description string
    LogoURI     string       // shown on the consent screen
    Origin      AgentOrigin  // self_registered | org_registered | first_party
    Status      AgentStatus  // pending | approved | blocked
    CreatedBy   id.UserID    // admin who registered it; zero if self-registered
    CreatedAt, UpdatedAt time.Time
}

// AgentGrant is one user's delegation to one agent.
type AgentGrant struct {
    ID         id.AgentGrantID
    AppID      id.AppID
    AgentID    id.AgentID
    UserID     id.UserID     // never zero
    OrgID      id.OrgID
    Scopes     []string
    ConsentID  id.ConsentID  // links to the consent plugin's record
    ExpiresAt  time.Time     // a value, not a pointer
    LastUsedAt *time.Time
    RevokedAt  *time.Time
    CreatedAt, UpdatedAt time.Time
}

// OrgAgentPolicy is the per-org gate.
type OrgAgentPolicy struct {
    OrgID         id.OrgID
    Mode          PolicyMode    // open | allowlist | blocked
    MaxGrantTTL   time.Duration // ceiling on AgentGrant.ExpiresAt
    AllowedScopes []string      // ceiling on what a member may delegate
}
```

Two of those field choices are load-bearing. `AgentGrant.UserID` is never the zero value and `ExpiresAt` is a value instead of a pointer, so the two invariants you agreed to are enforced by the struct and not by a code review.

Uniqueness: one active grant per `(AgentID, UserID, OrgID)`. Re-consenting updates the existing row the way `consent.GrantConsent` already does.

### Changes to session.Session

```go
PrincipalKind = "agent"   // joins "user" and "service_account"
AgentID   id.AgentID      // the acting agent
GrantID   id.AgentGrantID // so revoking a grant can find its sessions
```

`UserID` stays populated with the delegating human. This is a departure from how service accounts work and it's worth explaining, because the departure is the point.

Service-account sessions leave `UserID` at zero. That costs you a null check every time anything touches a session, and you can read the bill straight off the tree at `engine_session_roles.go:173`, again at `:190`, and again at `middleware/auth.go:570`, with every future consumer of a session obliged to remember the same rule or quietly ship a bug. An agent session with a real `UserID` needs none of that. Audit records, org resolution and RBAC all attribute to the human with no new branch, and the agent rides along as extra metadata, which is exactly how `ImpersonatedBy` already behaves.

You get one more thing for free. `store.DeleteUserSessions(userID)` already exists and is already called when a user is deleted, so it kills agent sessions with no change at all.

## Scopes and how they reach Warden

The host app declares its delegation vocabulary when it constructs the plugin.

```go
agentauth.New(
    agentauth.WithScope("invoices:read",  agentauth.Grants("read",  "invoice")),
    agentauth.WithScope("invoices:write", agentauth.Grants("write", "invoice")),
)
```

At request time, for a route that requires some `(action, resource)`:

1. Resolve the session, see `PrincipalKind == "agent"`, load the grant.
2. Refuse if the grant is expired or revoked, if the agent isn't approved, or if org policy has since flipped to blocked.
3. Scope gate. Does a granted scope map to this `(action, resource)`? If not, `403 insufficient_scope`.
4. User gate. Call `HasPermission(grant.UserID, action, resource)`, which is the existing `PermissionChecker` and is unchanged.

Both have to pass. That conjunction is the intersection, and it needs no new Warden subject kind, which is why the TODO at `rbac/warden_store.go:326` stays where it is.

Two limits you should know about before you build on this.

Scopes are type-level. `invoices:read` means invoices, not one customer's invoices, and the narrowing you probably wanted comes out of the user gate instead, because Warden's ReBAC has already worked out which invoices that particular user may read and your agent inherits every one of those constraints for nothing. What you cannot do is give an agent less than its owner at instance granularity. If you need that later, it is a change to the scope grammar and not a rearchitecture.

Unknown scopes fail at consent, not at request time. An agent asking for a scope with no registered mapping is turned away when the user tries to approve it, so a stored grant can never contain a scope that quietly means nothing.

## Flows

### Registration

Self-registration goes through the RFC 7591 endpoint and produces `Agent{Origin: self_registered, OrgID: zero, Status: pending}`, rate-limited through the existing `ratelimit` package. On its own it grants nothing. An agent with no grant can reach no data.

Org registration is an admin creating `Agent{Origin: org_registered, OrgID: O, Status: approved}` through the plugin's admin routes, behind `plugin.AdminGuard`.

The org policy gate does not run here. Registration is app-global, and "may this agent touch our data" only means something once a particular member of a particular org tries to authorize it.

### Consent

When user U in org O authorizes agent A for scopes S:

1. Load `OrgAgentPolicy(O)`. Blocked refuses. Allowlist requires an approved Agent row scoped to O. Open proceeds.
2. Every scope in S needs a registered Warden mapping and has to sit inside `policy.AllowedScopes`.
3. `ExpiresAt = min(requested, policy.MaxGrantTTL, global default)`.
4. Write the `AgentGrant` and a linked `consent.Consent` record.
5. Hand off to the existing authorization-code issuance.

Step 5 needs one thing `oauth2provider` doesn't have yet, an exported gate:

```go
type ConsentGate interface {
    Evaluate(ctx context.Context, clientID string, userID id.UserID,
             orgID id.OrgID, scopes []string) error
}
oauth2provider.New(oauth2provider.WithConsentGate(gate))
```

agentauth registers itself as the gate during `OnInit`. Roughly thirty additive lines, and an interface keeps agentauth out of that plugin's internals while four other branches are editing it.

### Tokens

Access tokens stay short, fifteen minutes by default. They carry `sub` set to the delegating user, plus `agent_id`, `grant_id` and the granted scopes. The session row lands with `PrincipalKind = "agent"`.

Refresh is where the TTL cap actually bites. Refresh checks `grant.ExpiresAt` and fails once the grant has lapsed. Skip that and you've built a permanent credential with a decorative expiry field on it.

Issuance goes through the normal hook path and emits `BeforeSessionCreate` and `AfterSignIn`, which is what puts agent traffic in front of `riskengine`, `impossibletravel` and the rest. This is the one place the design deliberately refuses to copy the API key plugin, which builds a synthetic `session.Session` by hand at `plugins/apikey/plugin.go:567` and therefore fires none of those hooks. If you take one shortcut here you lose the risk visibility this spec lists as a goal, so it is worth a test of its own.

### Revocation and offboarding

| Trigger | Hook | Action |
|---|---|---|
| User revokes from `/v1/me/agents/:id` | direct | set `RevokedAt`, delete sessions by `GrantID` |
| Admin blocks an agent | direct | revoke every grant for that agent in the org |
| User banned or soft-deleted | `AfterUserUpdate` / `AfterUserDelete` | revoke all grants that user issued |
| User removed from an org | `AfterMemberRemove` | revoke that user's grants scoped to that org |
| Grant TTL lapses | none | refresh refuses, request-time check refuses |

Every hook in that table already exists on the plugin bus, so none of it needs an engine change.

Sessions and grants both have to go. Kill the sessions on their own and the next refresh simply mints a fresh one, because a session is only the thing carrying the access while the grant is what keeps on authorizing it.

## Error handling

| Condition | Response |
|---|---|
| Grant missing, revoked or expired | `401` with `WWW-Authenticate: error="invalid_token"` |
| Granted scopes don't cover the route | `403` with `error="insufficient_scope", scope="invoices:read"` |
| Delegating user lacks the permission | `403` generic, matching `middleware/rbac.go:62` |
| Agent blocked, or org policy now denies | `403` with a distinct code |

Check the scope gate before the user gate. It is cheaper, being a map lookup ahead of a Warden call, and it happens to be the safe order too, because telling an agent `insufficient_scope` gives away only what that agent was granted and therefore already knows. Run the user gate first and report its failure honestly and you have handed the agent a way to enumerate its owner's permissions one probe at a time.

The agent path fails closed. `plugin.PermissionGuard` returns nil when the engine has no `PermissionChecker`, on purpose, so an engine without RBAC still gets session enforcement. For agents that fallback is a hole, because the user gate is the security model. No permission checker means deny. It runs against an existing convention in this repo, so the code needs a comment saying why.

## Caching

The per-request grant lookup would otherwise double store reads on every agent call. Cache the grant in-process keyed by `GrantID`, holding the grant plus agent status and org policy, with a short TTL as a backstop.

The usual worry about caching authorization is revocation latency, and here it mostly disappears. Revoking a grant already deletes the sessions that grant issued, and session resolution runs on every single request well before the grant is ever consulted, so a stale cached grant is unreachable by construction. Session deletion is the invalidation point.

What the TTL still covers is the slower stuff, a grant ageing past `ExpiresAt` or an org flipping to blocked. Those tolerate a few seconds of lag in a way revocation doesn't. Multi-node deployments carry the same bounded staleness and it belongs in the docs, not in a footnote.

## Testing

Integration tests go through `testutil.NewTestServer(t, WithPlugins(...))`, which already gives you org and member helpers.

Unit coverage: the scope mapping table, the three-way TTL clamp, and the policy mode matrix.

These carry the security claims and get written first, failing, before any implementation:

- Grant `invoices:write` to an agent whose owner has no write permission, expect `403`. This is the intersection property. If it ever passes, the feature is broken.
- Revoke the owner's permission mid-session, expect the agent to lose it on the next request. This proves agent authorization never rode the stamped-roles fast path.
- Ban the user, expect grants revoked and sessions gone and refresh failing. All three, because any one on its own leaves a live credential.
- Remove a user from org O, expect their O-scoped grants dead and their other orgs' grants alive.
- Refresh after `ExpiresAt`, expect refusal.
- Consent with an unmapped scope, expect rejection at consent.
- Org policy set to blocked, expect consent refused even for an already-approved agent.
- Issue an agent token with a recording plugin registered, expect `BeforeSessionCreate` and `AfterSignIn` to have fired. This is the test that stops somebody optimizing agent issuance into a synthetic session later and silently taking the risk plugins offline.

Store parity: the same suite across pg, sqlite, mongo and memory, following the per-backend pattern the other plugins use.

## Dependencies

RFC 7591 dynamic client registration is a hard dependency for the self-registration path. Without it, only org-registered agents work, which is a usable v1 if that work slips.

The `ConsentGate` interface in `oauth2provider` is a hard dependency and has to land first.

RFC 8707 resource indicators, RFC 8693 token exchange and DPoP all compose with this design and none of them block it.

## Open question

Agent requests pay a Warden lookup on every request. Human requests deliberately avoid that by reading stamped roles off the session, and the comment on `Session.Roles` spells out the reasoning: anything that has to be current at the instant it is checked belongs in a permission check and not on a session row. An agent grant is exactly that case, so the lookup is the correct call. Whether you can afford it at your traffic depends on Warden's own caching, and it is worth measuring before any of this ships.
