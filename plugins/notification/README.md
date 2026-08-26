# notification

Maps auth lifecycle events onto templated, localised notifications and dispatches them.

Where `email` builds a message and sends it, this plugin looks up a mapping, resolves a
template and a locale, and hands the result to the dispatcher. That buys you channels beyond
email, per-locale content, and the ability to switch off a single notification by nulling its
mapping without touching code.

It can dispatch asynchronously, which keeps a slow provider off your signup latency.

## Use it when

- You ship in more than one language
- You want notifications on channels other than email, push or in-app or SMS
- You need to disable or re-point one specific message without a release

## Skip it when

- You need one welcome email in one language. `email` does that with a third of the setup
- You have no template system wired. This plugin resolves templates, it does not author them
- You are already running `email`. Pick one, or both will fire on the same hook

## Wiring

```go
authsome.WithPlugin(notification.New(notification.Config{
    AppName:       "My App",
    BaseURL:       "https://example.com",
    DefaultLocale: "en",
    Async:         true,
})),
```

## Config

| Field | Type | Default | What it does |
|---|---|---|---|
| `AppName` | `string` | `AuthSome` | Name available to templates |
| `BaseURL` | `string` | empty | Root URL for building links |
| `EmailVerifyPath` | `string` | `/verify-email` | Page where a user enters their verification code. The recipient email is appended as `?email=` |
| `PasswordResetPath` | `string` | `/reset-password` | Reset page. The token is appended as `?token=` |
| `SignInPath` | `string` | `/sign-in` | Sign-in page, used for `login_url` in the welcome message |
| `DefaultLocale` | `string` | `en` | Locale used when the user has no preference |
| `Async` | `bool` | `false` | Dispatch in the background where a dispatcher is available |
| `Mappings` | `map[string]*Mapping` | defaults | Override hook-to-notification mapping. A nil entry disables that notification |

The three path fields exist because the default templates expect a URL even when the flow
delivers a code. They are how `verify_url` and `reset_url` get synthesised.

## Settings

| Key | Default |
|---|---|
| `notification.app_name` | `AuthSome` |
| `notification.base_url` | empty |
| `notification.default_locale` | `en` |
| `notification.async` | `false` |

## Lifecycle hooks

| Hook | What happens |
|---|---|
| `AfterSignUp` | Dispatches the welcome and verification notifications |
| `AfterUserCreate` | Dispatches the welcome notification for a user created outside signup |

## Related

`email` is the simpler alternative. `waitlist` contributes its own mappings here for approval
and rejection messages.
