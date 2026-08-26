# geoip

Resolves the IP address on an auth request into a country, city, coordinates and a set of
connection flags.

On its own this plugin blocks nothing. It's the data layer the rest of the security plugins
read from, so if you plan to run `geofence`, `vpndetect` or `impossibletravel`, you need this
one registered first. Lookups go through a MaxMind GeoLite2 database on local disk and get
cached, so the hot path stays cheap.

## Use it when

- You want the country and city recorded on every sign-in for your audit trail
- You're enabling any of the location-aware security plugins, which all read the location this plugin attaches
- You want to show users a readable "signed in from Lagos, Nigeria" in a session list

## Skip it when

- You don't ship a GeoLite2 database with your deployment. Without `DatabasePath` there is nothing to look up
- Your traffic is entirely server-to-server on a private network, where the source IP tells you nothing about a person

## Wiring

```go
authsome.WithPlugin(geoip.New(geoip.Config{
    DatabasePath: "/var/lib/geoip/GeoLite2-City.mmdb",
    CacheTTL:     24 * time.Hour,
}))
```

Register it before the plugins that depend on it.

## Config

| Field | Type | Default | What it does |
|---|---|---|---|
| `DatabasePath` | `string` | none | Path to the MaxMind `GeoLite2-City.mmdb` file. Required |
| `CacheTTL` | `time.Duration` | `24h` | How long a resolved location stays cached |

## Settings

Tunable at runtime from the admin dashboard or the settings API.

| Key | Default |
|---|---|
| `geoip.cache_ttl_hours` | `24` |

## Lifecycle hooks

| Hook | What happens |
|---|---|
| `BeforeSignUp` | Resolves the request IP and attaches the location to context |
| `BeforeSignIn` | Same, so downstream plugins can read it before the decision is made |
| `AfterSignIn` | Records the resolved location against the session |
| `OnShutdown` | Closes the database handle |

Downstream plugins pull the location off the context with `geoip.WithGeoLocation`.

## Related

`geofence` and `vpndetect` read this plugin's output directly. `impossibletravel` needs the
coordinates it resolves. `anomaly` uses the country for its new-country signal.
