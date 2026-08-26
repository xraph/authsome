# impossibletravel

Catches the case where the same account signs in from two places too far apart to have
travelled between them.

The arithmetic is simple. Take the last login location, take this one, work out the distance
and the elapsed time, and if the implied speed beats a passenger jet then something is wrong.
Lagos at nine and Toronto at ten means one of those two sessions is not the account holder.

## Use it when

- Credential stuffing is a real problem for you and you want a signal that stolen passwords can't fake
- You want a high-confidence account takeover alert. This check has very few false positives
- You're feeding a SIEM and need events worth paging someone about

## Skip it when

- You are not running `geoip`, since there are no coordinates to measure without it
- Your users sit behind a corporate proxy that egresses from a different continent to where they actually are
- The account is brand new. The check needs a login history to compare against

## Wiring

```go
authsome.WithPlugin(geoip.New(geoip.Config{DatabasePath: "/var/lib/geoip/GeoLite2-City.mmdb"})),
authsome.WithPlugin(impossibletravel.New(impossibletravel.Config{
    MaxSpeedKmH: 900,
    Action:      "flag",
})),
```

Start on `flag`. Watch what it catches for a few weeks, then decide whether you're willing to
move it to `block`.

## Config

| Field | Type | Default | What it does |
|---|---|---|---|
| `MaxSpeedKmH` | `float64` | `900` | Fastest speed treated as plausible. 900 km/h is roughly a commercial flight |
| `MinDistanceKm` | `float64` | `500` | Distances under this are ignored, which keeps city-level GeoIP jitter out of the results |
| `LookbackWindow` | `time.Duration` | `24h` | How far back to search for the previous login |
| `Action` | `string` | `flag` | `flag` records a security event. `block` refuses the sign-in |

## Settings

| Key | Default |
|---|---|
| `impossibletravel.max_speed_kmh` | `900` |
| `impossibletravel.min_distance_km` | `500` |
| `impossibletravel.lookback_window_hours` | `24` |
| `impossibletravel.action` | `flag` |

## Lifecycle hooks

| Hook | What happens |
|---|---|
| `AfterSignIn` | Compares this login against the previous one and flags or blocks |
| `AfterPrincipalAuth` | Same check for non-user principals |

## Related

Requires `geoip`. `anomaly` covers the softer patterns like odd hours and new devices, where
this plugin only cares about physics.
