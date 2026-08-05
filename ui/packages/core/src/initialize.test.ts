import { describe, expect, it } from "vitest";

import { AuthManager } from "./auth";
import { AuthClientError } from "./client";
import type { Session } from "./types";

const liveSession: Session = {
  session_token: "tok",
  refresh_token: "refresh",
  expires_at: new Date(Date.now() + 3600_000).toISOString(),
};

function managerWith(getMeFails: unknown, stored: Session | null = liveSession) {
  const store = new Map<string, string>();
  if (stored) {
    store.set("authsome:session", JSON.stringify(stored));
  }
  const mgr = new AuthManager({
    baseURL: "https://api.test",
    storage: {
      getItem: (k) => store.get(k) ?? null,
      setItem: (k, v) => void store.set(k, v),
      removeItem: (k) => void store.delete(k),
    },
  });
  const client = mgr.getClient() as unknown as Record<string, unknown>;
  client.getMe = async () => {
    throw getMeFails;
  };
  client.refresh = async () => {
    throw getMeFails;
  };
  return { mgr, store };
}

describe("initialize failure handling", () => {
  // The previous behaviour reported "authenticated" with a null user, so
  // anything gating on isAuthenticated rendered protected UI for a session
  // nothing had validated — and user.email crashed.
  it("does not report authenticated when the server is unreachable", async () => {
    const { mgr } = managerWith(new TypeError("fetch failed"));

    await mgr.initialize();

    expect(mgr.getState().status).not.toBe("authenticated");
    expect(mgr.getState().status).toBe("unknown");
  });

  it("keeps the session in storage through an outage", async () => {
    const { mgr, store } = managerWith(new TypeError("fetch failed"));

    await mgr.initialize();

    expect(store.get("authsome:session")).toBeTruthy();
    const state = mgr.getState();
    expect(state.status === "unknown" && state.session.session_token).toBe("tok");
  });

  // A verdict of "your credential is bad" is different from "no verdict".
  describe("signs out only on an explicit rejection", () => {
    it.each([
      ["401", 401],
      ["403", 403],
    ])("%s", async (_label, code) => {
      const { mgr } = managerWith(new AuthClientError("nope", code));
      await mgr.initialize();
      expect(mgr.getState().status).toBe("unauthenticated");
    });
  });

  // A misconfigured baseURL that 404s or 500s must not log everyone out.
  describe("treats non-auth failures as unknown, not signed out", () => {
    it.each([
      ["404 — wrong baseURL", 404],
      ["500 — server error", 500],
      ["502 — gateway down", 502],
    ])("%s", async (_label, code) => {
      const { mgr } = managerWith(new AuthClientError("boom", code));
      await mgr.initialize();
      expect(mgr.getState().status).toBe("unknown");
    });
  });

  it("reports unauthenticated when nothing is stored", async () => {
    const { mgr } = managerWith(new TypeError("fetch failed"), null);
    await mgr.initialize();
    expect(mgr.getState().status).toBe("unauthenticated");
  });

  it("reports unauthenticated when storage holds unparseable data", async () => {
    const { mgr, store } = managerWith(new TypeError("fetch failed"));
    store.set("authsome:session", "not-json");
    await mgr.initialize();
    expect(mgr.getState().status).toBe("unauthenticated");
  });
});
