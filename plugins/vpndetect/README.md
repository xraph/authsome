# vpndetect

Flags or blocks sign-ins that arrive over a VPN, an anonymising proxy or a Tor exit node.

It does not do its own network probing. It reads the connection flags that `geoip` already
resolved and applies your policy to them, so the cost is a map lookup rather than an outbound
call. Tor is blocked by default. VPN and proxy are not, because plenty of legitimate people
use both.

## Use it when

- Tor traffic to your auth endpoints is never legitimate and you want it gone
- You're in a regulated space where the origin of a session has to be attributable
- You want the VPN flag recorded on sessions for later investigation, without blocking anything today

## Skip it when

- A meaningful share of your users are on a corporate VPN. Blocking that locks out the exact people who pay you
- Privacy-conscious users are your market. Blocking Tor is a product decision, not just a security one
- You are not running `geoip`, which is where the flags come from

## Wiring

```go
authsome.WithPlugin(vpndetect.New(vpndetect.Config{
    BlockTor:   true,
    BlockVPN:   false,
    BlockProxy: false,
}))
```

Leave all three false and the plugin still runs. It records the flags and blocks nothing,
which is a reasonable way to find out how much of your traffic this would have caught.

## Config

| Field | Type | Default | What it does |
|---|---|---|---|
| `BlockVPN` | `bool` | `false` | Refuse connections flagged as VPN |
| `BlockProxy` | `bool` | `false` | Refuse connections flagged as an anonymising proxy |
| `BlockTor` | `bool` | `true` | Refuse connections from a Tor exit node |
| `BlockMessage` | `string` | empty | Error text returned on a block |

## Settings

| Key | Default |
|---|---|
| `vpndetect.block_vpn` | `false` |
| `vpndetect.block_proxy` | `false` |
| `vpndetect.block_tor` | `true` |
| `vpndetect.block_message` | empty |

## Lifecycle hooks

| Hook | What happens |
|---|---|
| `BeforeSignUp` | Blocks registration from a flagged connection |
| `BeforeSignIn` | Blocks the sign-in before credentials are checked |
| `BeforePrincipalAuth` | Same gate for non-user principals |

## Related

Requires `geoip`. Pairs naturally with `ipreputation`, which scores the address itself rather
than the kind of tunnel it arrived through.
