# social

Sign in with Google, GitHub, Apple, Microsoft and about thirty others.

The plugin runs the whole OAuth2 dance: the redirect out, the callback, the code exchange, and
the profile fetch. What comes back gets linked to a core user account, so a person who signed
up with a password can add GitHub later and both routes land on the same identity.

Providers can be configured in code at startup or added at runtime through the admin routes,
which is the difference between shipping a release and flipping a switch.

## Use it when

- You want signup in one click, with no password and no email round trip
- Your users already live in one ecosystem, GitHub for a developer tool, Google for most consumer products
- You want verified email addresses for free, since the provider already did that work

## Skip it when

- You are building for enterprise buyers. They want `sso` against their own identity provider, not personal Google accounts
- You cannot register a stable callback URL. Every provider pins it, and a mismatch fails the flow
- Account linking rules matter to you and you have not thought them through. Two providers returning the same email is a decision, not an accident

## Wiring

```go
authsome.WithPlugin(social.New(social.Config{
    Domain: "https://api.example.com",
    Providers: []social.Provider{
        social.NewGoogleProvider(social.ProviderConfig{
            ClientID:     googleID,
            ClientSecret: googleSecret,
        }),
        social.NewGitHubProvider(social.ProviderConfig{
            ClientID:     githubID,
            ClientSecret: githubSecret,
        }),
    },
}))
```

Set `Domain` and the callback URL is derived for you as
`{Domain}/v1/social/{provider}/callback`. Leave it empty and the URL comes off the request
Host header, which is fine locally and risky behind a proxy.

## Config

| Field | Type | Default | What it does |
|---|---|---|---|
| `Providers` | `[]Provider` | empty | Providers enabled at startup |
| `Domain` | `string` | request host | External base URL used to build callback URLs |
| `SessionTokenTTL` | `time.Duration` | `1h` | Lifetime of a session created by social sign-in |
| `SessionRefreshTTL` | `time.Duration` | `30d` | Lifetime of that session's refresh token |

## Settings

| Key | Default |
|---|---|
| `social.providers` | `[]` |
| `social.session_token_ttl_seconds` | `3600` |
| `social.session_refresh_ttl_seconds` | `2592000` |
| `auth.allowed_frontend_urls` | empty |

`auth.allowed_frontend_urls` is the redirect allowlist. Fill it in. An open redirect on an
OAuth callback is how tokens get stolen.

## Endpoints

| Method | Path | Purpose |
|---|---|---|
| `POST` | `/v1/social/:provider` | Start the OAuth flow |
| `GET` | `/v1/social/:provider/callback` | Handle the provider callback and mint a session |
| `GET` | `/v1/social/providers` | List providers available to the caller |
| `GET` | `/v1/admin/social/providers/catalog` | Every provider the plugin supports |
| `GET` | `/v1/admin/social/providers` | List configured providers |
| `PUT` | `/v1/admin/social/providers/:provider` | Configure a provider at runtime |
| `DELETE` | `/v1/admin/social/providers/:provider` | Remove a provider |

## Lifecycle hooks

None on the sign-in path. It registers as an `AuthMethodContributor` and an
`AuthMethodUnlinker`, so a user can see and disconnect a linked provider, plus a
`MigrationProvider` for the connection table.

## Providers

Amazon, Apple, Bitbucket, Discord, Dropbox, Facebook, GitHub, GitLab, Google, Instagram, LINE,
LinkedIn, Microsoft, Patreon, Pinterest, Slack, Spotify, Strava, Twitch, Twitter, Yahoo, Zoom.

## Related

`sso` is the enterprise equivalent, scoped per organisation. `oauth2provider` is the mirror
image: it makes you the provider other people sign in with.
