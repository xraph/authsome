# email

Sends the transactional emails auth generates: welcome, verification, password reset,
invitation.

No SMTP code lives in this plugin. It listens on the lifecycle hooks, builds the message, and
hands it to whatever `bridge.Mailer` you configured on the engine. Swapping SendGrid for
Postmark is an engine-level change and this plugin never notices.

It is the simple option. If you need locales, templates managed outside the codebase, or
delivery over channels other than email, use `notification` instead.

## Use it when

- You want auth emails working with one plugin and a mailer, no template system to learn
- Email is your only channel and English is your only locale
- You are getting started and want something correct now, replaceable later

## Skip it when

- You need multiple locales or channels. `notification` handles both and this plugin does not
- Your marketing team owns email templates in a tool they will not give up
- You are already running `notification`. Running both means two systems trying to send the same welcome message

## Wiring

```go
authsome.WithMailer(myMailer),
authsome.WithPlugin(email.New(email.Config{
    From:    "noreply@example.com",
    AppName: "My App",
    BaseURL: "https://example.com",
})),
```

`BaseURL` is what links in the emails are built from, so it has to be the URL users reach you
on, not your internal service address.

## Config

| Field | Type | Default | What it does |
|---|---|---|---|
| `From` | `string` | `noreply@authsome.local` | Sender address |
| `AppName` | `string` | `AuthSome` | Name used in subjects and bodies |
| `BaseURL` | `string` | empty | Root URL for building links |

## Settings

| Key | Default |
|---|---|
| `email.from_address` | `noreply@authsome.local` |
| `email.app_name` | `AuthSome` |
| `email.base_url` | empty |

## Lifecycle hooks

| Hook | What happens |
|---|---|
| `AfterSignUp` | Sends the welcome and verification email |
| `AfterUserCreate` | Sends the welcome email for a user created outside signup |

## Related

`notification` is the richer alternative. `magiclink` brings its own `Mailer` for link
delivery.
