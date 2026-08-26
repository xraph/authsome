# anomaly

Watches a user's login history and raises a score when the current sign-in does not look like
the ones before it.

Four patterns feed the score: signing in at an hour this account never signs in at, a country
it has never been seen from, a burst of attempts well above its normal frequency, and a device
type it has never used. None of these is proof of anything on its own. Together they are a
decent proxy for "this does not feel like the same person".

The plugin stays quiet until it has enough history to have an opinion, which is ten logins by
default.

## Use it when

- You want behavioural detection without writing the baselining yourself
- Your users have settled habits, so a deviation actually means something
- You want a risk score that feeds alerting, not a hard block

## Skip it when

- Your accounts are shared or machine-driven, where there is no stable pattern to deviate from
- Most of your accounts are new. Under `MinLoginHistory` logins the plugin does nothing
- You need a hard block. This plugin scores and flags, it does not refuse

## Wiring

```go
authsome.WithPlugin(anomaly.New(anomaly.Config{
    MinLoginHistory:   10,
    RiskThreshold:     70,
    EnableTimeAnomaly: true,
    EnableGeoAnomaly:  true,
}))
```

Geo detection wants `geoip` registered, otherwise there's no country to compare.

## Config

| Field | Type | Default | What it does |
|---|---|---|---|
| `MinLoginHistory` | `int` | `10` | Logins needed before detection starts |
| `RiskThreshold` | `int` | `70` | Score out of 100 above which an alert is raised |
| `EnableTimeAnomaly` | `bool` | `true` | Detect logins at unusual hours for this account |
| `EnableGeoAnomaly` | `bool` | `true` | Detect logins from a country this account has never used |

## Settings

| Key | Default |
|---|---|
| `anomaly.min_login_history` | `10` |
| `anomaly.risk_threshold` | `70` |
| `anomaly.enable_time_anomaly` | `true` |
| `anomaly.enable_geo_anomaly` | `true` |

## Lifecycle hooks

| Hook | What happens |
|---|---|
| `AfterSignIn` | Scores the sign-in against the account's history and raises an alert past the threshold |
| `AfterPrincipalAuth` | Same scoring for non-user principals |

## Related

`impossibletravel` is the sharp version of the geo signal. `deviceverify` handles the new
device case with a challenge, not a score.
