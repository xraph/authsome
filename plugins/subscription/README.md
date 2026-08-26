# subscription

Billing: plans, subscriptions, invoices, coupons, usage and feature entitlements.

None of the money logic lives here. It delegates to the ledger engine
(`github.com/xraph/ledger`) and spends its own effort on the part that touches auth: hooking
lifecycle events so a new organisation gets a trial without anyone clicking anything, and
answering the entitlement question your feature flags need.

You choose whether a tenant is an organisation or a user, which decides who the invoice
belongs to.

## Use it when

- You are charging for the product and want subscription state next to identity, not in a separate system
- Feature gating depends on plan, so your code needs to ask "is this tenant entitled to X"
- New signups should land on a trial automatically

## Skip it when

- You are not charging yet. This is a lot of surface area to carry for revenue you do not have
- Your billing already lives in Stripe and you are happy querying it there
- You have not wired the ledger engine. The plugin has nothing to delegate to without it

## Wiring

```go
authsome.WithPlugin(subscription.New(subscription.Config{
    DefaultPlanSlug:    "starter",
    AutoSubscribeOnOrg: true,
    TrialDays:          14,
})),
```

## Config

| Field | Type | Default | What it does |
|---|---|---|---|
| `PathPrefix` | `string` | `/v1/billing` | Where billing routes are mounted |
| `DefaultPlanSlug` | `string` | empty | Plan auto-assigned to a new tenant |
| `AutoSubscribeOnOrg` | `bool` | `false` | Subscribe on org creation, in organisation mode |
| `AutoSubscribeOnUser` | `bool` | `false` | Subscribe on signup, in user mode |
| `TrialDays` | `int` | `0` | Default trial length. `0` means no trial |
| `AllowSelfService` | `bool` | `false` | Let tenants change their own plan |

## Settings

| Key | Default |
|---|---|
| `subscription.default_plan` | empty |
| `subscription.tenant_mode` | `organization` |
| `subscription.auto_subscribe_org` | `true` |
| `subscription.auto_subscribe_user` | `false` |
| `subscription.trial_days` | `14` |
| `subscription.self_service_upgrade` | `true` |
| `subscription.grace_period_days` | `3` |

## Endpoints

Mounted under `PathPrefix`, `/v1/billing` by default.

| Method | Path | Purpose |
|---|---|---|
| `GET` | `/plans` | List plans |
| `POST` | `/plans` | Create a plan |
| `GET` | `/plans/:planId` | Read a plan |
| `POST` | `/plans/:planId/activate` | Activate a plan |
| `POST` | `/plans/:planId/archive` | Archive a plan |
| `GET` | `/subscriptions` | List subscriptions |
| `POST` | `/subscriptions` | Create a subscription |
| `GET` | `/subscriptions/active` | Read the active subscription |
| `POST` | `/subscriptions/:subId/change-plan` | Move to another plan |
| `POST` | `/subscriptions/:subId/pause` | Pause |
| `POST` | `/subscriptions/:subId/resume` | Resume |
| `POST` | `/subscriptions/:subId/cancel` | Cancel |
| `GET` | `/invoices` | List invoices |
| `GET` | `/invoices/:invoiceId` | Read an invoice |
| `POST` | `/invoices/:invoiceId/pay` | Mark paid |
| `POST` | `/invoices/:invoiceId/void` | Void |
| `GET` | `/entitlements/:featureKey` | Check a feature entitlement |
| `GET` | `/usage` | Usage summary |
| `GET` | `/coupons` | List coupons |
| `POST` | `/coupons` | Create a coupon |
| `DELETE` | `/coupons/:couponId` | Delete a coupon |

## Lifecycle hooks

| Hook | What happens |
|---|---|
| `AfterOrgCreate` | Auto-subscribes the new organisation, in organisation mode |
| `AfterSignUp` | Auto-subscribes the new user, in user mode |
| `AfterMemberAdd` | Records seat usage against the ledger, as `authsome.orgs.members` |

## Related

`organization` supplies the tenant in organisation mode. Entitlement checks pair well with
whatever feature flagging you already run.
