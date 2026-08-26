# deviceverify

Spots a sign-in from a device this account has never used, and challenges it.

The pattern is familiar from every bank: sign in from a new laptop and you get a code by email
or SMS before you are let through. This plugin implements that as a challenge with a short
expiry, and can notify the user either way so an unexpected "new device" email tells them
something is wrong.

It runs after sign-in, so credentials have already been checked. The device check is the second
gate.

## Use it when

- Stolen passwords are the threat and you want something beyond the password without full MFA enrolment
- You want the user notified about new devices even when you are not blocking
- Your users work from a small, stable set of machines, which makes a new one genuinely notable

## Skip it when

- Your users are on many devices or clear cookies constantly. The challenge fires so often it becomes noise people click through
- You already require `mfa` on every sign-in. Adding this is a second challenge for the same risk
- You have no email or SMS delivery to send a challenge over

## Wiring

```go
authsome.WithPlugin(deviceverify.New(deviceverify.Config{
    NotifyOnNewDevice: true,
    ChallengeTTL:      10 * time.Minute,
})),
```

## Config

| Field | Type | Default | What it does |
|---|---|---|---|
| `NotifyOnNewDevice` | `bool` | `true` | Send a notification when a new device appears |
| `ChallengeTTL` | `time.Duration` | `10m` | How long a verification challenge stays valid |

## Settings

| Key | Default |
|---|---|
| `deviceverify.notify_on_new_device` | `true` |
| `deviceverify.challenge_ttl_minutes` | `10` |

## Lifecycle hooks

| Hook | What happens |
|---|---|
| `AfterSignIn` | Checks the device against the account's known devices and challenges an unknown one |

## Related

Device registration and trust live in the core engine, on `/v1/devices`. `anomaly` treats a new
device type as one score among several. `notification` and `email` carry the challenge.
