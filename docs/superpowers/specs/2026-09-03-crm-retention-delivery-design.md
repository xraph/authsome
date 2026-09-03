# retention: CRM delivery for auth events

Status: approved design, not yet implemented

Date: 2026-09-03

## The problem

Your CS team runs renewals out of the CRM. They can see the deal, the emails
and the support tickets. What they cannot see is whether the customer logged in
this month, which is the single best predictor of whether that renewal happens.
Authsome knows. Nobody asks it.

Closing that gap sounds like a webhook, and for a while it is. Then the CRM has
a bad afternoon, and you find out what you've actually built.

The trap is that auth hooks run in the request path. `AfterSignIn` fires while
the user waits for their token. Call Salesforce inline and you've handed your
login latency to a third party, so their outage becomes your outage. So the
delivery has to come off the request path, which means a queue, which means all
the boring things queues need: idempotency, backoff, a dead-letter state you can
actually look at, and a lease so a crashed pod doesn't silently strand work
nobody ever comes back for.

## Scope

The full ask was a retention engine: sync to the CRM, score churn risk, and act
on it. That's three subsystems with a strict dependency chain, so it ships as
three slices. Only the first is specified here.

| Slice | What lands | Depends on |
|---|---|---|
| A (this doc) | Provider contract, two shipped providers, outbox, worker, hook wiring, consent gate | nothing |
| B | `last_seen_at`, `login_count_30d`, `lifecycle_state`, and a sweeper for dormancy transitions | A, for somewhere to write |
| C | Rule engine and action dispatch, including win-back mail through the notification plugin | A and B |

Slice A on its own gets login activity in front of the CS team, which is the
point of the exercise. B and C get their own brainstorm, spec and plan.

## The provider contract

"Any CRM" means two different things and you want both. Some CRMs deserve real
Go: they have quirks, rate limits and error semantics worth encoding. Most just
need an HTTP call with the fields mapped. So the interface is the contract, and
two implementations ship against it.

```go
// Provider is a CRM the retention plugin can mirror auth activity into.
type Provider interface {
    Name() string
    Capabilities() Capability
    UpsertContact(ctx context.Context, c *Contact) (RemoteRef, error)
    LogActivity(ctx context.Context, ref RemoteRef, a *Activity) error
}
```

`generic` is config-driven REST. You give it an endpoint, an auth mode and a
field mapping, and a CRM nobody has written a line of Go for works anyway,
which is what "any CRM" has to mean if it's going to mean anything. The vendor
provider is a real implementation against a real API, and it's there to prove
the interface survives contact with one.

Two details carry more weight than they look like they do.

`Capabilities()` is a bitmask (`CapContacts`, `CapActivities`). The worker
checks it before it enqueues, so a CRM with no activity concept never gets a
call it would have to fake success on. Without it, a provider that can't do
something returns `nil` to satisfy the signature, and from then on you can't
tell "delivered" from "quietly dropped". That's how "works with any X"
interfaces start lying to you.

`ProviderError` puts retry classification in the contract, so each provider
doesn't invent its own idea of what a bad response means:

```go
// ProviderError classifies a failure so the worker can decide retry vs
// dead-letter without knowing anything about the CRM's HTTP semantics.
type ProviderError struct {
    Err        error
    Retryable  bool
    RetryAfter time.Duration // honoured when non-zero, e.g. from 429
}
```

`RemoteRef` is `{Provider, ObjectType, ID}` and not a bare string, because
Salesforce-shaped CRMs need the object type before you can build a URL back to
the record.

## Data model

Two tables, prefixed `authsome_retention_`, migration group
`authsome-retention` depending on `authsome`. Four backends each, matching
every other plugin store in the tree.

`authsome_retention_contact_ref` is the dedup spine. Unique on
`(app_id, env_id, user_id, provider)`, holding `remote_object_type`,
`remote_id` and `synced_at`. The worker reads it before every upsert. A hit
means update, a miss means create. It's the only thing standing between a
thousand logins and a thousand duplicate contacts, so you're paying for this
table under every possible design, which is worth remembering when the outbox
next to it starts looking expensive.

`authsome_retention_outbox` is the pending work: `id`, `app_id` referencing
`authsome_apps(id)`, `env_id`, `user_id`, `provider`, `kind`
(`contact_upsert` or `activity_log`), `payload` JSON, `idempotency_key`,
`state`, `attempts`, `next_attempt_at`, `in_flight_until`, `last_error` and
`created_at`. Index on `(state, next_attempt_at)` for the claim query.

States are `pending`, `in_flight`, `done`, `dead` and `suppressed`. That last
one is how a consent refusal records itself, because "we chose not to send
this" and "we tried and failed to send this" need to be different rows when
somebody audits you.

## The worker

One goroutine, started in `OnInit` and stopped in `OnShutdown`, built the way
`plugins/sharedsignals/refresh.go` does it. Note `startWith(tick <-chan
time.Time)` there: tests push ticks by hand and never sleep, and you want that
here too.

Claiming has to be safe with several authsome instances running. Postgres uses
`UPDATE ... SET state='in_flight', in_flight_until=now()+lease WHERE id IN
(SELECT id FROM ... WHERE state='pending' AND next_attempt_at <= now() ORDER
BY next_attempt_at LIMIT n FOR UPDATE SKIP LOCKED) RETURNING *`. Mongo uses
`findAndModify`. SQLite and memory serialise on a mutex, which is honest
because both are single-process anyway.

`SKIP LOCKED` is doing real work in that query. Without it, two instances
claiming at once block each other and take turns where they should be taking
disjoint batches, so your throughput collapses back to single-instance under
exactly the load that made you run two of them in the first place.

The lease matters more. A pod killed between claim and completion leaves rows
sitting in `in_flight` forever, and because the claim filters on
`state='pending'`, those rows never come back. One user quietly desyncs per
crash, there's nothing in the logs pointing at it, and you won't hear about it
until months later when somebody asks why a customer's last login says March.
So the claim also takes rows whose `in_flight_until` has passed.

Ordering, without building a dependency graph: an `activity_log` job needs a
`contact_ref` row, and it's the worker that fixes this up. If the ref is
missing, the worker upserts the contact first and then logs the activity, in
the same job. It has already reloaded the user by that point, so the upsert
costs it nothing extra.

That's deliberately not a lookup in the hook. Checking for a ref at sign-in
would put a read on the login path, and the whole point of the outbox is that
a login writes one row and gets out. It also makes the sign-in path
self-healing: a contact deleted upstream, or a provider enabled after a user
already existed, both recover on the next login with no backfill job.

Backoff is exponential with jitter and a cap. A non-zero `RetryAfter` always
wins over the computed delay. After `max_attempts` the row goes `dead`, stays
queryable and bumps a counter.

The worker reloads the user at delivery and doesn't trust the payload it
enqueued. Anything you evaluate on the near side of a queue is a snapshot of a
fact that may have moved by the time it matters, and that goes for the email
address just as much as the consent grant.

## Hook wiring

```
AfterSignUp     -> enqueue contact_upsert + activity("signed_up")
AfterSignIn     -> enqueue activity("logged_in")
AfterSignOut    -> enqueue activity("logged_out")   // off by default
AfterUserUpdate -> enqueue contact_upsert
```

Every hook does one local insert and returns `nil`. A retention failure must
never fail a login, so the hooks log and swallow. The only thing that can go
wrong is a local write, and if your own database is down you've got larger
problems than CRM sync.

## Consent

This plugin ships PII to a third party, and there's already a consent plugin in
the tree. The check goes in the worker, immediately before the provider call,
for two reasons that happen to agree. It stays off the login path, and it reads
consent at the moment the data actually leaves, so a user who revokes between
login and delivery gets that revocation honoured, which an enqueue-time check
would quietly miss.

Retention reaches consent through `engine.Plugin("consent")` and a type
assertion to a narrow local interface, the way `anomaly`, `geofence`,
`impossibletravel` and `vpndetect` all reach geoip. Consent stays optional and
the plugin runs fine without it.

One additive change is needed, because the consent plugin currently exposes no
way to read a grant. `GetConsent` lives on the store, `SetConsentStore` has no
getter, and the only public surface is HTTP handlers:

```go
// HasConsent reports whether the user currently has an active grant for
// purpose. Missing record and revoked grant both report false.
func (p *Plugin) HasConsent(ctx context.Context, userID id.UserID,
    appID id.AppID, purpose string) (bool, error)
```

`require_consent` defaults to false. Fail-closed sounds more responsible, and
if purposes were a fixed enum it would be the right default. They're free text,
so fail-closed means every fresh install watches the plugin do nothing and
files a bug. Off by default, one flag to turn on, documented next to the flag.
When it's on, a missing record means no send and the row records `suppressed`.

## Config

Through `SettingsProvider`, shaped like `plugins/social`: provider list per
app, credentials by secret ref, an enabled flag per event, `max_attempts`,
`batch_size`, `tick_interval`, `lease_duration`, `require_consent` and
`consent_purpose`. `DataExportContributor` so a user's CRM refs show up in
their data export.

## Testing

TDD throughout, and the interesting cases all live in the worker.

The store conformance suite runs one table of cases against memory, sqlite,
postgres and mongo, including lease expiry and reclaim. Worker tests inject
ticks and a fake provider returning scripted `ProviderError`s, so backoff,
`RetryAfter` and dead-lettering all get asserted without a single sleep. Hook
tests check that enqueue is the only side effect, and that a store error still
returns `nil`.

Plugin fixtures in this repo wire no `Engine`, so anything that needs real
session behaviour goes in `api/`, not here.

## Out of scope

Erasure on post-sync revocation. If consent is withdrawn after a contact has
already landed in the CRM, honouring that means issuing a delete or a
suppression against a remote system, and it needs its own `Capability`, its own
job kind, and its own thinking about what a CRM that quietly refuses to delete
does to your compliance story. It's a real gap and it gets its own slice.

Also out: lifecycle state and scoring (slice B), rules and actions (slice C),
and a dashboard page. The relay stays available as an extra fan-out but it
can't be the transport, because `Send` returns no response payload and there's
nowhere to put the `RemoteRef`.

## Decisions taken after review

HubSpot is the reference vendor provider. Free developer accounts, a simple
REST surface, and contacts plus engagements that map onto the two interface
methods without much translation. Salesforce would be the more impressive
demo and considerably more work, so it stays a candidate for a later
provider rather than the one that proves the interface.

Treat the exact endpoint paths, the auth header shape and the search-by-email
call as things to confirm against HubSpot's current docs while implementing,
not as settled by this document. Vendor APIs move and this spec will not.

## Open questions

The body of `classifyHTTPError` in `provider_generic.go`. The signature and
everything around it gets scaffolded, and Rex writes the classification. It
decides whether a bad CRM response means try again soon, wait exactly this
long, or give up for good. Wrong one way and you hammer a rate-limited API
until it bans you. Wrong the other way and a transient 503 permanently drops a
customer's sync. Worth weighing: 429 with and without `Retry-After`, 5xx
against 4xx, whether a 401 is a dead credential or a token about to refresh,
whether a 404 on update means the contact was deleted upstream and the ref
should be dropped so the next attempt recreates it, and the transport case
where there's no response at all.
