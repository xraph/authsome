# Non-human principal enforcement

Status: design, revised 2026-08-24 to layer on the `principal` package
Layers on: [non-human principals in the authsome core](2026-08-24-non-human-principals-design.md)
Prerequisite for: [workload identity federation](2026-08-24-workload-identity-federation-design.md)

## Why this exists

A service account can hold a session, carry roles, resolve cleanly from its
token, and still get a 401 from every route that guards itself. Three places
enforce authentication, all three resolve a user, and all three treat a caller
without one as a caller who failed.

This is a narrow spec and it deliberately stays narrow. The `principal` package
design covers the domain model: kinds, refs, the actor chain, Warden, the store
work. It does not cover the three rejections below, and its plan doesn't either.
Grep `docs/superpowers/plans/` for `RequireAuth`, `SessionProvider`,
`ErrAuthenticationFailed` or `MustParse` and the only hits are route
registrations in the agentauth plan. So this document is the enforcement layer
that sits on top of that one, and it is small on purpose.

There's a specific claim in the principal design worth reading against this: it
says `UserFrom(ctx)` and every handler reading it are untouched. That's true and
it's the right call. What it misses is that `RequireAuth` gates on `UserFrom`
before any handler runs, so machine traffic gets rejected at the door of the
very routes that design is trying to make visible to the risk plugins.

## How this relates to the `principal` package

Use their abstraction, don't build a parallel one. Concretely:

The context helper is theirs. `principal.FromContext(ctx)` and
`middleware.PrincipalFrom(ctx)` are already specified, and everything below
reads through those instead of a private helper. An earlier draft of this spec
proposed an unexported `principalFrom`, which would have been a second answer to
a question already answered.

Principal kinds are theirs too. `KindUser`, `KindAgent`, `KindWorkload` and
`KindService` all exist in that design, so nothing here should compare
`PrincipalKind` against a bare `"service_account"` string. The whole point of
that package is that the bare string comparison currently sitting in five places
becomes one typed check.

Ordering is flexible. If the `principal` package lands first, this spec is
written against it directly. If it slips, every fix below can be written against
`session.PrincipalKind` and adapted later, because the change in each case is
which value you read and not what you do about it.

## The three surfaces

### SessionProvider rejects a nil user

`authprovider.SessionProvider.Authenticate` resolves the session, resolves the
user from it, and returns `auth.ErrAuthenticationFailed` when that fails
(`authprovider/session.go:109`). A non-human session has a nil `UserID`, so
`resolveUser("")` fails and every route registered with
`forge.WithGroupAuth("session")` or `plugin.SessionGuard` answers 401.

Fix: branch before the user lookup. When the session's principal is not a user,
build the `AuthContext` with the principal's ID as `Subject`, the session's
stamped `Roles`, and `Data` carrying the session with a nil `User`. Skip
`resolveUser`. Put the principal kind and ID into `Claims` so a downstream
authorizer can tell the kinds apart without unwrapping `Data`.

`BridgeToContext` already guards on `data.User != nil`
(`authprovider/session.go:200`), so it needs one change and not a rewrite: move
`WithAuthMethod` out of that guard. As written, a machine request reaches a
handler with no auth method recorded at all, which is confusing to debug and
worse to log.

### RequireAuth gates on a user

`middleware.RequireAuth()` checks `UserFrom(ctx)` and 401s when it's absent
(`middleware/auth.go:661`). The strategy branch sets `WithUser` only when
`result.User != nil` (`middleware/auth.go:610`), which is never true for a
service account. That covers the nine `api/*_handlers.go` groups built on
`RequireAuth()`.

Fix: gate on the resolved principal instead of on the user, reading through
`middleware.PrincipalFrom`. A user principal satisfies it exactly as before, so
this is additive for human traffic. Nothing else in the strategy branch moves,
since it already puts the session on the context.

Write the check against any non-user principal and resist naming service
accounts in it. Three kinds are arriving at once between the principal design and
the agentauth work, and a check written for the general case picks all of them
up while one written for service accounts has to be found and widened by whoever
implements second.

### The JWT path panics on an empty subject

This one is worse than a 401. The JWT branch builds its context with
`id.MustParse(claims.UserID)` (`middleware/auth.go:425`), and `id.Parse("")`
returns an error (`id/id.go:218`), so `MustParse` panics. Nothing mints a JWT
with an empty `sub` today, which is why it has never fired. Workload identity
would be the first thing to do it.

Two fixes, and take both.

First, every `id.MustParse` on a claim in that branch becomes a checked parse
that refuses the token. There are four, covering `app_id`, `sub`, `sid`,
`env_id` and `org_id` (`middleware/auth.go:424` through `:444`). They run on
attacker-supplied claim values. A valid signature is required to reach them so
this isn't remotely exploitable today, but a panic in auth middleware is a bad
thing to leave lying around and the fix costs nothing.

Second, skip `resolveUser` when the claims say the subject isn't a user.

### TokenClaims can't say what kind of principal it carries

`tokenformat.TokenClaims` has no field for it (`tokenformat/format.go:18`). Add:

```go
PrincipalKind string `json:"pk,omitempty"`
PrincipalID   string `json:"pid,omitempty"`
```

For a non-human principal, `sub` carries the principal ID and does not sit
empty, so `sub` means "the caller" for every token the system mints and `pk`
tells the JWT branch how to resolve it. Both fields are `omitempty`, so a user
token is unchanged on the wire and existing tokens keep validating.

`PrincipalID` rather than a service-account-specific field, because the
principal design has four kinds and a token claim per kind doesn't scale. It
maps onto `principal.Ref` directly.

## What this does not change

Role stamping stays as it is. `roleStampingStore.shouldStamp` already skips
service accounts as authorized by scope (`engine_session_roles.go:190`), and
leaves any session that already carries roles alone (`engine_session_roles.go:187`).
Both behaviours are what workload identity needs.

There are no store changes, no migrations and no new tables anywhere in this
spec, and every line of it lands in one of three files:
`authprovider/session.go`, `middleware/auth.go` and `tokenformat/format.go`.
The store work, including the postgres and sqlite service-account methods that
currently return `not implemented`, belongs to the principal design and is
already scheduled there.

## Coordinating with the specs landing alongside this one

`tokenformat.TokenClaims` is about to be edited by two designs at once. The RFC
8693 token exchange draft adds an `Act *ActClaim` field for the actor chain and
serialises it through `customClaims` at `tokenformat/jwt.go:69`, and this spec
adds `pk` and `pid`. Nothing conflicts and every field is `omitempty`, so the
merge is mechanical, but whoever implements second should open the other spec
and add all the fields in one pass rather than finding the collision in a
rebase.

That draft also adds a `Scopes` column to `authsome_sessions`. Nothing here
depends on it. Workload identity does, so if these land close together the
column is worth doing once.

## Testing

The deliverable is one test, and everything above is what makes it pass.

Mint a non-human session directly through the store. Send it at a route guarded
by `SessionGuard`, and at one guarded by `RequireAuth()`. Assert 200 on both.
Run the same table with the app's token format set to `jwt`, which is the case
that panics today.

Then the smaller ones:

- A JWT carrying a malformed `app_id`, `sid`, `env_id` or `org_id` is refused
  with a 401 and does not panic. One case per field.
- A non-human `AuthContext` carries the principal ID as `Subject` and the
  session's roles, so a route declaring `forge.WithAnyRole` behaves.
- A user session still authenticates unchanged through all three surfaces. This
  is the regression guard and it matters more than the new cases.

Run the first test against mongo. Postgres and sqlite cannot create a service
account at all until the principal design lands its store work, so on those two
backends the fixture has to write the session row directly.

## Risks

The change to `SessionProvider.Authenticate` means `SessionData.User` can now be
nil where callers previously assumed otherwise. `BridgeToContext` handles it,
but it may not be the only consumer. Audit every use of `SessionData` before
writing the branch, and if something else dereferences `.User`, fix it here
rather than leaving the plugin work to trip over it.

The larger risk is coordination, not code. Three designs now touch
`middleware/auth.go` and two touch `tokenformat.TokenClaims`. Whoever goes first
should say so on the others.
