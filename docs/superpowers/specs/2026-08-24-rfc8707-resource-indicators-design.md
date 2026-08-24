# RFC 8707 Resource Indicators

Date: 2026-08-24
Status: approved, ready for implementation planning

## What this fixes

The oauth2provider plugin issues tokens carrying scopes and no audience, which
means a token minted for service A is accepted by service B whenever both
services trust the same issuer, and service B has no way to tell that the user
only ever consented to service A. That's the confused deputy setup. RFC 8707
closes it by letting the client name the resource it wants a token for, and by
binding that name into the token as `aud`.

You get four things when this lands. Clients can pass `resource` at
`/authorize`, `/token`, and `/device/authorize`. Tokens carry `aud`, both as a
JWT claim and on the session record for opaque tokens. Authsome's own auth
middleware can refuse a token audienced at somebody else. Asking for a resource
the client was never granted returns `invalid_target`.

## Decisions already taken

These were settled during brainstorming. Don't reopen them while implementing.

Audience for opaque tokens lives on `session.Session`, not in a plugin-local
side table, because an opaque access token in this codebase is a session token
and the session record is already where introspection and middleware look.

Enforcement happens in the core auth middleware and is driven by config, so a
relying party does not have to remember to wrap anything. The expected audience
resolves per app, since two apps in one deployment are two different resources.

A client whose resource allowlist is empty gets `invalid_target` the moment it
asks for a resource. Nothing existing breaks, because a client that sends no
`resource` still gets a token with no audience, exactly as it does today.

Repeated `resource` parameters are honoured and `aud` is emitted as an array.
The RFC permits this while discouraging it, so we permit it too.

The device authorization grant is covered alongside the other two flows.

## Data model

Four new fields, all `[]string`, all defaulting to empty.

| Type | Field | Storage |
|---|---|---|
| `session.Session` | `Audience` | `authsome_sessions.audience` |
| `OAuth2Client` | `Resources` | `authsome_oauth2_clients.resources` |
| `AuthorizationCode` | `Resources` | `authsome_oauth2_auth_codes.resources` |
| `DeviceCode` | `Resources` | `authsome_oauth2_device_codes.resources` |

Postgres gets `JSONB NOT NULL DEFAULT '[]'` and SQLite gets `TEXT NOT NULL
DEFAULT '[]'`. Use migration version `20260824000001` in each group, which sits
after the current heads: `20260620000002` in core, `20260615000001` in the
plugin.

Memory needs nothing. `store/memory` holds `*session.Session` directly instead
of converting through a model, so you get the new field for free.

Mongo needs care in two places, and the second one has already bitten this
repo once. First, adding a field to `sessionModel`
leaves an existing deployment's generated `$jsonSchema` unaware of it, so the
migration has to call `RefreshValidator(ctx, (*sessionModel)(nil))`; copy the
shape of `add_session_principal_identity` at `store/mongo/migrations.go:878`.
Second, grove writes every mapped field whatever the bson tag says, so a nil
slice arrives at mongo as `audience: null`, the validator rejects it because
the schema declares an array, and the write fails for any session that happens
to carry no audience, which will be almost all of them. `toSessionModel` must
assign an empty slice when the field is empty. That's the same bug 9116564
fixed for `roles`, and you should write the guard in now instead of
rediscovering it against a live mongo later.

## Reading the resource parameter

The struct binder can't carry this parameter at all, and it's worth writing
down why, so nobody tries the obvious thing and spends an afternoon wondering
why every authorize request started returning 400.

The chain runs from forge `internal/router/handler.go:206` into go-utils
`http.Ctx.BindRequest`. `bindQueryParam` reads a single value through
`c.Query(name)`, which is `url.Values.Get`, so a repeated parameter loses
everything after the first value before the field type is even considered. Then
`setFieldValue` switches on String, Int, Uint, Float and Bool, with no case for
slices, so a `[]string` field falls through to the default branch and adds
`unsupported field type: slice` to the validation errors. Tagging a slice field
`query:"resource"` doesn't degrade quietly. It breaks `/authorize` outright.

So `resource` gets read off the raw request by one helper:

```go
// resourceParams extracts the repeatable RFC 8707 resource parameter.
//
// This cannot go through the struct binder. go-utils' bindQueryParam reads a
// single value via url.Values.Get, and setFieldValue has no reflect.Slice
// case, so a []string field tagged query:"resource" does not merely lose the
// second value, it fails the whole request with "unsupported field type".
// Reading the raw request is the only way to honour a parameter the RFC
// defines as repeatable.
func resourceParams(r *http.Request) []string
```

It returns `r.URL.Query()["resource"]`, and for POST it also merges
`r.PostForm["resource"]` after calling `ParseForm`. All three handlers use it.

`TokenRequest` can additionally keep a plain JSON field, because `BindJSON`
runs through `encoding/json` and handles slices without complaint:

```go
Resource []string `json:"resource,omitempty"`
```

A form-encoded request populates `resourceParams` and leaves the JSON field
empty, while a JSON request does the reverse, so in practice only one of them
ever carries values. If both somehow do, `resourceParams` wins and the two sets
are never concatenated.

For OpenAPI, declare the parameter by hand with `forge.WithParameter("resource",
"query", ...)`. It probably cannot express `style: form, explode: true` for an
array, so note that limitation for whoever picks up the SDK regeneration pass.

## Validation

Add `resolveResources(client, requested []string) ([]string, error)`, shaped
like `resolveScopes` in `plugins/oauth2provider/plugin.go:560`. It returns
`invalid_target` when any of these hold:

- a value is not an absolute URI (RFC 8707 section 2, via `url.Parse` plus an
  `IsAbs` check)
- a value carries a fragment component
- a value is not present in `client.Resources`
- `client.Resources` is empty and anything at all was requested

Duplicates collapse and order is preserved. An empty request returns an empty
slice, meaning an unrestricted token.

Errors go through `newOAuth2Error(http.StatusBadRequest, "invalid_target", ...)`
so the body carries the registered error code instead of forge's generic
bad-request shape.

Somebody has to be able to fill the allowlist, or the deny-by-default rule
makes the feature unreachable. `CreateClientRequest` gains `Resources
[]string`, `CreateClientResponse` echoes it back, and `handleCreateClient`
validates each entry as an absolute URI without a fragment before storing it,
using the same check `resolveResources` runs. The dashboard client editor in
`plugins/oauth2provider/dashboard.go` needs the field too.

One rule is easy to miss. The token endpoint may also send `resource` to narrow
what was already authorized, so in `handleAuthorizationCodeGrant` the requested
set has to be a subset of the set stored on the code, and a token request that
omits `resource` inherits the code's full set. The device code grant follows
the same rule against its stored device code.

## Issuance

`tokenformat.TokenClaims` gains `Audience []string`. In `tokenformat/jwt.go`,
`GenerateAccessToken` sets `RegisteredClaims.Audience` from the per-token value
when that value is non-empty and falls back to `JWTConfig.Audience` otherwise,
so anyone relying on the current static audience sees no change at all.
`ValidateAccessToken` drops `aud` on the way back out today. It starts
returning it.

`issueTokens` and `issueClientToken` both take the resolved resources and write
them to `sess.Audience` and to the JWT claim, so opaque and JWT tokens agree
about what a token is for.

Rotation has to carry audience forward. If `RotateSession` drops it, then a
refresh quietly converts a bound token into an unrestricted one, which is a
bigger hole than the one this whole change exists to close, and it'll be
invisible until somebody audits a rotated token by hand. Give it its own test.

## Enforcement

`middleware.SessionBindingConfig` gains:

```go
// ExpectedAudienceResolver returns the resource identifiers this deployment
// answers to for the current request's app. Nil, or an empty result, disables
// the check.
ExpectedAudienceResolver func(context.Context) []string
```

It sits beside `CookieNameResolver`, which already runs per request on the same
path, so both the shape and the hot-path cost are familiar.

`trySessionAuth` and `tryJWTAuth` apply the same three rules. No resolver, or
an empty result, skips the check entirely. An empty token audience passes,
since an unaudienced token is unrestricted. A non-empty token audience that is
disjoint from the expected set fails authentication, logs at warn, and returns
false, so the request carries on unauthenticated like any other auth failure.

You stay backwards compatible in both directions. Configure nothing and
behaviour does not change. Issue no audienced tokens and behaviour does not
change.

The per-app resource identifier lives in app settings and is read through the
`settings` manager.

## Metadata and introspection

`DiscoveryResponse` gains `resource_indicators_supported: true`.

That name is not registered anywhere. RFC 8707 registers the `resource`
parameter and the `invalid_target` error and defines no discovery metadata, and
the RFC 8414 IANA registry has no entry for it either, but it's the convention
that came out of the MCP ecosystem and it's what clients actually look for. Put
that in a code comment so nobody later reads the field and assumes a standard
blesses it.

`IntrospectResponse` in `api/requests.go:666` gains `aud []string`, populated
from `claims.Audience` on the JWT path and `sess.Audience` on the opaque path
in `api/introspect_handler.go`.

## Testing

Table-driven throughout, following the repo's Go testing conventions.

`resolveResources` takes the bulk of the unit tests: non-absolute URIs,
fragments, unregistered resources, the empty allowlist, duplicates, and the
empty request.

`resourceParams` needs tests for a repeated query parameter, a repeated form
value, and the JSON body path. A single-value regression there is invisible
until somebody sends two.

The plugin tests cover a full authorize-then-token round trip asserting `aud`
lands on the token, subset narrowing at the token endpoint, over-broad
narrowing rejected. Repeat those three against the device flow.

Middleware tests cover all three enforcement rules on both the opaque and the
JWT path.

Then add audience round-tripping to the `store/storetest` conformance suite.
That's the test that catches the mongo null-array problem the moment anybody
points `AUTHSOME_MONGO_URI` at it, and skipping that suite is how 9116564
reached main in the first place.

## Out of scope

SDK regeneration across Dart, Go and TypeScript is a separate pass. The working
tree already carries uncommitted changes under `sdk/` and `sdkgen/`, so folding
a regeneration in here would tangle two unrelated diffs.

Form-urlencoded binding is broken for every endpoint that relies on `form:`
tags, this one included. `bindField` dispatches only `path`, `query` and
`header`, and `Ctx.Bind` for `application/x-www-form-urlencoded` calls
`ParseForm` and then returns without writing anything into the struct, so the
token endpoint only works with a JSON body today, which happens to be what its
test sends. That's pre-existing, it's tracked separately, and `resourceParams`
reads `PostForm` directly, so this change works whichever way that one lands.

RFC 8693 token exchange and RFC 7591 dynamic client registration do not exist in
the tree yet. This composes with both when they arrive. Neither is needed now.
