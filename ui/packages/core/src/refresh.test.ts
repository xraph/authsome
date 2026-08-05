import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { AuthManager } from "./auth";
import { AuthClientError } from "./client";
import type { Session } from "./types";

const expiredSession: Session = {
  session_token: "tok",
  refresh_token: "refresh",
  // Already past, so initialize() goes straight to refreshSession.
  expires_at: new Date(Date.now() - 1000).toISOString(),
};

function managerWithFailingRefresh(err: unknown) {
  const store = new Map<string, string>([
    ["authsome:session", JSON.stringify(expiredSession)],
  ]);
  const mgr = new AuthManager({
    baseURL: "https://api.test",
    storage: {
      getItem: (k) => store.get(k) ?? null,
      setItem: (k, v) => void store.set(k, v),
      removeItem: (k) => void store.delete(k),
    },
  });
  let calls = 0;
  const client = mgr.getClient() as unknown as Record<string, unknown>;
  client.refresh = async () => {
    calls++;
    throw err;
  };
  client.getMe = async () => ({ id: "u_1", email: "a@b.c" });
  return { mgr, store, refreshCalls: () => calls };
}

beforeEach(() => vi.useFakeTimers());
afterEach(() => vi.useRealTimers());

describe("refreshSession failure handling", () => {
  // A rejected refresh token cannot become valid by asking again.
  it("signs out immediately when the token is rejected", async () => {
    const { mgr, refreshCalls } = managerWithFailingRefresh(
      new AuthClientError("invalid refresh token", 401),
    );

    await mgr.initialize();

    expect(mgr.getState().status).toBe("unauthenticated");
    expect(refreshCalls()).toBe(1);

    // No retry is pending.
    await vi.advanceTimersByTimeAsync(10 * 60_000);
    expect(refreshCalls()).toBe(1);
  });

  // The retry loop was previously unbounded: `attempt` was accepted and never
  // read, and the retry re-entered without it, so a tab left open retried
  // every 30s indefinitely and never signed the user out.
  it("stops retrying after a bounded number of attempts", async () => {
    const { mgr, refreshCalls } = managerWithFailingRefresh(
      new TypeError("fetch failed"),
    );

    await mgr.initialize();
    expect(refreshCalls()).toBe(1);

    // Far longer than the whole backoff schedule.
    await vi.advanceTimersByTimeAsync(60 * 60_000);

    const calls = refreshCalls();
    expect(calls).toBeGreaterThan(1);
    expect(calls).toBeLessThanOrEqual(6); // initial + MAX_REFRESH_ATTEMPTS

    // And it has genuinely stopped, not merely slowed.
    const settled = refreshCalls();
    await vi.advanceTimersByTimeAsync(60 * 60_000);
    expect(refreshCalls()).toBe(settled);
  });

  // Backoff, not a fixed interval: retries must spread out.
  it("backs off between retries", async () => {
    const { refreshCalls } = managerWithFailingRefresh(new TypeError("down"));
    const { mgr } = managerWithFailingRefresh(new TypeError("down"));

    await mgr.initialize();
    await vi.advanceTimersByTimeAsync(30_000);
    const afterFirst = refreshCalls();

    // A fixed 30s interval would have fired several times by 90s; backoff
    // means the third retry is still pending.
    await vi.advanceTimersByTimeAsync(60_000);
    expect(refreshCalls()).toBeLessThanOrEqual(afterFirst + 2);
  });

  // Exhausting retries means no verdict was obtained — not that the user is
  // signed out, and not that they are authenticated.
  it("lands in unknown, keeping the session, when the server never answers", async () => {
    const { mgr, store } = managerWithFailingRefresh(new TypeError("fetch failed"));

    await mgr.initialize();
    await vi.advanceTimersByTimeAsync(60 * 60_000);

    expect(mgr.getState().status).toBe("unknown");
    expect(store.get("authsome:session")).toBeTruthy();
  });
});
