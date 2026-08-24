# OAuth 2.0 Dynamic Client Registration: design

**Status:** draft
**Date:** 2026-08-24
**Owner:** Rex Raphael

## Context

The `oauth2provider` plugin can only mint clients through the admin surface at
`POST /v1/admin/oauth/clients`, behind a session and a `manage:oauth2_client`
permission. That's fine when a human registers an integration ahead of time,
and it falls apart for MCP, where a client nobody has heard of shows up,
discovers the authorization server, registers itself, and starts an
authorization code flow without anyone touching a dashboard.

This spec adds RFC 7591 (Dynamic Client Registration), RFC 7592 (Dynamic Client
Registration Management), RFC 8414 (Authorization Server Metadata) and RFC 9728
(Protected Resource Metadata).

RFC 8414 wasn't in the original scope and belongs here anyway. Walk the chain an
MCP client actually follows and you'll see why: it fetches the protected
resource metadata, reads `authorization_servers`, fetches the authorization
server metadata from that issuer, reads `registration_endpoint`, and only then
registers. Today the server publishes `/.well-known/openid-configuration` and
nothing else, so the middle link is missing and 9728 on its own would dead-end
on the very first hop that matters.

RFC 8707 resource indicators and DPoP have no code yet, but both have designs
dated the same day as this one, and both add columns to
`authsome_oauth2_clients`. The resource indicators design also claims migration
version `20260824000001` in the `authsome-oauth2` group, so this one takes
`20260824000030` instead. The model changes below are additive and sit
alongside theirs; see the data model section for what to expect when two of
these land in either order.

DPoP names the missing RFC 8414 endpoint as a known gap and leaves it to its
own change. This is that change, so landing the metadata work here unblocks
DPoP advertising `dpop_signing_alg_values_supported` where a client will
actually look for it.

## Goals

- A stock MCP client can discover the server and register itself with no
  operator configuration beyond turning the feature on.
- A dynamically registered client is strictly less capable than an
  admin-created one, and that difference is enforced at registration and again
  on every update.
- Registration is off by default, so upgrading doesn't hand anyone an open
  endpoint they didn't ask for.
- All four stores stay behaviourally identical, following the device-code
  method pattern.

## Non-goals

- Initial access tokens (RFC 7591 section 3.1). Registration is open or off,
  and there is no third mode.
- Software statements (RFC 7591 section 2.3). Nothing consumes them yet.
- Making OAuth scope an enforced authority boundary. It's advisory today and it
  stays advisory here, for reasons covered under "What scope actually does".
- Sector identifiers, `request_object_signing_alg`, and the rest of the OIDC
  Dynamic Client Registration extensions beyond what 7591 defines.
- Sweeping unused dynamic registrations on a TTL.

## Configuration

Five new fields on the plugin's existing `Config`:

```go
// DynamicRegistration enables RFC 7591 registration. Off by default: an
// upgrade must never open a public registration endpoint on its own.
DynamicRegistration bool

// RegistrationAppID is the app dynamic clients belong to when the request
// carries no resolvable publishable key. Leave empty on a multi-tenant
// deployment so an unkeyed request is refused rather than pooled.
RegistrationAppID string

// DynamicRegistrationScopes is the allowlist a dynamic client's requested
// scopes are intersected against. Defaults to openid, profile, email and
// offline_access.
DynamicRegistrationScopes []string

// RegistrationRateLimit caps POST /register per client IP.
// Defaults to 10 per hour.
RegistrationRateLimit RateLimit

// ProtectedResources declares additional RFC 9728 resource identifiers,
// keyed by the path suffix they are served under. AuthSome always
// describes itself at the unsuffixed path regardless of this map.
ProtectedResources map[string]ProtectedResource
```

## Security decisions

These were settled during brainstorming and are recorded as decisions rather
than options, because every one of them is load bearing.

### Which app owns a dynamic client

Registration resolves the app from the publishable key, falling back to
`Config.RegistrationAppID`.

Every `OAuth2Client` carries an `AppID` with a NOT NULL foreign key to
`authsome_apps`, RFC 7591 has no slot for it, and a body field would let any
caller name any tenant they like, so the app has to come off the transport
instead. You get that for free: `middleware.PublishableKeyMiddleware` already
resolves `X-Publishable-Key` (or `?publishable_key=`) into an app and stashes
the ID on the context, and it already runs on plugin routes, because
`api.RegisterRoutes` calls `router.Use` on the same router the extension later
hands to plugins.

That middleware never aborts. On a missing or unresolvable key it leaves the
context empty and lets the handler own the rejection, so when neither the key
nor the config fallback yields an app, registration returns 403.

The fallback is what makes the single-tenant MCP case work at all. A stock
client fetches well-known documents from the origin root and has nowhere to get
a publishable key from, so if you want out-of-the-box registration you set
`RegistrationAppID` and advertise a bare `registration_endpoint`. Run
multi-tenant and you leave it unset, handing out key-bearing URLs instead.

### What a dynamic client may hold

Grant types are clamped to `authorization_code` and `refresh_token`, and scopes
are clamped to an allowlist.

Grant type is the boundary that actually bites, because `issueClientToken`
creates a real `account.Session` with a nil user and returns its token, which
means a client holding `client_credentials` walks away with a working service
token that no user ever consented to. On an open endpoint, anyone who can reach
`/register` gets one. Requests for `client_credentials` or the device grant are
rejected with `invalid_client_metadata` rather than quietly dropped, since a
client asking for them is asking for a capability, not expressing a preference.

Scopes are intersected against `Config.DynamicRegistrationScopes`, which
defaults to `openid profile email offline_access`. Anything outside is dropped
silently and the response echoes back what was actually granted, which RFC 7591
section 2 explicitly permits. Dropping beats erroring here because MCP clients
tend to request a broad set optimistically, and a hard failure would turn a
registration that could have worked into a dead one.

#### What scope actually does

Worth stating plainly, because it changes how much the allowlist buys you.
`issueTokens` mints an `account.Session` and returns its token, and on the
opaque token path the scopes are never persisted at all: they're echoed in the
response and dropped on the floor. Only the JWT path stamps them into claims.
An access token from this provider is therefore a full user session token, and
`scope` gates nothing today.

So the allowlist isn't closing a live hole. It's making sure the population of
dynamically registered clients is already correct on the day scope becomes
enforced, instead of grandfathering in a pile of clients holding whatever they
happened to ask for.

### Redirect URI validation

Runtime matching is already exact-string in `resolveRedirectURI`, with no prefix
or subpath matching, so this section is only about what may be registered in the
first place.

Accept `https://` with a host, no fragment and no userinfo. Accept `http://`
where the host is exactly `127.0.0.1` or `[::1]` on any port, per RFC 8252
section 7.3, and ignore the port at match time because clients bind an ephemeral
one. Accept a private-use scheme containing a dot, for example
`com.example.app:/callback`.

Reject everything else, including `http://localhost` by name, wildcards, and any
scheme without a dot. Resolution of `localhost` depends on the client host's DNS
and hosts file, so it isn't the guaranteed-local target the literal IP is, and
RFC 8252 recommends the IP for exactly that reason.

Rejections return `invalid_redirect_uri`.

### Rate limiting

`middleware.RateLimit` on `POST /register`, keyed by `ClientIP`, ten per hour by
default and configurable through `Config.RegistrationRateLimit`. The RFC 7592
routes get a separate, looser limit keyed by `client_id`, since a well-behaved
client polls its own registration and shouldn't have to spend the registration
budget doing it.

One thing to know going in: the middleware fails open when the limiter errors.
A limiter outage removes the cap rather than blocking registration, which is the
right tradeoff for the rest of the API and worth being aware of on an endpoint
that's deliberately unauthenticated.

## Data model

RFC 7591 defines around fifteen metadata fields. Five of them change behaviour
and the rest are informational, only ever needing to round-trip on a 7592 read,
so those go in a blob instead of costing fifteen columns across four backends.

New columns on `authsome_oauth2_clients`:

| Column | Type | Purpose |
|---|---|---|
| `token_endpoint_auth_method` | TEXT | `none`, `client_secret_basic` or `client_secret_post` |
| `registration_token_hash` | TEXT | bcrypt of the RFC 7592 registration access token. Empty for admin-created clients |
| `dynamically_registered` | BOOLEAN | Drives policy, and lets the dashboard and audit log tell the two populations apart |
| `client_secret_expires_at` | TIMESTAMPTZ NULL | RFC 7591 requires it in the response. NULL means never expires, serialised as `0` |
| `metadata` | JSONB | `client_uri`, `logo_uri`, `contacts`, `tos_uri`, `policy_uri`, `software_id`, `software_version` |

`client_id_issued_at` maps onto the existing `created_at`, and `client_name`
maps onto the existing `Name`.

`token_endpoint_auth_method` becomes the source of truth for whether a client is
public, with the existing `Public` bool derived from it (`none` means public).
Two independent flags that can disagree is a bug waiting to happen. The admin
create path writes both.

Types in that table are the postgres ones. SQLite has neither, so it takes
`TEXT` for `metadata` (which is what the existing `redirect_uris`, `scopes` and
`grant_types` columns already do there) and `TIMESTAMP` for
`client_secret_expires_at`.

One migration at version `20260824000030` for postgres and sqlite, every column
defaulted so existing rows stay valid. Mongo and memory need no schema step.

The version is deliberately not `20260824000001`: the RFC 8707 resource
indicators design claims that one in the same `authsome-oauth2` group, and two
migrations sharing a version in one group fails at startup. The DPoP design
adds a `dpop_mode` column to this same table, so expect to merge rather than
choose.

## Endpoints

### POST /v1/oauth/register

Returns 404 unless `Config.DynamicRegistration` is true. In order:

1. Rate limit by client IP.
2. Resolve the app from `AppIDFrom(ctx)`, then `Config.RegistrationAppID`.
   Neither resolves, 403 `access_denied`.
3. Validate `redirect_uris`: required, non-empty, every entry passing the rules
   above.
4. Clamp grant types. A request for anything outside `authorization_code` and
   `refresh_token` is `invalid_client_metadata`, with a message naming the
   offending grant so the client author can see what happened.
5. Clamp scopes against the allowlist, dropping silently.
6. Issue. The `client_id` is 16 random bytes hex, matching the admin path, and
   a `client_secret` is issued only when `token_endpoint_auth_method` is not
   `none`. The `registration_access_token` is 32 random bytes hex,
   bcrypt-hashed before storage and returned exactly once.
7. 201 with the full registered metadata, `registration_access_token` and
   `registration_client_uri`.

Errors on this route and the 7592 routes use the RFC 7591 section 3.2.2 body
shape, `{"error": "...", "error_description": "..."}`, and not the forge
envelope. That's a deliberate divergence from the rest of the API, made because
MCP clients parse those two fields and won't find them anywhere else.

### GET, PUT, DELETE /v1/oauth/register/{client_id}

RFC 7592. Auth is the registration access token presented as a bearer token:
look the client up by the `client_id` in the path, reject if
`dynamically_registered` is false, then run `bcrypt.CompareHashAndPassword`
against the stored hash. A miss is 401 with a `WWW-Authenticate: Bearer` header.

An admin-created client isn't manageable through 7592 at all, even if someone
guesses its `client_id`, because it has no registration token hash to compare
against in the first place.

PUT runs the same validation pipeline as registration, so an update can't widen
grants or scopes past policy. Per RFC 7592 section 2.2, a `client_id` or
`client_secret` present in the body has to match what's stored, otherwise the
request is `invalid_client_metadata`.

The registration access token isn't rotated on read or update. The RFC permits
rotation, and rotation strands any client that doesn't persist the new value,
which in practice is most of them.

DELETE removes the client and returns 204.

Unlike `POST /register`, these three routes stay live when
`Config.DynamicRegistration` is false. Turning the feature off closes the door
to new registrations, and it shouldn't strand the clients that registered while
it was open: an operator who wants them gone needs DELETE to keep working, and
a client that can no longer read its own registration has no way to find out it
was revoked.

## Discovery

### The root router problem

Plugins only ever receive the grouped router. `extension.go` passes
`groupedRouter` to every `RouteProvider`, and the `api` package keeps a separate
`rootRouter` precisely because well-known routes can't live under a mount
prefix.

Which means that today, in extension mode, the OIDC discovery document
registered in `RegisterRoutes` is really being served at
`{basePath}/.well-known/openid-configuration`. Nobody noticed because OIDC
clients are usually handed the issuer URL directly. RFC 8414 and RFC 9728 won't
tolerate it, since a client fetches `https://host/.well-known/...` with no
configuration at all and a 404 ends discovery there.

Add a `RootRouteProvider` interface next to the existing `RouteProvider`:

```go
// RootRouteProvider is implemented by plugins that must serve routes at the
// origin root rather than under the extension's mount prefix. Well-known
// discovery documents are the only legitimate use: RFC 8414 and RFC 9728
// define their locations relative to the origin, so a prefixed copy is
// invisible to a client that only knows the host.
type RootRouteProvider interface {
	RegisterRootRoutes(router forge.Router) error
}
```

`extension.go` walks `RootRouteProviders()` with the ungrouped router before the
grouped pass. In standalone mode the two routers are the same instance, so the
extension guards the duplicate registration the way `api.go` already does for
its own well-known mirror. The oauth2provider moves all four well-known routes
into `RegisterRootRoutes` and keeps the prefixed OIDC path registered as a
mirror, so anyone currently fetching it doesn't break.

### The documents

A single `buildAuthServerMetadata()` produces the shared field set, so the 8414
and OIDC documents can't drift apart.

| Path | RFC | Notes |
|---|---|---|
| `/.well-known/oauth-authorization-server` | 8414 | New. The document MCP clients actually fetch |
| `/.well-known/openid-configuration` | OIDC Discovery | Existing, gains `registration_endpoint` |
| `/.well-known/oauth-protected-resource` | 9728 | New. `resource`, `authorization_servers`, `scopes_supported`, `bearer_methods_supported: ["header"]` |
| `/.well-known/oauth-protected-resource/{path}` | 9728 section 3.1 | Per-resource, driven by `Config.ProtectedResources` |

`registration_endpoint` appears only when `Config.DynamicRegistration` is true.
Advertising an endpoint that 404s sends clients down a dead path and produces a
worse error than not advertising it.

The 9728 document describes AuthSome as its own protected resource, with
`resource` and `authorization_servers` both set to the issuer.
`Config.ProtectedResources` lets you declare additional resource identifiers
served under the path-suffixed form, which covers a separately deployed MCP
server using AuthSome as its authorization server.

### The 401 hint

RFC 9728 section 5.1 has the resource server return
`WWW-Authenticate: Bearer resource_metadata="<url>"` on a 401. That's how a
client bootstraps discovery from a failed call instead of from a URL somebody
handed it, and it's the path the MCP spec uses.

Add `middleware.ResourceMetadataChallenge(url string)`, wired in `api.go`
alongside the other `router.Use` calls and driven by a new engine option so it
stays inert when unset. Nothing emits `WWW-Authenticate` today, so this is
additive and not a change to an existing header.

## Store work

The `Store` interface gains one method:

```go
// UpdateClient persists changes to an existing client, matched on ID.
UpdateClient(ctx context.Context, c *OAuth2Client) error
```

Everything else rides on `CreateClient`, `GetClient` and `DeleteClient`, widened
by the new fields.

Postgres and sqlite get the new fields on the shared `oauth2ClientModel` in
`store_models.go`, with `UpdateClient` implemented via `NewUpdate`, and one
migration between them. Mongo gets the new fields on its bson client struct and
implements `UpdateClient` via `ReplaceOne`, and its array-shaped fields get
explicit empty-slice initialisation so they marshal as `[]` and never `null`,
which is the bug commit 9116564 just fixed for session roles. Memory overwrites
the map entry under the write lock.

## Testing

Table-driven, following `authcode_test.go`, against `MemoryStore`. Tests come
first and fail first, per the repo's TDD discipline.

The policy tests matter most, because the policy is the feature. A registration
requesting `client_credentials` is rejected. One requesting `admin:all` has it
dropped rather than granted. Every rejected redirect URI class gets its own
case: wildcard, fragment, userinfo, non-loopback `http://`, `http://localhost`
by name, and a custom scheme with no dot. Every accepted class gets one too.

For tenancy, a publishable key selects the app, an absent key falls back to
config, neither present is a 403, and a key for app A never produces a client in
app B.

The RFC 7592 tests cover a wrong token returning 401, a valid token belonging to
a different client returning 401, an admin-created client being unreachable, PUT
failing to widen grants or scopes, and DELETE actually removing the client.

Discovery tests check that `registration_endpoint` is absent when the feature is
off and present when it's on, that the 8414 and OIDC documents agree field for
field, and that the 9728 document round-trips a configured extra resource.

Finally, `UpdateClient` joins the shared store conformance suite, so all four
backends are held to the same behaviour.

## Files

New in `plugins/oauth2provider/`: `register.go` for the 7591 and 7592 handlers,
`register_validate.go` for the redirect URI, grant and scope policy,
`metadata.go` for the four documents, plus tests.

Edited: `plugin.go`, `models.go`, `store.go`, `store_models.go`,
`store_memory.go`, `store_postgres.go`, `store_sqlite.go`, `store_mongo.go`,
`migrations.go`.

Outside the plugin: `plugin/plugin.go` for the new interface,
`extension/extension.go` for the root route pass, `api/api.go` for wiring the
challenge middleware, and a new `middleware/resource_metadata.go`.

## Phasing

Four PRs, each standing alone.

1. `RootRouteProvider`, plus moving the existing well-known route onto it. No
   new endpoints, and it fixes the prefixed OIDC path as a side effect.
2. Client model, migration, and `UpdateClient` across four stores.
3. RFC 7591 and RFC 7592 endpoints with the full policy pipeline.
4. RFC 8414 and RFC 9728 documents, `registration_endpoint` advertisement, and
   the 401 challenge middleware.

Phase 1 has to land before phase 4, because the 8414 and 9728 documents need
the root router to be reachable at all. Phase 2 has to land before phase 3,
since the registration handlers write `registration_token_hash` and
`dynamically_registered`. Phase 1 is independent of phases 2 and 3, so it can
go first or in parallel.
