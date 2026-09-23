import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { AuthManager } from "./auth";
import { AuthClientError } from "./client";
import type { Session, User } from "./types";

const user = { id: "user_1", email: "dev@example.test", roles: [] } as unknown as User & { roles: string[] };
const oldSession: Session = {
  session_token: "old-access",
  refresh_token: "old-refresh",
  expires_at: "2000-01-01T00:00:00Z",
};
const freshSession = (): Session => ({
  session_token: "new-access",
  refresh_token: "new-refresh",
  expires_at: new Date(Date.now() + 3_600_000).toISOString(),
});

function setup(session = oldSession) {
  const values = new Map([["authsome:session", JSON.stringify(session)]]);
  const storage = {
    getItem: (key: string) => values.get(key) ?? null,
    setItem: (key: string, value: string) => { values.set(key, value); },
    removeItem: (key: string) => { values.delete(key); },
  };
  const manager = () => new AuthManager({ baseURL: "https://auth.test", storage });
  return { values, manager };
}

beforeEach(() => vi.useFakeTimers());
afterEach(() => { vi.useRealTimers(); vi.unstubAllGlobals(); });

describe("session recovery", () => {
  it("persists rotated tokens before a profile request can fail", async () => {
    const { values, manager } = setup();
    const auth = manager();
    const replacement = freshSession();
    vi.spyOn(auth.getClient(), "refresh").mockResolvedValue(replacement);
    const profile = vi.spyOn(auth.getClient(), "getMe")
      .mockRejectedValueOnce(new TypeError("server restarting"))
      .mockResolvedValue(user);

    await auth.initialize();
    expect(JSON.parse(values.get("authsome:session")!)).toEqual(replacement);
    expect(auth.getState().status).toBe("unknown");
    await vi.advanceTimersByTimeAsync(30_000);
    expect(auth.getState()).toMatchObject({ status: "authenticated", session: replacement });
    expect(profile).toHaveBeenLastCalledWith("new-access");
    expect(auth.getClient().refresh).toHaveBeenCalledTimes(1);
    auth.destroy();
  });

  it("shares initialization when React mounts an effect twice", async () => {
    const { manager } = setup();
    const auth = manager();
    let exchanges = 0;
    vi.spyOn(auth.getClient(), "refresh").mockImplementation(async () => {
      if (++exchanges > 1) throw new AuthClientError("spent refresh token", 401);
      return freshSession();
    });
    vi.spyOn(auth.getClient(), "getMe").mockResolvedValue(user);
    const first = auth.initialize();
    auth.destroy();
    await Promise.all([first, auth.initialize()]);
    expect(auth.getState().status).toBe("authenticated");
    expect(exchanges).toBe(1);
    auth.destroy();
  });

  it("adopts tokens saved by another manager before exchanging a stale token", async () => {
    const { manager } = setup();
    const first = manager();
    const second = manager();
    let exchanges = 0;
    const refresh = async () => {
      if (++exchanges > 1) throw new AuthClientError("spent refresh token", 401);
      return freshSession();
    };
    for (const auth of [first, second]) {
      vi.spyOn(auth.getClient(), "refresh").mockImplementation(refresh);
      vi.spyOn(auth.getClient(), "getMe").mockResolvedValue(user);
    }
    await Promise.all([first.initialize(), second.initialize()]);
    expect(first.getState().status).toBe("authenticated");
    expect(second.getState().status).toBe("authenticated");
    expect(exchanges).toBe(1);
    first.destroy(); second.destroy();
  });

  it("uses the browser lock to serialize refresh across tabs", async () => {
    const { manager } = setup();
    const auth = manager();
    let locked = false;
    const request = vi.fn(async (_name: string, callback: () => Promise<void>) => {
      locked = true;
      try { await callback(); } finally { locked = false; }
    });
    vi.stubGlobal("navigator", { locks: { request } });
    vi.spyOn(auth.getClient(), "refresh").mockImplementation(async () => {
      expect(locked).toBe(true);
      return freshSession();
    });
    vi.spyOn(auth.getClient(), "getMe").mockResolvedValue(user);
    await auth.initialize();
    expect(request).toHaveBeenCalledTimes(1);
    expect(auth.getState().status).toBe("authenticated");
    auth.destroy();
  });

  it("renews an access token rejected before its advertised expiry", async () => {
    const { manager } = setup({ ...oldSession, expires_at: freshSession().expires_at });
    const auth = manager();
    vi.spyOn(auth.getClient(), "getMe")
      .mockRejectedValueOnce(new AuthClientError("expired access token", 401))
      .mockResolvedValue(user);
    vi.spyOn(auth.getClient(), "refresh").mockResolvedValue(freshSession());
    await auth.initialize();
    expect(auth.getState().status).toBe("authenticated");
    expect(auth.getClient().refresh).toHaveBeenCalledTimes(1);
    auth.destroy();
  });
});

it("removes rejected credentials so recovery cannot loop back from sign-in", async () => {
  const { values, manager } = setup();
  const auth = manager();
  vi.spyOn(auth.getClient(), "refresh").mockRejectedValue(new AuthClientError("revoked", 401));
  await auth.initialize();
  expect(auth.getState().status).toBe("unauthenticated");
  expect(values.has("authsome:session")).toBe(false);
  auth.destroy();
});
