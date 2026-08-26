# riskengine

Collects risk signals from other plugins, combines them into one score, and decides whether to
allow the request, force a step-up, or block it.

Every other security plugin here answers yes or no on its own. That gets you a system where
any single grumpy signal locks a legitimate user out. This plugin gives you the middle option.
Contributors hand in a score and a weight, the engine averages them, and three thresholds turn
the result into an action: allow, challenge with MFA, or block.

Contributors are discovered automatically. Any registered plugin implementing
`riskengine.RiskContributor` gets picked up at init, so you rarely need to pass them by hand.

## Use it when

- You'd rather step a suspicious sign-in up to MFA than refuse it
- You're running several security plugins and want one decision instead of a race between them
- You want to tune how much each signal counts by changing a weight, no code change

## Skip it when

- You have exactly one signal. A single contributor's score becomes the whole composite, so you have built an indirection for nothing
- Your policy is genuinely binary and you have no MFA to step up to
- You need the block reason to be specific. The composite score tells you the request was risky, not which signal made it so

## Wiring

```go
authsome.WithPlugin(riskengine.New()), // contributors are auto-discovered
```

Or set the thresholds and weights explicitly:

```go
authsome.WithPlugin(riskengine.NewWithConfig(riskengine.Config{
    LowThreshold:    30,
    MediumThreshold: 60,
    HighThreshold:   85,
    Weights: map[string]float64{
        "sharedsignals": 2.0,
    },
}))
```

## Config

| Field | Type | Default | What it does |
|---|---|---|---|
| `LowThreshold` | `int` | `30` | Below this, allow without comment |
| `MediumThreshold` | `int` | `60` | At or above this, require step-up MFA |
| `HighThreshold` | `int` | `85` | At or above this, refuse the request |
| `Weights` | `map[string]float64` | `1.0` each | Multiplier per contributor name |
| `BlockMessage` | `string` | empty | Error text returned on a block |

## Settings

| Key | Default |
|---|---|
| `riskengine.low_threshold` | `30` |
| `riskengine.medium_threshold` | `60` |
| `riskengine.high_threshold` | `85` |
| `riskengine.block_message` | empty |

## Lifecycle hooks

| Hook | What happens |
|---|---|
| `BeforeSignIn` | Scores the attempt and allows, challenges or blocks |
| `BeforePrincipalAuth` | Same decision for non-user principals |
| `BeforeSessionCreate` | Last gate before a session is minted |

## Writing a contributor

```go
type RiskContributor interface {
    Name() string
    EvaluateRisk(ctx context.Context, req *RiskRequest) (*RiskSignal, error)
}
```

Return a `RiskSignal` with a `Score` from 0 to 100, a `Weight`, and a `Reason` a human can
read in an audit log. `sharedsignals` is the worked example in this repo.

## Related

`sharedsignals` ships as a contributor. `anomaly`, `geofence`, `ipreputation` and `vpndetect`
currently gate directly rather than contributing, so they act independently of this engine.
