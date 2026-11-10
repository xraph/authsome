/**
 * React bindings for the subscription client plugin.
 *
 * ```tsx
 * <AuthProvider baseURL={url} plugins={[subscription()]}>...</AuthProvider>
 *
 * const { plan, subscription, changePlan } = useSubscription();
 * <Entitled feature="sso" fallback={<UpgradePrompt />}><SsoSettings /></Entitled>
 * ```
 */

import { useAuth, usePlugin } from "@authsome/ui-react";
import {
  useCallback,
  useEffect,
  useMemo,
  useState,
  useSyncExternalStore,
  type ReactNode,
} from "react";

import type { Entitlement, Plan, Subscription } from "./models";
import {
  subscription,
  type ChangeOptions,
  type ChangeResult,
  type Resource,
  type SubscriptionApi,
  type SubscriptionSnapshot,
} from "./plugin";

/**
 * Which tenant a hook or component is about. Leave it out for the session's
 * own; operators holding billing:manage can pass a customer's.
 */
export interface TenantOptions {
  tenantId?: string;
}

/** The plugin's API, for anything the hooks don't cover. */
export function useSubscriptionApi(options?: TenantOptions): SubscriptionApi {
  const api = usePlugin(subscription);
  // forTenant hands back one instance per ID, so this is stable.
  return options?.tenantId ? api.forTenant(options.tenantId) : api;
}

function useSnapshot(api: SubscriptionApi): SubscriptionSnapshot {
  return useSyncExternalStore(api.onChange, api.getSnapshot, api.getSnapshot);
}

/** Loads a resource once the user is signed in and it hasn't been loaded. */
function useAutoLoad(status: Resource<unknown>["status"], load: () => Promise<void>) {
  const { isAuthenticated } = useAuth();
  useEffect(() => {
    if (isAuthenticated && status === "idle") void load();
  }, [isAuthenticated, status, load]);
  return isAuthenticated;
}

const isPendingStatus = (status: Resource<unknown>["status"], signedIn: boolean) =>
  status === "loading" || (status === "idle" && signedIn);

/** Counts writes in flight, so components can disable buttons. */
function usePendingWrites() {
  const [pending, setPending] = useState(0);
  const track = useCallback(<T,>(work: () => Promise<T>): Promise<T> => {
    setPending((n) => n + 1);
    return work().finally(() => setPending((n) => n - 1));
  }, []);
  return { isPending: pending > 0, track };
}

export interface UseSubscriptionResult {
  /** The tenant's subscription, null when it has none (or before it loads). */
  subscription: Subscription | null;
  /** The subscription's plan. */
  plan: Plan | null;
  /** Whether you may change it. Hide upgrade buttons when false. */
  canManage: boolean;
  status: Resource<unknown>["status"];
  isLoading: boolean;
  error: Error | null;
  refresh: () => Promise<void>;
  subscribe: (plan: string, options?: ChangeOptions) => Promise<ChangeResult>;
  changePlan: (plan: string, options?: ChangeOptions) => Promise<ChangeResult>;
  setSeats: (seats: number, options?: Omit<ChangeOptions, "seats">) => Promise<ChangeResult>;
  cancel: (options?: { when?: "period_end" | "now" }) => Promise<Subscription>;
  resume: () => Promise<Subscription>;
  openPortal: (returnUrl?: string) => Promise<string>;
  /** A write is in flight. */
  isPending: boolean;
}

/**
 * The tenant's subscription and everything you can do to it. Loads on first
 * use; writes update every component reading it.
 */
export function useSubscription(options?: TenantOptions): UseSubscriptionResult {
  const api = useSubscriptionApi(options);
  const snap = useSnapshot(api);
  const signedIn = useAutoLoad(snap.subscription.status, api.load);
  const { isPending, track } = usePendingWrites();

  // Stable across renders, so they're safe in effect and memo dependencies.
  const writes = useMemo(
    () => ({
      subscribe: (plan: string, options?: ChangeOptions) => track(() => api.subscribe(plan, options)),
      changePlan: (plan: string, options?: ChangeOptions) => track(() => api.changePlan(plan, options)),
      setSeats: (seats: number, options?: Omit<ChangeOptions, "seats">) =>
        track(() => api.setSeats(seats, options)),
      cancel: (options?: { when?: "period_end" | "now" }) => track(() => api.cancel(options)),
      resume: () => track(() => api.resume()),
      openPortal: (returnUrl?: string) => track(() => api.openPortal(returnUrl)),
    }),
    [api, track],
  );

  return {
    subscription: snap.subscription.data,
    plan: snap.subscription.data?.plan ?? null,
    canManage: snap.canManage,
    status: snap.subscription.status,
    isLoading: isPendingStatus(snap.subscription.status, signedIn),
    error: snap.subscription.error,
    refresh: api.load,
    ...writes,
    isPending,
  };
}

/** The plans the tenant can buy, cheapest first. Loads on first use. */
export function usePlans(): {
  plans: Plan[];
  status: Resource<unknown>["status"];
  isLoading: boolean;
  error: Error | null;
  refresh: () => Promise<void>;
} {
  const api = useSubscriptionApi();
  const snap = useSnapshot(api);
  const signedIn = useAutoLoad(snap.plans.status, api.loadPlans);
  return {
    plans: snap.plans.data,
    status: snap.plans.status,
    isLoading: isPendingStatus(snap.plans.status, signedIn),
    error: snap.plans.error,
    refresh: api.loadPlans,
  };
}

/**
 * Whether the plan includes a feature. `allowed` stays false until the check
 * has loaded, and on an error, so gating on it fails closed.
 */
export function useEntitlement(feature: string, options?: TenantOptions): {
  allowed: boolean;
  entitlement: Entitlement | null;
  status: Resource<unknown>["status"];
  isLoading: boolean;
} {
  const api = useSubscriptionApi(options);
  const snap = useSnapshot(api);
  // Entitlements load with the subscription.
  const signedIn = useAutoLoad(snap.subscription.status, api.load);
  const entitlement = snap.entitlements.data[feature] ?? null;
  return {
    allowed: snap.entitlements.status === "ready" && entitlement?.allowed === true,
    entitlement,
    status: snap.entitlements.status,
    isLoading: isPendingStatus(snap.entitlements.status, signedIn),
  };
}

export interface EntitledProps extends TenantOptions {
  /** The plan feature to check, e.g. "sso". */
  feature: string;
  children: ReactNode;
  /** Shown when the plan doesn't include the feature, on an error, or when billing isn't installed. */
  fallback?: ReactNode;
  /** Shown while the check loads. Defaults to the fallback. */
  loading?: ReactNode;
}

/**
 * Shows its children only when the plan includes the feature. Anything short
 * of a positive answer (loading, an error, billing not installed) shows the
 * fallback, so a paywall never opens by accident.
 */
export function Entitled({ feature, children, fallback = null, loading, tenantId }: EntitledProps) {
  const { allowed, isLoading } = useEntitlement(feature, { tenantId });
  if (allowed) return <>{children}</>;
  if (isLoading) return <>{loading ?? fallback}</>;
  return <>{fallback}</>;
}

export { subscription };
export type { ChangeOptions, ChangeResult, Resource, SubscriptionApi, SubscriptionSnapshot };
