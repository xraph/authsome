# sharedsignals

Receives Shared Signals Framework events, so other identity providers can tell you when one of
your users has been compromised.

When Google detects a session hijack or Okta sees an account takeover, they can push a Security
Event Token to anyone subscribed. This plugin is the receiving end. It validates the SET
signature against the transmitter's JWKS, checks the audience and timestamps, stores the
signal, and feeds a risk score into `riskengine`.

The signals are worth having because they come from detection work somebody else already did, on
infrastructure you do not run.

## Use it when

- Your users sign in through Google, Okta or Entra and you want to hear about compromises those providers detect
- You are implementing CAEP for continuous access evaluation
- You want a high-quality external input to your risk scoring

## Skip it when

- No transmitter is willing to push to you. This plugin only receives, and someone has to be sending
- You have no `riskengine` registered. The signals get stored but nothing acts on them
- You cannot expose a public HTTPS endpoint for the push path

## Wiring

```go
authsome.WithPlugin(riskengine.New()),
authsome.WithPlugin(sharedsignals.New(sharedsignals.Config{
    Audience:   "https://auth.example.com",
    SignalTTL:  24 * time.Hour,
})),
```

`riskengine` discovers this plugin as a contributor automatically.

## Config

| Field | Type | Default | What it does |
|---|---|---|---|
| `Audience` | `string` | none | The `aud` inbound SETs must carry. Streams inherit it |
| `SignalTTL` | `time.Duration` | `24h` | How long a stored signal stays active |
| `MaxActionsPerHour` | `int` | `100` | Circuit breaker default for a new stream |
| `ClockSkew` | `time.Duration` | small | Tolerance for an `iat` slightly in the future |
| `MaxSETAge` | `time.Duration` | generous | How far in the past an `iat` may be |
| `MaxBodyBytes` | `int64` | bounded | Cap on the push request body |
| `MaxRiskScore` | `int` | `84` | Ceiling on the score handed to `riskengine` |
| `KeyRefreshInterval` | `time.Duration` | default | JWKS background refresh. Negative turns the ticker off |

`MaxRiskScore` defaults to 84 for a reason worth knowing. A single contributor's score becomes
the whole composite, and `riskengine` blocks at 85. At 100 a confirmed compromise would refuse
the sign-in outright, locking out a user who holds correct credentials and a correct second
factor. At 84 it challenges. Raise it to 100 if you would rather a confirmed compromise bar the
door: that number is a policy statement, not a measurement.

## Settings

| Key | Default |
|---|---|
| `sharedsignals.enabled` | `true` |
| `sharedsignals.signal_ttl_hours` | `24` |
| `sharedsignals.max_actions_per_hour` | `100` |
| `sharedsignals.risk_weight` | `2` |
| `sharedsignals.max_risk_score` | `84` |

## Endpoints

| Method | Path | Purpose |
|---|---|---|
| `POST` | `/v1/ssf/streams/:push_path/events` | Receive Security Event Tokens |
| `GET` | `/v1/ssf/admin/streams` | List inbound streams |
| `POST` | `/v1/ssf/admin/streams` | Register an inbound stream |
| `PATCH` | `/v1/ssf/admin/streams/:id` | Update a stream |
| `DELETE` | `/v1/ssf/admin/streams/:id` | Delete a stream |

## Lifecycle hooks

None on the sign-in path. It implements `riskengine.RiskContributor`, plus `PermissionChecker`
for its admin routes, `MigrationProvider` for the stream and signal tables, and `OnShutdown` to
stop the JWKS refresh ticker.

## Related

`riskengine` consumes what this plugin produces. `sso` and `social` are usually where the
transmitters on the other end come from.
