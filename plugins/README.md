# Plugins

Authsome ships 26 plugins. The core engine handles users, sessions, devices and RBAC; every
authentication method and every policy control is a plugin you register explicitly. Nothing
here is on by default, so you only carry what you use.

```go
engine, err := authsome.NewEngine(
    authsome.WithStore(store),
    authsome.WithPlugin(password.New()),
    authsome.WithPlugin(mfa.New(mfa.Config{Issuer: "My App"})),
)
```

Each plugin has its own README with config, settings, endpoints and the hooks it runs on.

## Pick by what you are trying to do

| You want | Use |
|---|---|
| Classic email and password sign-in | [`password`](password/) |
| Sign in with Google, GitHub, Apple | [`social`](social/) |
| Passwordless over email | [`magiclink`](magiclink/) |
| Face ID, Touch ID, YubiKey | [`passkey`](passkey/) |
| Phone number and an SMS code | [`phone`](phone/) |
| A second factor on top of a password | [`mfa`](mfa/) |
| Credentials for scripts and CI | [`apikey`](apikey/) |
| Enterprise SSO against Okta or Entra | [`sso`](sso/) |
| To be the OAuth2 provider others sign in with | [`oauth2provider`](oauth2provider/) |
| Companies as customers, not just people | [`organization`](organization/) |
| Automated user provisioning and offboarding | [`scim`](scim/) |
| To charge for the product | [`subscription`](subscription/) |
| Proof of GDPR consent | [`consent`](consent/) |
| A pre-launch signup queue | [`waitlist`](waitlist/) |
| Auth emails, quickly | [`email`](email/) |
| Localised, multi-channel messages | [`notification`](notification/) |
| To block whole countries | [`geofence`](geofence/) |
| To block Tor and anonymising proxies | [`vpndetect`](vpndetect/) |
| To block known-bad IPs | [`ipreputation`](ipreputation/) |
| To catch account takeover across continents | [`impossibletravel`](impossibletravel/) |
| Behavioural detection on login patterns | [`anomaly`](anomaly/) |
| A challenge when a new device appears | [`deviceverify`](deviceverify/) |
| One risk score instead of five separate blocks | [`riskengine`](riskengine/) |
| Compromise signals pushed from Google or Okta | [`sharedsignals`](sharedsignals/) |
| To let an AI agent act for a user, on a leash | [`agentauth`](agentauth/) |
| Country and city on every session | [`geoip`](geoip/) |

## By category

| Category | Plugins |
|---|---|
| Authentication | [`password`](password/), [`social`](social/), [`magiclink`](magiclink/), [`passkey`](passkey/), [`mfa`](mfa/), [`apikey`](apikey/), [`sso`](sso/), [`phone`](phone/) |
| Identity | [`organization`](organization/), [`consent`](consent/), [`subscription`](subscription/), [`waitlist`](waitlist/) |
| Communication | [`email`](email/), [`notification`](notification/) |
| Risk and security | [`anomaly`](anomaly/), [`geofence`](geofence/), [`geoip`](geoip/), [`impossibletravel`](impossibletravel/), [`ipreputation`](ipreputation/), [`riskengine`](riskengine/), [`vpndetect`](vpndetect/), [`sharedsignals`](sharedsignals/) |
| Provisioning | [`scim`](scim/), [`deviceverify`](deviceverify/), [`oauth2provider`](oauth2provider/), [`agentauth`](agentauth/) |

## What depends on what

Most plugins stand alone. These do not, and registration order matters where it says so.

- `geofence`, `vpndetect` and `impossibletravel` read the location that `geoip` resolves. Register `geoip` first or they have nothing to work with.
- `anomaly` uses `geoip` for its new-country signal, and degrades quietly without it.
- `sharedsignals` implements `riskengine.RiskContributor`. Its signals are stored either way, but nothing acts on them unless `riskengine` is registered.
- `riskengine` finds contributors automatically at init, so you rarely pass them by hand.
- `sso` and `scim` scope their connections to an organisation, which means `organization`.
- `subscription` in organisation mode needs `organization` for its tenants.
- `phone` and `mfa` share the engine's SMS bridge. Configure it once.
- `waitlist` contributes notification mappings that `notification` renders.
- `agentauth` plugs its consent gate into `oauth2provider`, automatically when both are registered on the same engine. Its org policy surface needs `organization`.

Pick one of `email` or `notification`, not both. They hook the same events and you will send
every message twice.

## How a plugin decides things

Two mechanisms, and the difference matters when you are debugging.

`Config` is passed at construction and fixed for the life of the process. `settings` are
dynamic: they are declared with `settings.Define`, scoped global or per-app, and changeable at
runtime from the admin dashboard or the settings API. Where a plugin has both, the setting is
what actually applies. Every plugin README lists both tables.

Plugins also declare their lifecycle hooks as compile-time assertions:

```go
var (
    _ plugin.Plugin        = (*Plugin)(nil)
    _ plugin.BeforeSignUp  = (*Plugin)(nil)
    _ plugin.RouteProvider = (*Plugin)(nil)
)
```

That block at the top of any `plugin.go` is the fastest answer to "when does this thing run".

## Writing your own

See [docs/content/docs/plugins/creating-plugins.mdx](../docs/content/docs/plugins/creating-plugins.mdx).
The interfaces you can implement are in [`plugin/plugin.go`](../plugin/plugin.go).
