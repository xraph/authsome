import { definePlugin, type PluginContext } from "@authsome/ui-core";

import {
  toEntitlement,
  toPlan,
  toSubscription,
  type Entitlement,
  type EntitlementJSON,
  type Plan,
  type PlanJSON,
  type Subscription,
  type SubscriptionJSON,
} from "./models";

/**
 * One piece of server state. "unavailable" means the server doesn't have the
 * subscription plugin installed; treat it like a plan without the feature.
 */
export interface Resource<T> {
  status: "idle" | "loading" | "ready" | "error" | "unavailable";
  data: T;
  error: Error | null;
}

/** Everything the plugin knows, as one immutable snapshot. */
export interface SubscriptionSnapshot {
  /** The tenant's subscription; data is null when it has none. */
  subscription: Resource<Subscription | null>;
  /** Whether the signed-in user may change the subscription. */
  canManage: boolean;
  /** Plans the tenant can buy, cheapest first. */
  plans: Resource<Plan[]>;
  /** Every feature on the current plan, checked. */
  entitlements: Resource<Record<string, Entitlement>>;
}

/** What a plan or seat change did. */
export type ChangeResult =
  | { type: "changed"; subscription: Subscription }
  /** Payment comes first: send the customer to url. Nothing changed yet. */
  | { type: "checkout"; url: string };

export interface ChangeOptions {
  seats?: number;
  /** Where the payment page sends the customer back to. */
  returnUrl?: string;
  /** Where it sends them if they back out. */
  cancelUrl?: string;
}

export interface SubscriptionApi {
  getSnapshot(): SubscriptionSnapshot;
  /** Listen for snapshot changes. Returns a function that stops listening. */
  onChange(listener: () => void): () => void;
  /** Loads the subscription and its entitlements. */
  load(): Promise<void>;
  /** Loads the plans the tenant can buy. */
  loadPlans(): Promise<void>;
  /** Starts a subscription on a plan, by slug or ID. */
  subscribe(plan: string, options?: ChangeOptions): Promise<ChangeResult>;
  /** Moves the subscription to another plan, by slug or ID. */
  changePlan(plan: string, options?: ChangeOptions): Promise<ChangeResult>;
  /** Changes the seat count on the current plan. */
  setSeats(seats: number, options?: Omit<ChangeOptions, "seats">): Promise<ChangeResult>;
  /** Cancels at the end of the period (the default) or now. */
  cancel(options?: { when?: "period_end" | "now" }): Promise<Subscription>;
  /** Takes back a cancellation scheduled for the end of the period. */
  resume(): Promise<Subscription>;
  /** A link to the payment provider's billing portal. */
  openPortal(returnUrl?: string): Promise<string>;
  /** The cached check for a feature, null when it isn't on the plan or not loaded. */
  entitlement(feature: string): Entitlement | null;
}

const idle = <T>(data: T): Resource<T> => ({ status: "idle", data, error: null });

function initialSnapshot(): SubscriptionSnapshot {
  return {
    subscription: idle<Subscription | null>(null),
    canManage: false,
    plans: idle<Plan[]>([]),
    entitlements: idle<Record<string, Entitlement>>({}),
  };
}

function asError(err: unknown): Error {
  return err instanceof Error ? err : new Error(String(err));
}

/** Builds the subscription API over a plugin context. */
export function createSubscriptionApi(ctx: PluginContext): SubscriptionApi {
  let snapshot = initialSnapshot();
  const listeners = new Set<() => void>();
  // Bumped whenever the session or organization changes, so a response for
  // the one the user just left is dropped instead of shown.
  let generation = 0;

  const set = (patch: Partial<SubscriptionSnapshot>) => {
    snapshot = { ...snapshot, ...patch };
    for (const listener of listeners) {
      try {
        listener();
      } catch {
        // A listener's error mustn't stop the others.
      }
    }
  };

  const unavailable = () => ctx.isServerEnabled() === false;
  const signedIn = () => ctx.getState().status === "authenticated";

  const loadEntitlements = async (gen: number) => {
    set({ entitlements: { ...snapshot.entitlements, status: "loading", error: null } });
    try {
      const res = await ctx.request<{ entitlements: Record<string, EntitlementJSON> }>(
        "GET",
        "/v1/billing/entitlements",
      );
      if (gen !== generation) return;
      const entitlements: Record<string, Entitlement> = {};
      for (const [key, value] of Object.entries(res.entitlements ?? {})) {
        entitlements[key] = toEntitlement(value);
      }
      set({ entitlements: { status: "ready", data: entitlements, error: null } });
    } catch (err) {
      if (gen !== generation) return;
      set({ entitlements: { ...snapshot.entitlements, status: "error", error: asError(err) } });
    }
  };

  const load = async () => {
    if (unavailable()) {
      set({
        subscription: { status: "unavailable", data: null, error: null },
        entitlements: { status: "unavailable", data: {}, error: null },
      });
      return;
    }
    if (!signedIn()) return;
    const gen = generation;
    set({ subscription: { ...snapshot.subscription, status: "loading", error: null } });
    const entitlements = loadEntitlements(gen);
    try {
      const res = await ctx.request<{ subscription: SubscriptionJSON | null; can_manage: boolean }>(
        "GET",
        "/v1/billing/subscription",
      );
      if (gen !== generation) return;
      set({
        subscription: {
          status: "ready",
          data: res.subscription ? toSubscription(res.subscription) : null,
          error: null,
        },
        canManage: res.can_manage,
      });
    } catch (err) {
      if (gen !== generation) return;
      set({ subscription: { ...snapshot.subscription, status: "error", error: asError(err) } });
    }
    await entitlements;
  };

  const loadPlans = async () => {
    if (unavailable()) {
      set({ plans: { status: "unavailable", data: [], error: null } });
      return;
    }
    // /plans is app-wide and wants the app's ID: the client config carries
    // it, and so does the signed-in user.
    const state = ctx.getState();
    const appId =
      ctx.getClientConfig()?.app_id ??
      (state.status === "authenticated" ? state.user.app_id : undefined);
    set({ plans: { ...snapshot.plans, status: "loading", error: null } });
    try {
      const res = await ctx.request<{ plans: PlanJSON[] }>("GET", "/v1/billing/plans", {
        query: { app_id: appId },
      });
      set({ plans: { status: "ready", data: (res.plans ?? []).map(toPlan), error: null } });
    } catch (err) {
      set({ plans: { ...snapshot.plans, status: "error", error: asError(err) } });
    }
  };

  // Writes answer with the new subscription. It goes straight into the
  // cache, and the entitlements are rechecked since the plan may have moved.
  const commit = (json: SubscriptionJSON | undefined): Subscription | null => {
    if (!json) return null;
    const sub = toSubscription(json);
    set({ subscription: { status: "ready", data: sub, error: null } });
    void loadEntitlements(generation);
    return sub;
  };

  const change = async (
    method: "POST" | "PATCH",
    body: Record<string, unknown>,
  ): Promise<ChangeResult> => {
    const res = await ctx.request<{ subscription?: SubscriptionJSON; checkout_url?: string }>(
      method,
      "/v1/billing/subscription",
      { body },
    );
    if (res.checkout_url) return { type: "checkout", url: res.checkout_url };
    const sub = commit(res.subscription);
    if (!sub) throw new Error("authsome: the server answered with neither a subscription nor a checkout URL");
    return { type: "changed", subscription: sub };
  };

  const changeBody = (options: ChangeOptions | undefined, extra: Record<string, unknown>) => ({
    ...extra,
    ...(options?.seats !== undefined ? { seats: options.seats } : {}),
    ...(options?.returnUrl ? { return_url: options.returnUrl } : {}),
    ...(options?.cancelUrl ? { cancel_url: options.cancelUrl } : {}),
  });

  const mutate = async (path: string, body: unknown): Promise<Subscription> => {
    const res = await ctx.request<{ subscription?: SubscriptionJSON }>("POST", path, { body });
    const sub = commit(res.subscription);
    if (!sub) throw new Error("authsome: the server answered without a subscription");
    return sub;
  };

  // Follow the session: a sign-out forgets everything, and a sign-in or an
  // organization switch reloads whatever had been loaded.
  ctx.on("signedOut", () => {
    generation++;
    snapshot = initialSnapshot();
    set({});
  });
  const reload = () => {
    generation++;
    if (snapshot.subscription.status !== "idle") void load();
  };
  ctx.on("signedIn", reload);
  ctx.on("orgChanged", reload);

  return {
    getSnapshot: () => snapshot,
    onChange(listener) {
      listeners.add(listener);
      return () => {
        listeners.delete(listener);
      };
    },
    load,
    loadPlans,
    subscribe: (plan, options) => change("POST", changeBody(options, { plan })),
    changePlan: (plan, options) => change("PATCH", changeBody(options, { plan })),
    setSeats: (seats, options) => change("PATCH", changeBody({ ...options, seats }, {})),
    cancel: (options) => mutate("/v1/billing/subscription/cancel", { when: options?.when ?? "period_end" }),
    resume: () => mutate("/v1/billing/subscription/resume", {}),
    async openPortal(returnUrl) {
      const res = await ctx.request<{ url: string }>("POST", "/v1/billing/portal", {
        body: returnUrl ? { return_url: returnUrl } : {},
      });
      return res.url;
    },
    entitlement: (feature) => snapshot.entitlements.data[feature] ?? null,
  };
}

/**
 * The subscription client plugin. Register it with `plugins: [subscription()]`
 * and read it with `manager.plugin(subscription)` or, in React,
 * `useSubscription()`.
 */
export const subscription = definePlugin("subscription", (ctx) => createSubscriptionApi(ctx));
