import { definePlugin } from "@authsome/ui-core";
import type { AuthPlugin } from "@authsome/ui-core";
import { act, render, screen, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";

import { Protect } from "./components";
import { AuthProvider } from "./context";
import { useActiveOrg, usePlugin } from "./hooks";

const BASE = "https://api.example.test";

function json(body: unknown) {
  return new Response(JSON.stringify(body), { status: 200, headers: { "Content-Type": "application/json" } });
}

function harness(opts: { me?: Record<string, unknown>; plugins?: AuthPlugin<unknown>[] } = {}) {
  const store = new Map<string, string>([[
    "authsome:session",
    JSON.stringify({ session_token: "tok", refresh_token: "refresh", expires_at: new Date(Date.now() + 3_600_000).toISOString() }),
  ]]);
  const fetchFn = (async (url: string | URL, init?: RequestInit) => {
    const path = new URL(String(url)).pathname;
    if (path === "/v1/me") return json({ id: "usr_1", email: "a@test", roles: ["member"], ...opts.me });
    if (path === "/v1/me/switch-org") {
      const { org_id } = JSON.parse(String(init?.body));
      return json({ session_id: "ses_1", org_id });
    }
    return new Response("{}", { status: 404 });
  }) as unknown as typeof fetch;

  const wrap = (children: ReactNode) => (
    <AuthProvider baseURL={BASE} fetch={fetchFn} plugins={opts.plugins} storage={{
      getItem: (k) => store.get(k) ?? null,
      setItem: (k, v) => void store.set(k, v),
      removeItem: (k) => void store.delete(k),
    }}>
      {children}
    </AuthProvider>
  );

  return {
    async mount(node: ReactNode) {
      let out!: ReturnType<typeof render>;
      await act(async () => { out = render(wrap(node)); });
      return out;
    },
  };
}

const greeter = definePlugin("greeter", () => ({ hello: () => "hi" }));

describe("usePlugin", () => {
  it("returns the registered plugin's API", async () => {
    let seen: ReturnType<typeof usePlugin<{ hello: () => string }>> | null = null;
    function Probe() {
      seen = usePlugin(greeter);
      return null;
    }
    await harness({ plugins: [greeter()] }).mount(<Probe />);
    expect(seen!.hello()).toBe("hi");
  });

  it("throws a helpful error when the plugin isn't registered", async () => {
    const spy = vi.spyOn(console, "error").mockImplementation(() => {});
    function Probe() {
      usePlugin(greeter);
      return null;
    }
    await expect(harness().mount(<Probe />)).rejects.toThrow(/greeter\(\)/);
    spy.mockRestore();
  });
});

describe("useActiveOrg", () => {
  it("reports the restored org and re-renders on a switch", async () => {
    let seen = null as unknown as ReturnType<typeof useActiveOrg>;
    function Probe() {
      seen = useActiveOrg();
      return <span data-testid="org">{seen.orgId ?? "none"}</span>;
    }
    await harness({ me: { active_org_id: "org_1" } }).mount(<Probe />);
    await waitFor(() => expect(screen.getByTestId("org").textContent).toBe("org_1"));

    await act(async () => { await seen.switchOrg("org_2"); });
    expect(screen.getByTestId("org").textContent).toBe("org_2");
    expect(seen.isSwitching).toBe(false);
  });
});

describe("Protect", () => {
  it("shows children only to holders of the role", async () => {
    const h = harness({ me: { roles: ["admin"] } });
    await h.mount(
      <>
        <Protect role="admin" fallback={<span>no-admin</span>}><span>admin-panel</span></Protect>
        <Protect role="owner" fallback={<span>no-owner</span>}><span>owner-panel</span></Protect>
      </>,
    );
    await waitFor(() => expect(screen.queryByText("admin-panel")).not.toBeNull());
    expect(screen.queryByText("owner-panel")).toBeNull();
    expect(screen.queryByText("no-owner")).not.toBeNull();
  });

  it("fails closed for a permission it can't check", async () => {
    const warn = vi.spyOn(console, "warn").mockImplementation(() => {});
    await harness({ me: { roles: ["admin"] } }).mount(
      <Protect permission="billing:manage" fallback={<span>hidden</span>}><span>secret</span></Protect>,
    );
    await waitFor(() => expect(screen.queryByText("hidden")).not.toBeNull());
    expect(screen.queryByText("secret")).toBeNull();
    expect(warn).toHaveBeenCalledWith(expect.stringMatching(/permission/));
    warn.mockRestore();
  });

  it("lets a condition decide a permission", async () => {
    await harness().mount(
      <Protect permission="billing:manage" condition={() => true} fallback={<span>hidden</span>}><span>shown</span></Protect>,
    );
    await waitFor(() => expect(screen.queryByText("shown")).not.toBeNull());
  });
});
