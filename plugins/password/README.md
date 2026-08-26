# password

Email and password sign-in, with a policy you can tighten without redeploying.

The core engine already knows how to hash and check a password. This plugin wraps that as a
named, toggleable auth method and adds the rules around it: a minimum length, an optional
special-character requirement, and a domain allowlist that stops anyone outside your company
from registering.

## Use it when

- You want the familiar credential flow as a primary sign-in method
- You're gating signup to company email domains for an internal tool
- You need password rules an admin can change from the dashboard, without a redeploy

## Skip it when

- You're building passwordless. `magiclink` and `passkey` cover that, and shipping both undercuts the security argument for either
- Your only clients are machines. Use `apikey`
- Users arrive through an identity provider you don't control. `sso` or `social` own the credential in that case

## Wiring

```go
authsome.WithPlugin(password.New(password.Config{
    MinLength:      10,
    RequireSpecial: true,
}))
```

Restrict signup to your own domains:

```go
authsome.WithPlugin(password.New(password.Config{
    AllowedDomains: []string{"example.com", "example.org"},
}))
```

## Config

| Field | Type | Default | What it does |
|---|---|---|---|
| `MinLength` | `int` | engine default | Minimum characters. `0` defers to the engine policy |
| `RequireSpecial` | `bool` | `false` | Require at least one special character |
| `AllowedDomains` | `[]string` | empty | Email domains permitted to sign up. Empty allows all |

## Settings

| Key | Default |
|---|---|
| `password.min_length` | `8` |
| `password.require_special` | `false` |
| `password.allowed_domains` | empty |

All three are enforceable and scoped global or per-app, so a stricter app can override the
global floor.

## Endpoints

This plugin does not add routes. Sign-up and sign-in stay on the core endpoints, `/v1/signup`
and `/v1/signin`.

## Lifecycle hooks

| Hook | What happens |
|---|---|
| `BeforeSignUp` | Rejects the registration if the password or the email domain fails policy |

It also registers a `StrategyProvider` so password shows up as a selectable auth method, and
an `AuthMethodContributor` so the dashboard knows the user has one.

## Related

Account lockout, rate limits, password history and expiry live in the engine config rather
than here. See `authsome.PasswordConfig` and `authsome.LockoutConfig`.
