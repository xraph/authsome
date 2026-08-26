# ipreputation

Scores the source IP against threat intelligence and blocks or flags the ones that come back
dirty.

Known botnet nodes, scanners and addresses with a history of abuse tend to show up on
reputation feeds long before they show up in your logs. This plugin asks a `Provider` for a
score from 0 to 100, then applies two thresholds: one where you want to know, and one where
you want it stopped. Results are cached so you are not paying for a lookup on every request.

## Use it when

- You're being hit by automated credential stuffing from rented infrastructure
- You already pay for a threat feed and want it enforced at the auth boundary
- You want a cheap first filter so the expensive checks run on fewer requests

## Skip it when

- You have no `Provider` wired. The plugin needs a data source and does not ship one
- Your users are largely on carrier-grade NAT or shared mobile IPs, where one bad actor poisons the score for thousands of innocent people
- Latency on the auth path is tight and your provider is a remote HTTP call

## Wiring

```go
authsome.WithPlugin(ipreputation.New(ipreputation.Config{
    Provider:       myThreatFeed, // implements ipreputation.Provider
    BlockThreshold: 80,
    WarnThreshold:  50,
    CacheTTL:       6 * time.Hour,
}))
```

## Config

| Field | Type | Default | What it does |
|---|---|---|---|
| `Provider` | `Provider` | none | Where scores come from. Required |
| `BlockThreshold` | `int` | `80` | Score at or above this is refused |
| `WarnThreshold` | `int` | `50` | Score at or above this is flagged but allowed |
| `CacheTTL` | `time.Duration` | `6h` | How long a score stays cached |
| `BlockMessage` | `string` | see settings | Error text returned on a block |

## Settings

| Key | Default |
|---|---|
| `ipreputation.block_threshold` | `80` |
| `ipreputation.warn_threshold` | `50` |
| `ipreputation.cache_ttl_hours` | `6` |
| `ipreputation.block_message` | `access denied due to IP reputation` |

## Lifecycle hooks

| Hook | What happens |
|---|---|
| `BeforeSignUp` | Blocks registration from a high-scoring address |
| `BeforeSignIn` | Blocks the sign-in before credentials are checked |
| `BeforePrincipalAuth` | Same gate for non-user principals |

## Related

`vpndetect` classifies the kind of connection. This plugin scores the address itself. Running
both gives you two independent reasons to be suspicious.
