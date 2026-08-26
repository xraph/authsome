import { act, render, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, it } from "vitest";

import { AuthProvider, useAuth } from "./context";
import { useOrganizations, useUser } from "./hooks";

// AuthConfig takes both fetch and storage, and AuthProvider spreads its props
// straight into the manager, so a test can drive the whole stack through the
// public surface without touching globals.

const BASE = "https://api.example.test";

const liveSession = {
  session_token: "tok",
  refresh_token: "refresh",
  expires_at: new Date(Date.now() + 3_600_000).toISOString(),
};

function makeUser(firstName: string) {
  return {
    id: "usr_1",
    app_id: "app_1",
    env_id: "env_1",
    email: "a@test",
    email_verified: true,
    first_name: firstName,
    last_name: "Lovelace",
    banned: false,
    phone_verified: false,
    created_at: new Date(0).toISOString(),
    updated_at: new Date(0).toISOString(),
  };
}

function json(body: unknown) {
  return new Response(JSON.stringify(body), {
    status: 200,
    headers: { "Content-Type": "application/json" },
  });
}

/**
 * A provider wired to a stored session and a routed fetch, so the manager
 * hydrates to authenticated. `calls` records paths so a test can prove a fetch
 * happened (or did not).
 */
function harness(opts: { signedIn?: boolean } = {}) {
  const signedIn = opts.signedIn ?? true;
  const store = new Map<string, string>();
  if (signedIn) store.set("authsome:session", JSON.stringify(liveSession));

  const calls: string[] = [];
  let meFirstName = "Ada";
  let orgs = [{ id: "org_1", name: "Acme" }];

  const fetchFn = (async (url: string | URL) => {
    const path = new URL(String(url)).pathname;
    calls.push(path);
    if (path === "/v1/me") return json(makeUser(meFirstName));
    if (path === "/v1/orgs") return json({ items: orgs, total: orgs.length });
    return new Response("{}", { status: 404 });
  }) as unknown as typeof fetch;

  const wrap = (children: ReactNode) => (
    <AuthProvider baseURL={BASE} storage={{
      getItem: (k) => store.get(k) ?? null,
      setItem: (k, v) => void store.set(k, v),
      removeItem: (k) => void store.delete(k),
    }} fetch={fetchFn}>
      {children}
    </AuthProvider>
  );

  return {
    calls,
    setMeFirstName: (n: string) => { meFirstName = n; },
    setOrgs: (o: typeof orgs) => { orgs = o; },
    async mount(Probe: () => ReactNode) {
      let out!: ReturnType<typeof render>;
      await act(async () => { out = render(wrap(<Probe />)); });
      return out;
    },
  };
}

describe("useUser", () => {
  it("reports the user the context is holding", async () => {
    const h = harness();
    let seen = null as unknown as ReturnType<typeof useUser>;
    await h.mount(function Probe() {
      seen = useUser();
      return null;
    });
    await waitFor(() => expect(seen.user?.first_name).toBe("Ada"));
  });

  it("replaces the user with the freshly fetched profile on reload", async () => {
    const h = harness();
    let seen = null as unknown as ReturnType<typeof useUser>;
    await h.mount(function Probe() {
      seen = useUser();
      return null;
    });
    await waitFor(() => expect(seen.user?.first_name).toBe("Ada"));

    h.setMeFirstName("Grace");
    await act(async () => { await seen.reload(); });

    expect(seen.user?.first_name).toBe("Grace");
  });

  // reload() puts a fetched profile in front of the context user. That
  // override has to be dropped the moment the context user itself changes, or
  // a signed-out app keeps rendering the previous person's profile.
  it("drops the reloaded profile when the context user changes", async () => {
    const h = harness();
    let seen = null as unknown as ReturnType<typeof useUser>;
    let auth = null as unknown as ReturnType<typeof useAuth>;
    await h.mount(function Probe() {
      seen = useUser();
      auth = useAuth();
      return null;
    });
    await waitFor(() => expect(seen.user?.first_name).toBe("Ada"));

    h.setMeFirstName("Grace");
    await act(async () => { await seen.reload(); });
    expect(seen.user?.first_name).toBe("Grace");

    await act(async () => { await auth.signOut(); });

    expect(seen.user).toBeNull();
  });

  it("does not fetch on reload when there is no session", async () => {
    const h = harness({ signedIn: false });
    let seen = null as unknown as ReturnType<typeof useUser>;
    await h.mount(function Probe() {
      seen = useUser();
      return null;
    });
    const before = h.calls.length;
    await act(async () => { await seen.reload(); });
    expect(h.calls.length).toBe(before);
    expect(seen.user).toBeNull();
  });
});

describe("useOrganizations", () => {
  it("loads the list once the session is authenticated", async () => {
    const h = harness();
    let seen = null as unknown as ReturnType<typeof useOrganizations>;
    await h.mount(function Probe() {
      seen = useOrganizations();
      return null;
    });
    await waitFor(() => expect(seen.organizations).toHaveLength(1));
    expect(seen.organizations[0].name).toBe("Acme");
    expect(seen.total).toBe(1);
  });

  // The flag has to settle to false, and it has to have been true at some
  // point while the request was in flight, or a spinner never shows.
  it("reports loading while the request is in flight and clears it after", async () => {
    const h = harness();
    const flags: boolean[] = [];
    let seen = null as unknown as ReturnType<typeof useOrganizations>;
    await h.mount(function Probe() {
      seen = useOrganizations();
      flags.push(seen.isLoading);
      return null;
    });
    await waitFor(() => expect(seen.organizations).toHaveLength(1));

    expect(flags).toContain(true);
    expect(seen.isLoading).toBe(false);
  });

  it("does not hit the orgs endpoint when unauthenticated", async () => {
    const h = harness({ signedIn: false });
    await h.mount(function Probe() {
      useOrganizations();
      return null;
    });
    await waitFor(() => expect(h.calls).not.toContain("/v1/me"));
    expect(h.calls).not.toContain("/v1/orgs");
  });

  it("refetches on reload", async () => {
    const h = harness();
    let seen = null as unknown as ReturnType<typeof useOrganizations>;
    await h.mount(function Probe() {
      seen = useOrganizations();
      return null;
    });
    await waitFor(() => expect(seen.organizations).toHaveLength(1));

    h.setOrgs([{ id: "org_1", name: "Acme" }, { id: "org_2", name: "Globex" }]);
    await act(async () => { await seen.reload(); });

    expect(seen.organizations).toHaveLength(2);
    expect(seen.total).toBe(2);
  });
});
