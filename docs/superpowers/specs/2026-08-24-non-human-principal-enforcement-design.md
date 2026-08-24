# Non-human principal enforcement

Status: design, approved 2026-08-24
Prerequisite for: [workload identity federation](2026-08-24-workload-identity-federation-design.md)

## Why this exists

A service account can hold a session today and still get a 401 on every route
that guards itself. The data model landed, the enforcement path did not.

`serviceaccount.ServiceAccount` is a real entity with its own store.
`session.Session` carries `PrincipalKind` and `ServiceAccountID`, and the apikey
plugin already mints service-account sessions with a nil `UserID`
(`plugins/apikey/plugin.go:566`). What none of that did was teach the three
places that actually enforce authentication to accept a principal that is not a
user. Each of them resolves a user and treats failure as failure.

So the credential exists, carries roles, resolves from its token, and then dies
at the guard. You can confirm the gap by grepping for a test that sends a
service-account credential at a protected route. There isn't one.

This spec closes that. It is small, it is entirely inside existing files, and it
is a prerequisite for workload identity federation, which otherwise issues a
credential with nowhere to spend it.

## The three surfaces

### SessionProvider rejects a nil user

`authprovider.SessionProvider.Authenticate` resolves the session, then resolves
the user from it, and returns `auth.ErrAuthenticationFailed` when that lookup
fails (`authprovider/session.go:102`). A service-account session has a nil
`UserID`, so `resolveUser("")` fails and every route registered with
`forge.WithGroupAuth("session")` or `plugin.SessionGuard` answers 401.

Fix: branch before the user lookup. When `sess.PrincipalKind` is
`"service_account"`, build the `AuthContext` with the service account ID as
`Subject`, the session's stamped `Roles`, and `Data` carrying the session with a
nil `User`. Skip `resolveUser` entirely. Add `principal_kind` and
`service_account_id` to `Claims` so downstream authorizers can tell the two
kinds apart without unwrapping `Data`.

`BridgeToContext` already guards on `data.User != nil`
(`authprovider/session.go:201`), so it needs one change and not a rewrite: move
`WithAuthMethod` out of that guard. Right now a service-account request
would reach a handler with no auth method recorded at all, which is a confusing
thing to debug and a worse thing to log.

### RequireAuth gates on a user

`middleware.RequireAuth()` checks `UserFrom(ctx)` and 401s when it's absent
(`middleware/auth.go:660`). The strategy branch sets `WithUser` only when
`result.User != nil` (`middleware/auth.go:613`), which is never true for a
service account. That covers the nine `api/*_handlers.go` groups built on
`RequireAuth()`.

Fix: introduce a small unexported `principalFrom(ctx)` helper, satisfied by
either a user or a session whose `PrincipalKind` is non-empty and not `"user"`.
`RequireAuth` gates on that instead. Nothing else in the strategy branch moves,
because it already puts the session in context.

Write that helper against any non-user principal and resist the urge to name
service accounts in it. The agent delegation design landing the same day adds a
third `PrincipalKind` of `"agent"`, so a helper written for the general case
picks that up for free while one written for service accounts specifically has
to be found and widened by whoever implements it second.

### The JWT path panics on an empty subject

This one is worse than a 401. The JWT branch builds its context with
`id.MustParse(claims.UserID)` (`middleware/auth.go:425`), and `id.Parse("")`
returns an error (`id/id.go:218`), so `MustParse` panics. Nothing mints a JWT
with an empty `sub` today, which is why it has never fired. Workload identity
would be the first thing to do it.

Two fixes, and take both.

First, every `id.MustParse` on a claim value in that branch becomes a checked
parse that refuses the token. There are four of them, covering `app_id`,
`sub`, `sid`, `env_id` and `org_id` (`middleware/auth.go:424` through
`middleware/auth.go:444`). They run on attacker-supplied claim values. A valid
signature is required to reach them, so this isn't remotely exploitable today,
but a panic in auth middleware is a bad thing to leave lying around and the fix
costs nothing.

Second, skip `resolveUser` when the claims say service account.

### TokenClaims needs to carry the principal kind

`tokenformat.TokenClaims` has no way to say what kind of principal a token
belongs to (`tokenformat/format.go:18`). Add two fields:

```go
PrincipalKind    string `json:"pk,omitempty"`
ServiceAccountID string `json:"sa,omitempty"`
```

For a service account, `sub` carries the service account ID and does not sit
empty. That way `sub` means "the principal" for every token the system mints,
and the JWT branch decides how to resolve it by reading `pk`. Both fields are
`omitempty`, so a user token is unchanged on the wire and existing tokens keep
validating.

## What this does not change

Role stamping stays exactly as it is. `roleStampingStore.shouldStamp` already
skips service accounts on the grounds that they're authorized by scope
(`engine_session_roles.go:190`), and it leaves any session that already carries
roles alone. Both behaviours are what workload identity needs, so leave them.

No store changes, no migrations, no new tables. Every change here is in
`authprovider/session.go`, `middleware/auth.go` and `tokenformat/format.go`.

## Coordinating with the specs landing alongside this one

`tokenformat.TokenClaims` is about to be edited by two designs at once. The RFC
8693 token exchange draft adds an `Act *ActClaim` field for the actor chain and
serialises it through `customClaims` at `tokenformat/jwt.go:69`, and this spec
adds `pk` and `sa`. The fields don't conflict and all of them are `omitempty`,
so the merge is mechanical, but whoever implements second should open the other
spec and add all the fields in one pass instead of discovering the collision in
a rebase.

The same draft also adds a `Scopes` column to `authsome_sessions`. Nothing here
depends on it, though workload identity does, so if these land close together
the column is worth doing once.

## Testing

The deliverable is one test, and everything above is what makes it pass.

Mint a service-account session directly through the store. Send it at a route
guarded by `SessionGuard`, and at a route guarded by `RequireAuth()`. Assert 200
on both. Run the same table with the app's token format set to `jwt`, which is
the case that panics today.

Then the smaller ones:

- A JWT carrying a malformed `app_id`, `sid`, `env_id` or `org_id` is refused
  with a 401 and does not panic. One case per field.
- A service-account `AuthContext` carries the service account ID as `Subject`
  and the session's roles, so a route declaring `forge.WithAnyRole` behaves.
- A user session still authenticates unchanged through all three surfaces. This
  is the regression guard and it matters more than the new cases.

## Risks

The change to `SessionProvider.Authenticate` means `SessionData.User` can now be
nil where callers previously assumed it wasn't. `BridgeToContext` handles it,
but it is not necessarily the only consumer. Audit every use of `SessionData`
before writing the branch, and if something else dereferences `.User`, fix it in
this spec, so the plugin work doesn't trip over it later.

Beyond that the blast radius is small and the regression surface is well covered
by the existing auth tests.
