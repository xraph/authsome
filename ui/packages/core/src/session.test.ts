import { describe, expect, it } from "vitest";

import { AuthManager } from "./auth";
import type { AuthResponse } from "./generated/api-types";

// toSession is internal, so it is exercised through the state it produces:
// AuthManager schedules its refresh from session.expires_at, and hardcoding an
// hour meant that timer was set against a lifetime the token did not have.
// These assert the derivation directly via a stubbed client.
describe("session expiry derivation", () => {
  function managerWithSignInResponse(res: Partial<AuthResponse>) {
    const store = new Map<string, string>();
    const mgr = new AuthManager({
      baseURL: "https://api.test",
      storage: {
        getItem: (k) => store.get(k) ?? null,
        setItem: (k, v) => void store.set(k, v),
        removeItem: (k) => void store.delete(k),
      },
    });
    const client = mgr.getClient() as unknown as Record<string, unknown>;
    client.signIn = async () => ({
      user: { id: "u_1", email: "a@b.c" },
      session_token: "tok",
      refresh_token: "refresh",
      ...res,
    });
    client.getMe = async () => ({ id: "u_1", email: "a@b.c" });
    return { mgr, store };
  }

  it("uses the server's expires_at verbatim", async () => {
    const serverExpiry = "2099-06-01T12:00:00.000Z";
    const { mgr, store } = managerWithSignInResponse({ expires_at: serverExpiry });

    await mgr.signIn({ email: "a@b.c", password: "pw" });

    const persisted = JSON.parse(store.get("authsome:session") ?? "{}");
    expect(persisted.expires_at).toBe(serverExpiry);
  });

  // A short server TTL is the case that broke: assuming an hour left the client
  // holding a dead token without refreshing.
  it("honours a server TTL shorter than the fallback", async () => {
    const shortExpiry = new Date(Date.now() + 5 * 60_000).toISOString();
    const { mgr, store } = managerWithSignInResponse({ expires_at: shortExpiry });

    await mgr.signIn({ email: "a@b.c", password: "pw" });

    const persisted = JSON.parse(store.get("authsome:session") ?? "{}");
    expect(persisted.expires_at).toBe(shortExpiry);
    expect(new Date(persisted.expires_at).getTime()).toBeLessThan(Date.now() + 3600_000);
  });

  it("falls back to one hour only when the server omits expires_at", async () => {
    const before = Date.now();
    const { mgr, store } = managerWithSignInResponse({ expires_at: undefined });

    await mgr.signIn({ email: "a@b.c", password: "pw" });

    const persisted = JSON.parse(store.get("authsome:session") ?? "{}");
    const got = new Date(persisted.expires_at).getTime();
    expect(got).toBeGreaterThanOrEqual(before + 3600_000 - 5_000);
    expect(got).toBeLessThanOrEqual(Date.now() + 3600_000 + 5_000);
  });
});
