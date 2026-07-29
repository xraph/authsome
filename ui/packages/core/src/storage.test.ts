import { afterEach, describe, expect, it, vi } from "vitest";

import { AuthManager, createLocalStorage, SESSION_STORAGE_KEY } from "./index";

/** Installs a fake window.localStorage and reports what lands in it. */
function stubLocalStorage() {
  const backing = new Map<string, string>();
  vi.stubGlobal("window", {
    localStorage: {
      getItem: (k: string) => backing.get(k) ?? null,
      setItem: (k: string, v: string) => void backing.set(k, v),
      removeItem: (k: string) => void backing.delete(k),
    },
  });
  return backing;
}

async function signInWith(storage?: ReturnType<typeof createLocalStorage>) {
  const mgr = new AuthManager({ baseURL: "https://api.test", storage });
  const client = mgr.getClient() as unknown as Record<string, unknown>;
  client.signIn = async () => ({
    user: { id: "u_1", email: "a@b.c" },
    session_token: "tok",
    refresh_token: "refresh_good_for_weeks",
    expires_at: new Date(Date.now() + 3600_000).toISOString(),
  });
  client.getMe = async () => ({ id: "u_1", email: "a@b.c" });
  await mgr.signIn({ email: "a@b.c", password: "pw" });
  return mgr;
}

afterEach(() => vi.unstubAllGlobals());

describe("default token storage", () => {
  // A refresh token in localStorage is readable by any script on the page, so
  // one XSS turns a page-lifetime problem into weeks of account access. The
  // default must not opt everyone into that silently.
  it("does not write tokens to localStorage", async () => {
    const backing = stubLocalStorage();

    await signInWith();

    expect(backing.size).toBe(0);
    expect(backing.get(SESSION_STORAGE_KEY)).toBeUndefined();
  });

  it("still holds the session for the life of the page", async () => {
    stubLocalStorage();

    const mgr = await signInWith();

    const state = mgr.getState();
    expect(state.status).toBe("authenticated");
    expect(state.status === "authenticated" && state.session.session_token).toBe("tok");
  });

  // Persistence remains available, but as a deliberate choice.
  it("persists when createLocalStorage() is passed explicitly", async () => {
    const backing = stubLocalStorage();

    await signInWith(createLocalStorage());

    const stored = backing.get(SESSION_STORAGE_KEY);
    expect(stored).toBeTruthy();
    expect(JSON.parse(stored as string).session_token).toBe("tok");
  });

  it("createLocalStorage falls back to memory when localStorage is absent", () => {
    vi.stubGlobal("window", undefined);
    const storage = createLocalStorage();
    storage.setItem("k", "v");
    expect(storage.getItem("k")).toBe("v");
  });
});
