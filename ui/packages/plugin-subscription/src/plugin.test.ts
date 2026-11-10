import { AuthManager, type ClientConfig, type Session } from "@authsome/ui-core";
import { afterEach, describe, expect, it, vi } from "vitest";

import { formatMoney, subscription, type Subscription } from "./index";

const managers: AuthManager[] = [];
afterEach(() => managers.splice(0).forEach((m) => m.destroy()));
const json = (body: unknown, status = 200) => new Response(JSON.stringify(body), { status });

const inAnHour = () => new Date(Date.now() + 3_600_000).toISOString();

export function planJSON(slug: string, overrides: Record<string, unknown> = {}) {
  return {
    id: `plan_${slug}`, name: slug[0].toUpperCase() + slug.slice(1), slug, description: "",
    currency: "usd", status: "active", trial_days: 0, features_count: 1, is_addon: false, app_id: "app_1",
    base_amount: "29.00", billing_period: "monthly", amount_minor: 2900,
    features: [{ key: "sso", name: "SSO", type: "boolean", limit: 1, soft_limit: false }],
    ...overrides,
  };
}

export function subJSON(overrides: Record<string, unknown> = {}) {
  return {
    id: "sub_1", tenant_id: "org_1", status: "active", plan: planJSON("pro"),
    current_period_start: "2026-11-01T00:00:00Z", current_period_end: "2026-12-01T00:00:00Z",
    trial_start: null, trial_end: null, cancel_at: null, canceled_at: null, paused_at: null,
    app_id: "app_1",
    ...overrides,
  };
}

type Handler = (init?: RequestInit, url?: URL) => Response | Promise<Response>;

function setup(opts: { serverPlugins?: string[]; me?: Record<string, unknown>; appId?: string } = {}) {
  const session: Session = { session_token: "tok", refresh_token: "ref", expires_at: inAnHour() };
  const store = new Map<string, string>([["authsome:session", JSON.stringify(session)]]);
  const routes = new Map<string, Handler>([
    ["GET /v1/me", () => json({ id: "usr_1", email: "a@test", app_id: "app_user", active_org_id: "org_1", ...opts.me })],
    ["POST /v1/signout", () => json({ status: "ok" })],
    ["GET /v1/billing/subscription", () => json({ subscription: subJSON(), can_manage: true })],
    ["GET /v1/billing/entitlements", () => json({ entitlements: { sso: { allowed: true, feature: "sso", used: 0, limit: 1, remaining: 1 } } })],
  ]);
  const calls: string[] = [];
  const fetch = vi.fn<typeof globalThis.fetch>(async (input, init) => {
    const url = new URL(String(input));
    const key = `${init?.method ?? "GET"} ${url.pathname}`;
    calls.push(`${key}${url.search}`);
    const handler = routes.get(key);
    if (!handler) return json({ error: `no route ${key}` }, 404);
    return handler(init, url);
  });
  const initialClientConfig = opts.serverPlugins
    ? ({ supported_plugins: opts.serverPlugins, app_id: opts.appId } as unknown as ClientConfig)
    : opts.appId ? ({ app_id: opts.appId } as unknown as ClientConfig) : undefined;
  const manager = new AuthManager({
    baseURL: "https://api.test", fetch, initialClientConfig,
    plugins: [subscription()],
    storage: {
      getItem: (k) => store.get(k) ?? null,
      setItem: (k, v) => { store.set(k, v); },
      removeItem: (k) => { store.delete(k); },
    },
  });
  managers.push(manager);
  return { manager, routes, calls, api: () => manager.plugin(subscription) };
}

describe("models", () => {
  it("maps the server's view into typed, camelCase models", async () => {
    const { manager, routes, api } = setup();
    routes.set("GET /v1/billing/subscription", () => json({
      can_manage: false,
      subscription: subJSON({
        status: "trialing", trial_end: new Date(Date.now() + 3 * 86_400_000).toISOString(),
        cancel_at: "2026-12-01T00:00:00Z",
        seats: { feature_key: "seats", used: 3, limit: null },
        plan: planJSON("team", { features: [{ key: "api", name: "API", type: "metered", limit: null, soft_limit: true, period: "monthly" }] }),
      }),
    }));
    await manager.initialize();
    await api().load();

    const snap = api().getSnapshot();
    const sub = snap.subscription.data as Subscription;
    expect(snap.subscription.status).toBe("ready");
    expect(snap.canManage).toBe(false);
    expect(sub.plan.slug).toBe("team");
    expect(sub.plan.price).toEqual({ amountMinor: 2900, currency: "usd", interval: "monthly" });
    expect(sub.plan.features[0]).toEqual({ key: "api", name: "API", type: "metered", limit: null, period: "monthly", softLimit: true });
    expect(sub.seats).toEqual({ featureKey: "seats", used: 3, limit: null });
    expect(sub.isTrialing).toBe(true);
    expect(sub.isActive).toBe(true);
    expect(sub.willCancel).toBe(true);
    expect(sub.cancelAt).toBe("2026-12-01T00:00:00Z");
    expect(sub.trialDaysLeft).toBe(3);
  });

  it("formats money in minor units by the currency's digits", () => {
    expect(formatMoney({ amountMinor: 2900, currency: "usd" }, "en-US")).toBe("$29.00");
    expect(formatMoney({ amountMinor: 500, currency: "jpy" }, "en-US")).toBe("¥500");
  });
});

describe("loading", () => {
  it("loads the subscription, can_manage and entitlements together", async () => {
    const { manager, api } = setup();
    await manager.initialize();
    await api().load();
    const snap = api().getSnapshot();
    expect(snap.subscription.data?.id).toBe("sub_1");
    expect(snap.canManage).toBe(true);
    expect(snap.entitlements.status).toBe("ready");
    expect(api().entitlement("sso")?.allowed).toBe(true);
    expect(api().entitlement("nope")).toBeNull();
  });

  it("is unavailable, without asking, when the server has no subscription plugin", async () => {
    const { manager, api, calls } = setup({ serverPlugins: ["organization"] });
    await manager.initialize();
    await api().load();
    expect(api().getSnapshot().subscription.status).toBe("unavailable");
    expect(api().getSnapshot().entitlements.status).toBe("unavailable");
    expect(calls.some((c) => c.includes("/v1/billing"))).toBe(false);
  });

  it("drops a response for the org the session has since left", async () => {
    const { manager, routes, api } = setup();
    await manager.initialize();
    let release!: () => void;
    const gate = new Promise<void>((r) => (release = r));
    routes.set("GET /v1/billing/subscription", async () => {
      await gate;
      return json({ subscription: subJSON({ tenant_id: "org_1" }), can_manage: true });
    });
    const loading = api().load();

    routes.set("POST /v1/me/switch-org", () => json({ session_id: "s", org_id: "org_2" }));
    routes.set("GET /v1/billing/subscription", () => json({ subscription: null, can_manage: true }));
    await manager.switchOrg("org_2");
    release();
    await loading;
    await vi.waitFor(() => expect(api().getSnapshot().subscription.status).toBe("ready"));
    expect(api().getSnapshot().subscription.data).toBeNull();
  });

  it("resets on sign-out and reloads on an org switch", async () => {
    const { manager, routes, api, calls } = setup();
    await manager.initialize();
    await api().load();
    routes.set("POST /v1/me/switch-org", () => json({ session_id: "s", org_id: "org_2" }));
    const before = calls.filter((c) => c.startsWith("GET /v1/billing/subscription")).length;
    await manager.switchOrg("org_2");
    await vi.waitFor(() =>
      expect(calls.filter((c) => c.startsWith("GET /v1/billing/subscription")).length).toBe(before + 1));

    await manager.signOut();
    expect(api().getSnapshot().subscription.status).toBe("idle");
    expect(api().getSnapshot().subscription.data).toBeNull();
  });

  it("reads plans for the client config's app, else the user's", async () => {
    const withConfig = setup({ appId: "app_cfg" });
    withConfig.routes.set("GET /v1/billing/plans", () => json({ plans: [planJSON("free", { amount_minor: undefined, base_amount: undefined, billing_period: undefined }), planJSON("pro")], total: 2 }));
    await withConfig.manager.initialize();
    await withConfig.api().loadPlans();
    expect(withConfig.calls).toContain("GET /v1/billing/plans?app_id=app_cfg");
    expect(withConfig.api().getSnapshot().plans.data.map((p) => p.slug)).toEqual(["free", "pro"]);
    expect(withConfig.api().getSnapshot().plans.data[0].price).toBeNull();

    const fromUser = setup();
    fromUser.routes.set("GET /v1/billing/plans", () => json({ plans: [], total: 0 }));
    await fromUser.manager.initialize();
    await fromUser.api().loadPlans();
    expect(fromUser.calls).toContain("GET /v1/billing/plans?app_id=app_user");
  });
});

describe("writes", () => {
  it("writes a changed subscription into the cache", async () => {
    const { manager, routes, api } = setup();
    await manager.initialize();
    await api().load();
    routes.set("PATCH /v1/billing/subscription", (init) => {
      expect(JSON.parse(String(init?.body))).toEqual({ plan: "team", seats: 5 });
      return json({ subscription: subJSON({ plan: planJSON("team") }) });
    });

    const result = await api().changePlan("team", { seats: 5 });
    expect(result.type).toBe("changed");
    expect(api().getSnapshot().subscription.data?.plan.slug).toBe("team");
  });

  it("leaves the cache alone when payment comes first", async () => {
    const { manager, routes, api } = setup();
    await manager.initialize();
    await api().load();
    routes.set("POST /v1/billing/subscription", () => json({ checkout_url: "https://pay.example/1" }));

    const result = await api().subscribe("business", { returnUrl: "https://app/billing" });
    expect(result).toEqual({ type: "checkout", url: "https://pay.example/1" });
    expect(api().getSnapshot().subscription.data?.plan.slug).toBe("pro");
  });

  it("surfaces a refused write without touching the cache", async () => {
    const { manager, routes, api } = setup();
    await manager.initialize();
    await api().load();
    routes.set("POST /v1/billing/subscription/cancel", () => json({ error: "your role can't manage billing for this organization" }, 403));

    await expect(api().cancel()).rejects.toMatchObject({ code: 403 });
    expect(api().getSnapshot().subscription.data?.cancelAt).toBeNull();
  });

  it("cancels at the period end by default and resumes", async () => {
    const { manager, routes, api } = setup();
    await manager.initialize();
    await api().load();
    routes.set("POST /v1/billing/subscription/cancel", (init) => {
      expect(JSON.parse(String(init?.body))).toEqual({ when: "period_end" });
      return json({ subscription: subJSON({ cancel_at: "2026-12-01T00:00:00Z" }) });
    });
    routes.set("POST /v1/billing/subscription/resume", () => json({ subscription: subJSON() }));

    expect((await api().cancel()).willCancel).toBe(true);
    expect(api().getSnapshot().subscription.data?.willCancel).toBe(true);
    expect((await api().resume()).willCancel).toBe(false);
  });

  it("notifies listeners on every change", async () => {
    const { manager, api } = setup();
    await manager.initialize();
    const listener = vi.fn();
    api().onChange(listener);
    await api().load();
    expect(listener).toHaveBeenCalled();
  });
});
