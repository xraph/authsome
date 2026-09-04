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
    DropRef    bool          // the CRM no longer holds this record
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

Rows do not live forever. `PurgeTerminal` deletes `done` rows after 30 days
and `dead` and `suppressed` rows after 180, both configurable, both swept on
the delivery worker's own ticker once an hour. Non-terminal rows are never
eligible at any age. A `pending` row older than the window is a stuck job,
not litter, and deleting it destroys work nobody knows is missing.

Pruning costs you something, and it is worth naming rather than discovering.
The unique index on `idempotency_key` lives on the table, so deleting a row
releases its key. **The retention window is therefore also the replay
window.** Inside it, a hook that fires twice for one event still enqueues
once. Outside it, that same replay would enqueue again.

Thirty days is chosen against that, not against disk. A duplicate hook
dispatch arrives within seconds of the original, from the same request or a
retry of it. The outermost thing that can re-present a job to this table is
its own retry budget, and twelve attempts against a thirty-minute cap tops
out around 1.7 hours. Thirty days is four hundred times that. For a replay to
land outside the window you would need the identical request, carrying the
identical session id, dispatched a month later, which is not a thing that
happens to a login. And the sign-up and sign-in keys derive from a session
id, which is never reissued.

`dead` and `suppressed` get six times longer because they are the audit
trail rather than the steady state. `suppressed` is the row that proves the
consent gate refused a send, `dead` is the row that says what an outage cost
you, and both get read on a review cycle measured in quarters. They are also
rare, so keeping them half a year costs almost nothing next to the login
traffic in `done`.

The comparison is on `created_at`, because there is no `completed_at` column
and adding one to buy a few hours of precision against a window measured in
months is not worth the migration.

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

Sign-out activity. It was in this design until implementation proved it
cannot be built. `OnAfterSignOut(ctx, sessionID)` carries a session id and
nothing else, and `service.go` deletes the session row before it emits, so by
the time the hook runs there is nothing left to resolve the id against. You
cannot get a user, an app or an environment out of it. Reading the app id off
the request context instead would work for HTTP sign-outs and silently miss
every other path, which is a worse answer than not shipping it. Unblocking it
means changing the hook to carry the user, or emitting before the delete, and
both are engine changes rather than plugin ones. The `retention.track_sign_out`
setting is gone with it: an operator toggle that provably does nothing is worse
than an absent feature.

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

## Retry classification

One rule generates the whole table: **a failure that affects every job retries,
a failure that affects only this job dies now.**

That asymmetry comes from this codebase rather than from taste. `dead` is
terminal and nothing re-enqueues it, so dead-lettering a whole-integration
problem destroys the entire backlog over something an operator is about to fix.
Retrying a bad payload, by contrast, wastes eight requests and delays a
diagnostic you wanted immediately. The costs are nowhere near symmetric, so the
policy is not either.

| Response | Verdict | Delay |
|---|---|---|
| no response (DNS, dial, timeout) | retry | exponential |
| 429 with `Retry-After` | retry | honour it, clamped to 1s..30m |
| 429 without | retry | exponential |
| 401, 403 | retry | flat 2 minutes |
| 500, 502, 503, 504, other 5xx | retry | exponential |
| 501 | terminal | |
| 404 on a record we hold a ref for | retry, and drop the ref | exponential |
| 408, 409 | retry | exponential |
| 400, 422, 413, other 4xx | terminal | |
| anything unrecognised | terminal | |

Two rows are worth explaining, because both look wrong at a glance.

**401 and 403 retry.** The instinct is that a dead credential should fail fast.
But a bad token fails every job, not this one, and a dead row never comes back.
Getting it wrong this way costs eight wasted requests per job. Getting it wrong
the other way permanently destroys the backlog because somebody fat-fingered a
token rotation. The flat two minutes buys roughly a fourteen-minute window at a
low request rate, instead of burning the whole budget inside the first minute.

**404 drops the ref.** The contact was deleted upstream, so retrying the same
update fails forever. Dropping the ref makes the next attempt find nothing and
recreate the contact, which turns a permanent failure into a self-healing one.
That is what `DropRef` on `ProviderError` is for, and it only fires when a ref
actually existed, so a transient 404 on a create cannot orphan anything.

A note on what the classifier can and cannot express. It never sees the attempt
count, so `RetryAfter` replaces the exponential curve rather than raising its
floor. A flat delay therefore trades a long tail for a low early rate, which is
why 401 and 403 take one and 429-without-a-header does not. If that trade shows
up a third time, add a `MinBackoff` field so a classification can lift the floor
without flattening the curve.

## Retry budget

`MaxAttempts` defaults to 12 now, not 8: this section used to leave that
number open, and it doesn't anymore.

The old default of 8 bought seven retries and about ten and a half minutes
before a retryable failure dead-lettered: 5, 10, 20, 40, 80, 160, 320 seconds,
doubling from `BaseBackoff`. That's shorter than plenty of ordinary vendor
incidents, and once a job dead-letters nothing re-enqueues it, so the backlog
for everyone who signed in during the outage is gone for good.

Twelve attempts gets you eleven retries and roughly 1.7 hours: the same curve
out to 1280 seconds, then two more waits pinned at the thirty-minute cap.
That covers a real CRM outage instead of just a blip. The price is four extra
requests per job that was always going to fail anyway, and a row that sits
around longer before it dead-letters. Both are cheap next to losing real
signup and login activity, for an error the classifier already decided was
worth retrying.

`BaseBackoff` and the thirty-minute cap are untouched. Only the attempt count
moved.

## Duplicate activities

Delivery is at-least-once and the two job kinds are not equally protected.
A repeated `contact_upsert` is absorbed by the contact ref, which turns it
into an update. Nothing plays that role for `activity_log`, so a `MarkDone`
that fails after the provider call already succeeded leaves the job
`in_flight`, the lease expires, it is redelivered, and the CS team sees the
same login twice.

Three shapes were on the table. HubSpot has no idempotency key you can send:
`hs_unique_creation_key` reads like one and is managed internally, not
settable on a create. A local "already delivered" marker written just before
`MarkDone` only narrows the window, because the marker write fails in
exactly the same way the `MarkDone` did. What is left is to carry a
deterministic id into the activity and look for it before creating a second
one, which is what ships.

The id is the outbox job's own `idempotency_key`, which is already stable
across attempts and unique across events. HubSpot carries it in
`hs_note_body`, on its own line, and finds it again with `CONTAINS_TOKEN` on
the notes search endpoint. Not a custom property, which would be the tidier
home: a custom property has to exist in the target portal first and HubSpot
answers a create naming an undefined one with a 400, the same constraint
that keeps the contact upsert to three built-in fields. `hs_note_body` is
built in and documented as searchable. `CONTAINS_TOKEN` matches tokens
rather than substrings, so every hit is checked against the exact marker
before it counts.

The search runs only on a redelivery. HubSpot rate limits search to five
requests per second per account, twenty times tighter than the ordinary
object endpoints, and a first delivery has nothing to collide with.

Knowing it is a redelivery is the part that needed a new signal, because
`attempts` is blind to precisely this case: nothing incremented it, since
nothing got to record a failure. The row's state before the claim is the
only trace, so `ClaimDue` now reports `Job.Reclaimed` when it took a row
through the expired-lease clause. It cannot be recorded any earlier either.
The failure it detects is a store outage, and any marker we would write to
warn ourselves goes to the store that just refused a write.

Two gaps stay open, and they stay open in writing rather than quietly.
HubSpot's docs say newly created objects take a few moments to appear in
search results, so a redelivery landing inside that lag can still create a
second note. The window this is built for is a failed `MarkDone`, which
comes back only after the lease expires, two minutes by default, and that is
clear of it. A redelivery after a failed attempt can come back in five
seconds and is not.

The generic provider cannot close this at all and does not pretend to. It
sends the id as `external_id`, mapped through `FieldMap` like everything
else, so a CRM that upserts on a field of its own gets what it needs. It
does not advertise `CapActivityDedupe`, because sending a field is not
knowing the far end honours it. The worker warns on every redelivery to a
provider without that bit, which is the same reasoning `Capabilities` was
introduced with: a provider that quietly returns `nil` for something it
cannot do leaves you unable to tell delivered from dropped.
