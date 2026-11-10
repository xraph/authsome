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
| `TrialDays` | `int` | `0` | Trial length for auto-subscribed tenants. `0` means no trial. Subscriptions created through the API take the plan's own `trial_days` |
| `Checkout` | `CheckoutProvider` | `nil` | Takes payment before a tenant moves onto a paid plan. See [Taking payment](#taking-payment) |
| `AllowSelfService` | `bool` | `false` | Deprecated and unused. Self-service is the `subscription.self_service_upgrade` setting |
| `DisableRoutes` | `bool` | `false` | Mount no HTTP routes. Set it when you serve your own billing API on top of the plugin's ledger |
| `DisableAutoSubscribe` | `bool` | `false` | Turn off the org-create and signup hooks, whatever the settings say. Set it when you create subscriptions yourself |

## Settings

| Key | Default |
|---|---|
| `subscription.default_plan` | empty |
| `subscription.tenant_mode` | `organization` |
| `subscription.auto_subscribe_org` | `true` |
| `subscription.auto_subscribe_user` | `false` |
| `subscription.trial_days` | `14` |
| `subscription.self_service_upgrade` | `true` |
| `subscription.billing_roles` | `["owner"]` |
| `subscription.grace_period_days` | `3` |

`billing_roles` can be set per app and per organization, so an organization can let its
admins manage billing without changing it for everyone else.

## Self-service

These are the routes your app's own UI calls. They work on the caller's tenant, so you
never pass a subscription ID, and you don't have to pass a tenant either: in organization
mode it's the session's active organization, in user mode it's the user.

| Method | Path | Purpose |
|---|---|---|
| `GET` | `/subscription` | Your subscription (with its plan), or `null`, plus `can_manage` |
| `POST` | `/subscription` | Subscribe. Body: `plan` (slug or ID), optional `seats` |
| `PATCH` | `/subscription` | Change plan, seats, or both |
| `POST` | `/subscription/cancel` | Cancel at the period end, or now with `{"when":"now"}` |
| `POST` | `/subscription/resume` | Take back a cancellation scheduled for the period end |
| `GET` | `/entitlements` | Every feature on your plan, checked in one call |
| `POST` | `/portal` | A billing portal URL, when your checkout provider has one |

Anyone in the organization can read. To write you need self-service on
(`subscription.self_service_upgrade`) and a role listed in `subscription.billing_roles`,
which is just `owner` out of the box. `GET /subscription` tells you which with
`can_manage`, so you can hide the upgrade button from people who'd get a 403.

Writes answer with the updated subscription, so you don't need to fetch it again:

```json
{
  "subscription": {
    "id": "sub_...",
    "status": "active",
    "plan": { "slug": "team", "name": "Team", "amount_minor": 4900, "features": [] },
    "current_period_end": "2026-12-10T09:30:00Z",
    "cancel_at": null,
    "seats": { "feature_key": "seats", "used": 3, "limit": 10 }
  }
}
```

A cancellation at the period end leaves `status` at `active` and sets `cancel_at`. That's
your cue to show "ends on" and a button that calls `/subscription/resume`.

Pausing is still an operator action, through the ID routes below. A tenant can't unpause
itself, because a pause is often how you stop service for an unpaid invoice.

## Taking payment

Without a checkout provider, subscribing grants the plan straight away. That's fine for free
plans and trials. To charge for paid ones, give the plugin a `CheckoutProvider`:

```go
type stripeCheckout struct{ /* your Stripe client */ }

func (s stripeCheckout) Checkout(ctx context.Context, req subscription.CheckoutRequest) (string, error) {
    // Create a checkout session for req.Plan and req.TenantID, and return its URL.
}

authsome.WithPlugin(subscription.New(subscription.Config{
    Checkout: stripeCheckout{},
}))
```

Subscribing or changing to a plan with a price now answers `{"checkout_url": "..."}` and
changes nothing. Send the customer there. When your payment provider's webhook tells you
they paid, finish the change yourself with `plugin.Service().Subscribe` or
`plugin.Service().ChangePlan`, using the tenant, plan and seat count from the
`CheckoutRequest`. Cancel, resume and free plans never go through the provider.

If your provider also implements `PortalProvider`, `POST /portal` returns a link to its
hosted page for payment details and invoices. Without one that route answers 404.

A provider error is logged and the client gets a 502, so nothing from your payment
provider's error text reaches the browser.

## Endpoints

Mounted under `PathPrefix`, `/v1/billing` by default. The self-service routes are above;
these are the rest, mostly for operators.

| Method | Path | Purpose |
|---|---|---|
| `GET` | `/plans` | List plans you can buy: active base plans, cheapest first. Operators can pass `status=all`, and anyone can pass `include_addons=true` |
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

Subscription, invoice, entitlement and usage reads on these routes are scoped to a tenant:
pass its `tenant_id`. (The self-service routes above fill it in from the session instead.)
You can read a tenant you belong to (organisation mode) or are (user mode).
Anyone else gets a 404, whether the tenant exists or not, unless they hold
`billing:manage`, which reads every tenant. Plans stay app-wide.

The ID routes answer a transition the ledger refuses (changing or resuming a canceled
subscription, say) with a 409. You need
`billing:manage` for `GET /coupons` and for the dashboard billing views as well, because
both list data from the whole app. One caveat: on an engine with no RBAC, the coupon list
and the write routes only ask for a session, since there is no permission to check. The
tenant reads go the other way and treat a missing RBAC as not holding `billing:manage`.

`billing:manage` doesn't cross apps. When your request is scoped to an app (an API key
or an app-bound session), every billing route stays inside it. An `app_id` naming another
app gets a 404, and so does a subscription, invoice, coupon or plan from another app
addressed by ID. Leave `app_id` off and the route uses your request's app.

To run your own billing API, set `DisableRoutes` and call `plugin.Service()` from your
handlers. Pass your ledger to `authsome.WithLedgerEngine` and the plugin picks it up
when the engine starts, so you can skip `RebindLedgerOnPlugins`. If you build the ledger
after the engine and hand it over with `SetLedgerEngine`, you still need to call it.

## Lifecycle hooks

| Hook | What happens |
|---|---|
| `AfterOrgCreate` | Auto-subscribes the new organisation, in organisation mode |
| `AfterSignUp` | Auto-subscribes the new user, in user mode |
| `AfterMemberAdd` | Records seat usage against the ledger, as `authsome.orgs.members` |

## Related

`organization` supplies the tenant in organisation mode. Entitlement checks pair well with
whatever feature flagging you already run.
