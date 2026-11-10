/**
 * @authsome/plugin-subscription: the client half of authsome's
 * subscription plugin.
 *
 * ```ts
 * import { subscription } from "@authsome/plugin-subscription";
 *
 * const auth = new AuthManager({ baseURL, plugins: [subscription()] });
 * const billing = auth.plugin(subscription);
 * await billing.load();
 * ```
 *
 * React hooks live in "@authsome/plugin-subscription/react" and styled
 * components in "@authsome/plugin-subscription/components".
 */

export {
  subscription,
  createSubscriptionApi,
  type ChangeOptions,
  type ChangeResult,
  type Resource,
  type SubscriptionApi,
  type SubscriptionSnapshot,
} from "./plugin";
export {
  currencyDecimals,
  formatMoney,
  type Entitlement,
  type Money,
  type Plan,
  type PlanFeature,
  type Price,
  type Seats,
  type Subscription,
  type SubscriptionStatus,
} from "./models";
