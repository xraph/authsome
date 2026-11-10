# @authsome/plugin-subscription

The client half of authsome's `subscription` plugin. You get the tenant's subscription, its plan, seats and feature checks as typed models, React hooks around them, and a pricing table you can drop in.

```bash
npm install @authsome/plugin-subscription @authsome/ui-react
```

The server needs the plugin too: `authsome.WithPlugin(subscription.New(...))`.

## Set it up

```tsx
import { AuthProvider } from "@authsome/ui-react";
import { subscription } from "@authsome/plugin-subscription";

<AuthProvider baseURL={url} publishableKey={pk} plugins={[subscription()]}>
  <App />
</AuthProvider>
```

## Hooks

```tsx
import { useSubscription, usePlans, Entitled } from "@authsome/plugin-subscription/react";

function PlanBadge() {
  const { plan, subscription, isLoading } = useSubscription();
  if (isLoading) return <Spinner />;
  if (!subscription) return <Badge>Free</Badge>;
  return <Badge>{plan.name}{subscription.willCancel && " (ending)"}</Badge>;
}
```

`useSubscription()` loads on first use and gives you the subscription, its plan, `canManage`, and every write: `subscribe`, `changePlan`, `setSeats`, `cancel`, `resume` and `openPortal`. Writes update every component reading the subscription, so you never refetch by hand. `isPending` is true while one is in flight.

A plan change either happens or needs paying for first:

```ts
const result = await changePlan("team");
if (result.type === "checkout") window.location.assign(result.url);
```

`<Entitled>` shows its children only when the plan includes the feature. While the check loads, on an error, or when the server doesn't have the plugin, you get the fallback:

```tsx
<Entitled feature="sso" fallback={<UpgradePrompt plan="team" />}>
  <SsoSettings />
</Entitled>
```

`usePlans()` and `useEntitlement(feature)` cover the rest.

## Components

```tsx
import {
  PricingTable,
  CurrentPlanCard,
  UsageMeter,
  ManageBillingButton,
} from "@authsome/plugin-subscription/components";

<CurrentPlanCard />
<PricingTable highlight="team" returnUrl={location.href} />
<UsageMeter feature="api_calls" label="API calls this month" />
<ManageBillingButton returnUrl={location.href} />
```

They're built on `@authsome/ui-components`, so they pick up the same theme as the sign-in forms. Buttons only show for someone who can manage billing. If you use Tailwind 4, add the package to your sources the way you do for `@authsome/ui-components`:

```css
@source "../node_modules/@authsome/plugin-subscription/dist";
```

`CurrentPlanCard` asks before it cancels. Every component, hook and `<Entitled>` takes a `tenantId` for support staff who hold `billing:manage` and look after a customer's billing:

```tsx
<CurrentPlanCard tenantId={customer.orgId} />
```

## Without React

```ts
import { AuthManager } from "@authsome/ui-core";
import { subscription } from "@authsome/plugin-subscription";

const auth = new AuthManager({ baseURL: url, plugins: [subscription()] });
const billing = auth.plugin(subscription);

await billing.load();
billing.onChange(() => render(billing.getSnapshot()));
```

The snapshot follows the session: signing out clears it, and signing in or switching organization reloads it. A response for an organization you've already left is dropped rather than shown. `billing.forTenant(id)` gives you the same API pinned to another tenant, with its own cache.

If the server doesn't have the subscription plugin, you get one console warning outside production, and the resources read as `unavailable` without a request.
