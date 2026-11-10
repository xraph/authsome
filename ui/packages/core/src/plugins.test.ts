import { afterEach, describe, expect, it, vi } from "vitest";
import { AuthManager, SESSION_STORAGE_KEY } from "./auth";
import { definePlugin, PluginNotRegisteredError, type PluginContext } from "./plugins";
import type { AuthConfig, ClientConfig, Session, User } from "./types";

const user = { id: "usr_1", email: "ada@test" } as User;
const managers: AuthManager[] = [];
afterEach(() => managers.splice(0).forEach((m) => m.destroy()));
const json = (body: unknown, status = 200) => new Response(JSON.stringify(body), { status });

function setup(opts: { authenticated?: boolean; me?: Record<string, unknown>; config?: Partial<AuthConfig> } = {}) {
  const session: Session = {
    session_token: "token-1", refresh_token: "refresh-1",
    expires_at: new Date(Date.now() + 3_600_000).toISOString(),
  };
  const store = new Map<string, string>();
  if (opts.authenticated) store.set(SESSION_STORAGE_KEY, JSON.stringify(session));
  const routes = new Map<string, (init?: RequestInit) => Response | Promise<Response>>([
    ["GET /v1/me", () => json({ ...user, ...opts.me })],
    ["POST /v1/signout", () => json({ status: "ok" })],
  ]);
  const fetch = vi.fn<typeof globalThis.fetch>(async (input, init) => {
    const url = new URL(String(input));
    const handler = routes.get(`${init?.method ?? "GET"} ${url.pathname}`);
    if (!handler) throw new Error(`Unexpected route: ${init?.method} ${url.pathname}`);
    return handler(init);
  });
  const manager = new AuthManager({
    baseURL: "https://api.test", fetch,
    storage: {
      getItem: (k) => store.get(k) ?? null,
      setItem: (k, v) => { store.set(k, v); },
      removeItem: (k) => { store.delete(k); },
    },
    ...opts.config,
  });
  managers.push(manager);
  return { manager, routes, fetch, store, session };
}

describe("plugins", () => {
  const counter = definePlugin("counter", (ctx: PluginContext) => ({ ctx, value: 42 }));

  it("returns what init returned, built once and lazily", async () => {
    const init = vi.fn(() => ({ n: 1 }));
    const lazy = definePlugin("lazy", init);
    const { manager } = setup({ config: { plugins: [lazy()] } });
    expect(init).not.toHaveBeenCalled();
    const first = manager.plugin(lazy);
    expect(first).toEqual({ n: 1 });
    expect(manager.plugin(lazy)).toBe(first);
    expect(init).toHaveBeenCalledTimes(1);
  });

  it("finds a plugin by its definition or its instance", () => {
    const instance = counter();
    const { manager } = setup({ config: { plugins: [instance] } });
    expect(manager.plugin(counter).value).toBe(42);
    expect(manager.plugin(instance).value).toBe(42);
    expect(manager.hasPlugin(counter)).toBe(true);
  });

  it("throws a named error for a plugin that wasn't registered", () => {
    const { manager } = setup();
    expect(manager.hasPlugin(counter)).toBe(false);
    expect(() => manager.plugin(counter)).toThrow(PluginNotRegisteredError);
    expect(() => manager.plugin(counter)).toThrow(/add counter\(\) to the plugins/i);
  });

  it("refuses the same plugin twice", () => {
    expect(() => setup({ config: { plugins: [counter(), counter()] } })).toThrow(/registered twice/);
  });

  it("surfaces an init error on first use, not at construction", () => {
    const broken = definePlugin("broken", () => {
      throw new Error("bad config");
    });
    const { manager } = setup({ config: { plugins: [broken()] } });
    expect(() => manager.plugin(broken)).toThrow("bad config");
  });

  it("passes options from the registration to init", () => {
    const withOpts = definePlugin("opts", (_ctx, options?: { greeting: string }) => options?.greeting);
    const { manager } = setup({ config: { plugins: [withOpts({ greeting: "hi" })] } });
    expect(manager.plugin(withOpts)).toBe("hi");
  });

  it("reports whether the server has the plugin", () => {
    const config = { supported_plugins: ["counter"] } as unknown as ClientConfig;
    const unknown = setup({ config: { plugins: [counter()] } }).manager.plugin(counter).ctx;
    expect(unknown.isServerEnabled()).toBeNull();

    const known = setup({ config: { plugins: [counter()], initialClientConfig: config } }).manager.plugin(counter).ctx;
    expect(known.isServerEnabled()).toBe(true);

    const other = definePlugin("other", (ctx) => ctx);
    const missing = setup({ config: { plugins: [other()], initialClientConfig: config } }).manager.plugin(other);
    expect(missing.isServerEnabled()).toBe(false);
  });

  it("sends authenticated requests with the session token and query", async () => {
    const { manager, routes, fetch } = setup({ authenticated: true, config: { plugins: [counter()], publishableKey: "pk_test" } });
    routes.set("GET /v1/billing/subscription", () => json({ subscription: null }));
    await manager.initialize();
    const ctx = manager.plugin(counter).ctx;

    const body = await ctx.request<{ subscription: null }>("GET", "/v1/billing/subscription", { query: { tenant_id: "org_1", skip: undefined } });
    expect(body).toEqual({ subscription: null });
    const [url, init] = fetch.mock.calls.at(-1)!;
    expect(String(url)).toBe("https://api.test/v1/billing/subscription?tenant_id=org_1");
    expect((init?.headers as Record<string, string>).Authorization).toBe("Bearer token-1");
    expect((init?.headers as Record<string, string>)["X-Publishable-Key"]).toBe("pk_test");
  });

  it("sends credentials instead of a token in cookie mode", async () => {
    const { manager, routes, fetch } = setup({ config: { plugins: [counter()], session: "cookie" } });
    routes.set("POST /v1/billing/subscription", () => json({ ok: true }));
    await manager.plugin(counter).ctx.request("POST", "/v1/billing/subscription", { body: { plan: "pro" } });
    const [, init] = fetch.mock.calls.at(-1)!;
    expect(init?.credentials).toBe("include");
    expect((init?.headers as Record<string, string>).Authorization).toBeUndefined();
    expect(init?.body).toBe(JSON.stringify({ plan: "pro" }));
  });

  it("throws AuthClientError with the server's message", async () => {
    const { manager, routes } = setup({ config: { plugins: [counter()] } });
    routes.set("GET /v1/thing", () => json({ error: "nope", type: "forbidden" }, 403));
    await expect(manager.plugin(counter).ctx.request("GET", "/v1/thing")).rejects.toMatchObject({
      message: "nope", code: 403, type: "forbidden",
    });
  });
});

describe("lifecycle events", () => {
  it("fires signedIn, sessionRefreshed and signedOut once each", async () => {
    const { manager, routes } = setup({ authenticated: true });
    const seen: string[] = [];
    for (const e of ["signedIn", "signedOut", "sessionRefreshed", "orgChanged"] as const) {
      manager.on(e, () => seen.push(e));
    }
    await manager.initialize();
    expect(seen).toEqual(["signedIn"]);

    routes.set("POST /v1/refresh", () => json({ session_token: "token-2", refresh_token: "refresh-2", expires_at: new Date(Date.now() + 3_600_000).toISOString() }));
    await manager.refreshNow();
    expect(seen).toEqual(["signedIn", "sessionRefreshed"]);

    await manager.signOut();
    expect(seen).toEqual(["signedIn", "sessionRefreshed", "signedOut"]);
  });

  it("stops calling a listener once it unsubscribes", async () => {
    const { manager } = setup({ authenticated: true });
    const listener = vi.fn();
    const off = manager.on("signedIn", listener);
    off();
    await manager.initialize();
    expect(listener).not.toHaveBeenCalled();
  });
});

describe("active organization", () => {
  it("restores the active org from /v1/me", async () => {
    const { manager } = setup({ authenticated: true, me: { active_org_id: "org_1" } });
    expect(manager.getActiveOrgId()).toBeNull();
    await manager.initialize();
    expect(manager.getActiveOrgId()).toBe("org_1");
  });

  it("switches org, keeps a reissued token and announces it", async () => {
    const { manager, routes, store } = setup({ authenticated: true });
    await manager.initialize();
    const orgChanged = vi.fn();
    manager.on("orgChanged", orgChanged);
    routes.set("POST /v1/me/switch-org", (init) => {
      expect(JSON.parse(String(init?.body))).toEqual({ org_id: "org_2" });
      expect((init?.headers as Record<string, string>).Authorization).toBe("Bearer token-1");
      return json({ session_id: "ses_1", org_id: "org_2", session_token: "token-org2", expires_at: new Date(Date.now() + 3_600_000).toISOString() });
    });

    await manager.switchOrg("org_2");
    expect(manager.getActiveOrgId()).toBe("org_2");
    expect(orgChanged).toHaveBeenCalledTimes(1);
    expect(manager.getSessionToken()).toBe("token-org2");
    expect(JSON.parse(store.get(SESSION_STORAGE_KEY)!).session_token).toBe("token-org2");
  });

  it("refuses to switch when signed out", async () => {
    const { manager } = setup();
    await expect(manager.switchOrg("org_2")).rejects.toThrow(/signed in/);
  });

  it("forgets the org on sign-out and doesn't hand it to the next user", async () => {
    const { manager, routes } = setup({ authenticated: true, me: { active_org_id: "org_1" } });
    await manager.initialize();
    expect(manager.getActiveOrgId()).toBe("org_1");
    await manager.signOut();
    expect(manager.getActiveOrgId()).toBeNull();

    routes.set("POST /v1/signin", () => json({ user: { ...user, id: "usr_2" }, session_token: "t", refresh_token: "r", expires_at: new Date(Date.now() + 3_600_000).toISOString() }));
    await manager.signIn({ email: "grace@test", password: "pw" });
    expect(manager.getActiveOrgId()).toBeNull();
  });
});
