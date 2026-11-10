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
  /**
   * The tenant this API is pinned to, or null for the session's own (its
   * active organization, or the user in user mode).
   */
  readonly tenantId: string | null;
  /**
   * The same API for another tenant, with its own cache. For operators who
   * hold billing:manage and look after customers' billing; the server
   * refuses anyone else. Calling it twice with one ID returns one instance.
   */
  forTenant(tenantId: string): SubscriptionApi;
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

// State every scope of one plugin instance shares.
interface Shared {
  scopes: Map<string, SubscriptionApi>;
  warned: boolean;
}

const UNAVAILABLE_WARNING =
  "authsome: the server doesn't have the subscription plugin, so billing reads as " +
  "unavailable and every entitlement check fails closed. Register it on the server with " +
  "authsome.WithPlugin(subscription.New(...)), or drop subscription() from your client plugins.";

function isProduction(): boolean {
  const env = (globalThis as { process?: { env?: Record<string, string | undefined> } }).process?.env;
  return env?.NODE_ENV === "production";
}

/** Builds the subscription API over a plugin context. */
export function createSubscriptionApi(ctx: PluginContext): SubscriptionApi {
  return createScope(ctx, null, { scopes: new Map(), warned: false });
}

function createScope(ctx: PluginContext, tenantId: string | null, shared: Shared): SubscriptionApi {
  // A pinned scope names its tenant on every request; the session's scope
  // leaves it to the server.
  const scoped: Record<string, string> = tenantId ? { tenant_id: tenantId } : {};
  let snapshot = initialSnapshot();
  const listeners = new Set<() => void>();
  // Bumped whenever the session or organization changes. Anything that
  // started before the bump (a load, a plans fetch, a write) finds a
  // different number when it lands, and leaves the cache alone, so a
  // response for the organization or user the session just left is never
  // shown as the current one.
  let generation = 0;
  // In-flight loads, shared by every caller in the same generation, so ten
  // components mounting at once make one pair of requests.
  let loading: { gen: number; promise: Promise<void> } | null = null;
  let loadingPlans: { gen: number; promise: Promise<void> } | null = null;

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

  const unavailable = () => {
    if (ctx.isServerEnabled() !== false) return false;
    if (!shared.warned && !isProduction()) {
      shared.warned = true;
      console.warn(UNAVAILABLE_WARNING);
    }
    return true;
  };
  const signedIn = () => ctx.getState().status === "authenticated";

  const loadEntitlements = async (gen: number) => {
    set({ entitlements: { ...snapshot.entitlements, status: "loading", error: null } });
    try {
      const res = await ctx.request<{ entitlements: Record<string, EntitlementJSON> }>(
        "GET",
        "/v1/billing/entitlements",
        { query: scoped },
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

  const fetchSubscription = async (gen: number) => {
    set({ subscription: { ...snapshot.subscription, status: "loading", error: null } });
    const entitlements = loadEntitlements(gen);
    try {
      const res = await ctx.request<{ subscription: SubscriptionJSON | null; can_manage: boolean }>(
        "GET",
        "/v1/billing/subscription",
        { query: scoped },
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

  const load = (): Promise<void> => {
    if (unavailable()) {
      set({
        subscription: { status: "unavailable", data: null, error: null },
        entitlements: { status: "unavailable", data: {}, error: null },
      });
      return Promise.resolve();
    }
    if (!signedIn()) return Promise.resolve();
    if (loading && loading.gen === generation) return loading.promise;
    const gen = generation;
    const promise = fetchSubscription(gen).finally(() => {
      if (loading?.promise === promise) loading = null;
    });
    loading = { gen, promise };
    return promise;
  };

  const fetchPlans = async (gen: number) => {
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
      if (gen !== generation) return;
      set({ plans: { status: "ready", data: (res.plans ?? []).map(toPlan), error: null } });
    } catch (err) {
      if (gen !== generation) return;
      set({ plans: { ...snapshot.plans, status: "error", error: asError(err) } });
    }
  };

  const loadPlans = (): Promise<void> => {
    if (unavailable()) {
      set({ plans: { status: "unavailable", data: [], error: null } });
      return Promise.resolve();
    }
    if (loadingPlans && loadingPlans.gen === generation) return loadingPlans.promise;
    const gen = generation;
    const promise = fetchPlans(gen).finally(() => {
      if (loadingPlans?.promise === promise) loadingPlans = null;
    });
    loadingPlans = { gen, promise };
    return promise;
  };

  // Writes answer with the new subscription. It goes into the cache, and the
  // entitlements are rechecked since the plan may have moved, unless the
  // session changed while the write was out: then the caller still gets the
  // result, but the cache now belongs to someone else.
  const commit = (json: SubscriptionJSON | undefined, gen: number): Subscription | null => {
    if (!json) return null;
    const sub = toSubscription(json);
    if (gen === generation) {
      set({ subscription: { status: "ready", data: sub, error: null } });
      void loadEntitlements(gen);
    }
    return sub;
  };

  const change = async (
    method: "POST" | "PATCH",
    body: Record<string, unknown>,
  ): Promise<ChangeResult> => {
    const gen = generation;
    const res = await ctx.request<{ subscription?: SubscriptionJSON; checkout_url?: string }>(
      method,
      "/v1/billing/subscription",
      { body: { ...body, ...scoped } },
    );
    if (res.checkout_url) return { type: "checkout", url: res.checkout_url };
    const sub = commit(res.subscription, gen);
    if (!sub) throw new Error("authsome: the server answered with neither a subscription nor a checkout URL");
    return { type: "changed", subscription: sub };
  };

  const changeBody = (options: ChangeOptions | undefined, extra: Record<string, unknown>) => ({
    ...extra,
    ...(options?.seats !== undefined ? { seats: options.seats } : {}),
    ...(options?.returnUrl ? { return_url: options.returnUrl } : {}),
    ...(options?.cancelUrl ? { cancel_url: options.cancelUrl } : {}),
  });

  const mutate = async (path: string, body: Record<string, unknown>): Promise<Subscription> => {
    const gen = generation;
    const res = await ctx.request<{ subscription?: SubscriptionJSON }>("POST", path, {
      body: { ...body, ...scoped },
    });
    const sub = commit(res.subscription, gen);
    if (!sub) throw new Error("authsome: the server answered without a subscription");
    return sub;
  };

  // Follow the session: a sign-out forgets everything; a sign-in or an
  // organization switch drops the previous tenant's subscription at once
  // (so no screen shows it, or offers buttons for it, while the new one
  // loads) and reloads whatever had been loaded. A pinned tenant doesn't
  // change with the organization, but can_manage does change with the user.
  ctx.on("signedOut", () => {
    generation++;
    snapshot = initialSnapshot();
    set({});
  });
  const reload = () => {
    generation++;
    if (snapshot.subscription.status === "idle") return;
    set({
      subscription: { status: "loading", data: null, error: null },
      entitlements: { status: "loading", data: {}, error: null },
      canManage: false,
    });
    void load();
  };
  ctx.on("signedIn", reload);
  if (!tenantId) ctx.on("orgChanged", reload);

  return {
    tenantId,
    forTenant(id) {
      let scope = shared.scopes.get(id);
      if (!scope) {
        scope = createScope(ctx, id, shared);
        shared.scopes.set(id, scope);
      }
      return scope;
    },
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
        body: { ...(returnUrl ? { return_url: returnUrl } : {}), ...scoped },
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
