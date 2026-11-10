import type { ClientConfig } from "@authsome/ui-core";
import { AuthProvider } from "@authsome/ui-react";
import { act, render } from "@testing-library/react";
import type { ReactNode } from "react";

import { subscription } from "./plugin";

export const json = (body: unknown, status = 200) =>
  new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } });

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

export type Handler = (init?: RequestInit) => Response | Promise<Response>;

/**
 * A signed-in AuthProvider with the subscription plugin, over a routed
 * fetch. Routes default to an active "pro" subscription the user can manage.
 */
export function harness(opts: { serverPlugins?: string[] } = {}) {
  const store = new Map<string, string>([[
    "authsome:session",
    JSON.stringify({ session_token: "tok", refresh_token: "ref", expires_at: new Date(Date.now() + 3_600_000).toISOString() }),
  ]]);
  const routes = new Map<string, Handler>([
    ["GET /v1/me", () => json({ id: "usr_1", email: "a@test", app_id: "app_1", active_org_id: "org_1" })],
    ["GET /v1/billing/subscription", () => json({ subscription: subJSON(), can_manage: true })],
    ["GET /v1/billing/entitlements", () => json({ entitlements: { sso: { allowed: true, feature: "sso", used: 2, limit: 10, remaining: 8 } } })],
    ["GET /v1/billing/plans", () => json({ plans: [planJSON("free", { amount_minor: undefined, base_amount: undefined }), planJSON("pro"), planJSON("team", { amount_minor: 9900, base_amount: "99.00" })] })],
  ]);
  const calls: string[] = [];
  // The same calls with their query strings, for tests that care.
  const urls: string[] = [];
  const fetchFn = (async (input: string | URL, init?: RequestInit) => {
    const url = new URL(String(input));
    const key = `${init?.method ?? "GET"} ${url.pathname}`;
    calls.push(key);
    urls.push(`${key}${url.search}`);
    const handler = routes.get(key);
    return handler ? handler(init) : json({ error: `no route ${key}` }, 404);
  }) as unknown as typeof fetch;
  const initialClientConfig = opts.serverPlugins
    ? ({ supported_plugins: opts.serverPlugins } as unknown as ClientConfig)
    : undefined;

  const wrap = (children: ReactNode) => (
    <AuthProvider
      baseURL="https://api.test"
      fetch={fetchFn}
      plugins={[subscription()]}
      initialClientConfig={initialClientConfig}
      storage={{
        getItem: (k) => store.get(k) ?? null,
        setItem: (k, v) => void store.set(k, v),
        removeItem: (k) => void store.delete(k),
      }}
    >
      {children}
    </AuthProvider>
  );

  return {
    routes,
    calls,
    urls,
    async mount(node: ReactNode): Promise<void> {
      await act(async () => {
        render(wrap(node));
      });
    },
  };
}
