import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { AuthManager, SESSION_STORAGE_KEY } from "./auth";
import type { Session } from "./types";

const MAX_DELAY = 2_147_483_647;
const managers: AuthManager[] = [];
const json = (body: unknown, status = 200) => new Response(JSON.stringify(body), { status });

beforeEach(() => {
  vi.useFakeTimers();
  vi.setSystemTime(new Date("2026-10-09T12:00:00Z"));
});
afterEach(() => {
  managers.splice(0).forEach((manager) => manager.destroy());
  vi.restoreAllMocks();
  vi.useRealTimers();
});

function setup(ttl: number) {
  const session: Session = {
    session_token: "token", refresh_token: "refresh",
    expires_at: new Date(Date.now() + ttl).toISOString(),
  };
  const store = new Map([[SESSION_STORAGE_KEY, JSON.stringify(session)]]);
  const refresh = vi.fn(() => json({
    session_token: "rotated", refresh_token: "rotated-refresh",
    expires_at: new Date(Date.now() + 3_600_000).toISOString(),
  }));
  const routes = new Map<string, () => Response | Promise<Response>>([
    ["GET /v1/me", () => json({ id: "usr_1" })],
    ["POST /v1/refresh", refresh],
    ["POST /v1/signout", () => json({ status: "ok" })],
  ]);
  const fetch = vi.fn<typeof globalThis.fetch>(async (input, init) => {
    const route = `${init?.method} ${new URL(String(input)).pathname}`;
    const handler = routes.get(route);
    if (!handler) throw new Error(`Unexpected route: ${route}`);
    return handler();
  });
  const manager = new AuthManager({
    baseURL: "https://api.test", fetch,
    storage: {
      getItem: (key) => store.get(key) ?? null,
      setItem: (key, value) => { store.set(key, value); },
      removeItem: (key) => { store.delete(key); },
    },
  });
  managers.push(manager);
  return { manager, store, routes, session, refresh, fetch };
}

describe("session refresh deadlines", () => {
  it("keeps a 2099 expiry and schedules only a bounded wait", async () => {
    const expiry = "2099-06-01T12:00:00.000Z";
    const { manager, store, refresh } = setup(Date.parse(expiry) - Date.now());
    const timeout = vi.spyOn(globalThis, "setTimeout");
    await manager.initialize();
    expect(JSON.parse(store.get(SESSION_STORAGE_KEY)!).expires_at).toBe(expiry);
    expect(timeout).toHaveBeenLastCalledWith(expect.any(Function), MAX_DELAY);
    await vi.advanceTimersByTimeAsync(1);
    expect(refresh).not.toHaveBeenCalled();
  });

  it("re-evaluates after each bounded wait without refreshing before the deadline", async () => {
    const { manager, refresh, store, session } = setup(2 * MAX_DELAY + 61_000);
    const timeout = vi.spyOn(globalThis, "setTimeout");
    await manager.initialize();
    await vi.advanceTimersByTimeAsync(MAX_DELAY);
    expect(refresh).not.toHaveBeenCalled();
    await vi.advanceTimersByTimeAsync(MAX_DELAY);
    expect(refresh).not.toHaveBeenCalled();
    expect(store.get(SESSION_STORAGE_KEY)).toBe(JSON.stringify(session));
    await vi.advanceTimersByTimeAsync(999);
    expect(refresh).not.toHaveBeenCalled();
    await vi.advanceTimersByTimeAsync(1);
    expect(refresh).toHaveBeenCalledTimes(1);
    expect(manager.getSessionToken()).toBe("rotated");
    for (const [, delay] of timeout.mock.calls) {
      expect(delay).toBeGreaterThan(0);
      expect(delay).toBeLessThanOrEqual(MAX_DELAY);
    }
  });

  it("refreshes a session already inside the sixty-second window", async () => {
    const { manager, refresh } = setup(30_000);
    await manager.initialize();
    await vi.advanceTimersByTimeAsync(0);
    expect(refresh).toHaveBeenCalledTimes(1);
    expect(manager.getSessionToken()).toBe("rotated");
  });

  it("preserves the refresh retry delay after a transport failure", async () => {
    const { manager, refresh } = setup(120_000);
    refresh.mockImplementationOnce(() => json({ error: "unavailable" }, 500));
    await manager.initialize();
    await vi.advanceTimersByTimeAsync(60_000);
    expect(refresh).toHaveBeenCalledTimes(1);
    expect(manager.getState().status).toBe("unknown");
    await vi.advanceTimersByTimeAsync(29_999);
    expect(refresh).toHaveBeenCalledTimes(1);
    await vi.advanceTimersByTimeAsync(1);
    expect(refresh).toHaveBeenCalledTimes(2);
    expect(manager.getSessionToken()).toBe("rotated");
  });

  it("cancels the old deadline when a new session replaces it", async () => {
    const { manager, routes, refresh, store } = setup(120_000);
    await manager.initialize();
    const replacement = {
      session_token: "new-token", refresh_token: "new-refresh",
      expires_at: new Date(Date.now() + 2 * MAX_DELAY).toISOString(),
    };
    routes.set("POST /v1/signin", () => json({ ...replacement, user: { id: "usr_2" } }));
    await manager.signIn({ email: "other@test", password: "password" });
    expect(vi.getTimerCount()).toBe(1);
    await vi.advanceTimersByTimeAsync(60_000);
    expect(refresh).not.toHaveBeenCalled();
    expect(manager.getSessionToken()).toBe("new-token");
    expect(store.get(SESSION_STORAGE_KEY)).toBe(JSON.stringify(replacement));
  });

  it("cancels the bounded wait when destroyed", async () => {
    const { manager, refresh } = setup(2 * MAX_DELAY);
    await manager.initialize();
    manager.destroy();
    expect(vi.getTimerCount()).toBe(0);
    await vi.advanceTimersByTimeAsync(2 * MAX_DELAY);
    expect(refresh).not.toHaveBeenCalled();
  });

  it("cancels before sign-out waits for the server and cannot refresh after sign-out", async () => {
    const { manager, routes, refresh, store } = setup(MAX_DELAY + 60_000);
    let finish!: (response: Response) => void;
    routes.set("POST /v1/signout", () => new Promise((resolve) => { finish = resolve; }));
    await manager.initialize();
    const signout = manager.signOut();
    expect(vi.getTimerCount()).toBe(0);
    await vi.advanceTimersByTimeAsync(MAX_DELAY);
    expect(refresh).not.toHaveBeenCalled();
    finish(json({ status: "ok" }));
    await signout;
    await vi.advanceTimersByTimeAsync(MAX_DELAY);
    expect(refresh).not.toHaveBeenCalled();
    expect(manager.getState().status).toBe("unauthenticated");
    expect(store.has(SESSION_STORAGE_KEY)).toBe(false);
  });
});
