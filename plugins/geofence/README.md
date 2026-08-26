# geofence

Allows or blocks authentication based on the country the request came from.

You get two ways to express the rule. An allowlist, where only the countries you name can
authenticate, or a blocklist, where everyone except the countries you name can. Pick one. The
check runs before sign-in and before sign-up, so a blocked country can't create an account
either.

## Use it when

- You operate under a licence or sanctions regime that only permits certain countries
- Your product genuinely has no users outside a handful of markets, and you'd rather cut the attack surface
- You need a fast, blunt control while you build something more nuanced

## Skip it when

- You have travelling users. A country block hits them on holiday and they will open a support ticket
- You want a graded response instead of a hard yes or no. `riskengine` can step up to MFA where this plugin only blocks
- You are not running `geoip`. Without a resolved location this plugin has nothing to match on

## Wiring

```go
authsome.WithPlugin(geoip.New(geoip.Config{DatabasePath: "/var/lib/geoip/GeoLite2-City.mmdb"})),
authsome.WithPlugin(geofence.New(geofence.Config{
    DefaultPolicy:    "allow_all",
    BlockedCountries: []string{"KP", "IR"},
})),
```

For an allowlist, flip `DefaultPolicy` to `deny_all` and fill `AllowedCountries` instead.

## Config

| Field | Type | Default | What it does |
|---|---|---|---|
| `DefaultPolicy` | `string` | `allow_all` | Action when no rule matches. `allow_all` or `deny_all` |
| `AllowedCountries` | `[]string` | empty | ISO 3166-1 alpha-2 codes. When set, only these get through |
| `BlockedCountries` | `[]string` | empty | ISO 3166-1 alpha-2 codes that are refused |
| `BlockMessage` | `string` | `access denied from your location` | Error text returned to a blocked caller |

## Settings

| Key | Default |
|---|---|
| `geofence.default_policy` | `allow_all` |
| `geofence.allowed_countries` | `[]` |
| `geofence.blocked_countries` | `[]` |
| `geofence.block_message` | `access denied based on location` |

## Lifecycle hooks

| Hook | What happens |
|---|---|
| `BeforeSignUp` | Refuses registration from a blocked country |
| `BeforeSignIn` | Refuses the sign-in before credentials are checked |
| `BeforePrincipalAuth` | Same gate for non-user principals |

## Related

Requires `geoip`. If you want country to be one input among several rather than the whole
decision, look at `riskengine`.
