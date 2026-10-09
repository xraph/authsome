import { afterEach, describe, expect, it, vi } from "vitest";
import { AuthManager, SESSION_STORAGE_KEY } from "./auth";
import type { Session, User } from "./types";

const user = { id: "usr_1", email: "ada@test" } as User;
const managers: AuthManager[] = [];
afterEach(() => managers.splice(0).forEach((manager) => manager.destroy()));
const json = (body: unknown, status = 200) => new Response(JSON.stringify(body), { status });

function setup(authenticated = false) {
  const session: Session = {
    session_token: "existing-token", refresh_token: "existing-refresh",
    expires_at: new Date(Date.now() + 3_600_000).toISOString(),
  };
  const store = new Map<string, string>();
  if (authenticated) store.set(SESSION_STORAGE_KEY, JSON.stringify(session));
  const routes = new Map<string, () => Response | Promise<Response>>([
    ["GET /v1/me", () => json(user)],
    ["POST /v1/signout", () => json({ status: "ok" })],
    ["POST /v1/signin", () => json({ error: "MFA required", type: "mfa_required", mfa_ticket: "ticket-1", available_methods: ["totp"] }, 403)],
  ]);
  const fetch = vi.fn<typeof globalThis.fetch>(async (input, init) => {
    const route = `${init?.method} ${new URL(String(input)).pathname}`;
    const handler = routes.get(route);
    if (!handler) throw new Error(`Unexpected route: ${route}`);
    return handler();
  });
  const onStateChange = vi.fn();
  const onError = vi.fn();
  const manager = new AuthManager({
    baseURL: "https://api.test", fetch, onStateChange, onError,
    storage: {
      getItem: (key) => store.get(key) ?? null,
      setItem: (key, value) => { store.set(key, value); },
      removeItem: (key) => { store.delete(key); },
    },
  });
  managers.push(manager);
  return { manager, store, routes, fetch, session, onStateChange, onError };
}

const credentials = { email: "ada@test", password: "password" };

describe("recovery login", () => {
  it("redeems the real sign-in ticket and persists only the issued session", async () => {
    const { manager, routes, fetch, store, session } = setup();
    const issued = { ...session, session_token: "issued-token", refresh_token: "issued-refresh" };
    routes.set("POST /v1/mfa/challenge", () => json({ ...issued, user }));
    await manager.signIn(credentials);
    expect(manager.getState().status).toBe("mfa_required");
    expect(store.has(SESSION_STORAGE_KEY)).toBe(false);
    await manager.submitRecoveryCode("RECOVERY");
    expect(fetch).toHaveBeenLastCalledWith("https://api.test/v1/mfa/challenge", expect.objectContaining({
      method: "POST", body: JSON.stringify({ mfa_ticket: "ticket-1", code: "RECOVERY" }),
    }));
    expect(fetch).toHaveBeenCalledTimes(2);
    expect(manager.getState()).toEqual({ status: "authenticated", user, session: issued });
    expect(JSON.parse(store.get(SESSION_STORAGE_KEY)!)).toEqual(issued);
  });

  it.each([401, 500])("propagates rejection %s and retains the ticket", async (status) => {
    const { manager, routes, store, onError } = setup();
    routes.set("POST /v1/mfa/challenge", () => json({ error: "recovery failed" }, status));
    await manager.signIn(credentials);
    const state = manager.getState();
    await expect(manager.submitRecoveryCode("BADCODE")).rejects.toThrow("recovery failed");
    expect(manager.getState()).toBe(state);
    expect(manager.getMFATicket()).toBe("ticket-1");
    expect(store.has(SESSION_STORAGE_KEY)).toBe(false);
    expect(onError).toHaveBeenCalledWith({ error: "recovery failed" });
  });

  it.each([false, true])("rejects recovery outside ticket state (authenticated=%s)", async (authenticated) => {
    const { manager, fetch, store } = setup(authenticated);
    await manager.initialize();
    fetch.mockClear();
    const before = store.get(SESSION_STORAGE_KEY);
    await expect(manager.submitRecoveryCode("RECOVERY")).rejects.toThrow("mfa_required");
    expect(fetch).not.toHaveBeenCalled();
    expect(store.get(SESSION_STORAGE_KEY)).toBe(before);
  });
});

describe("authenticated SMS ceremony", () => {
  it.each(["idle", "unauthenticated", "mfa_required"])("rejects %s without a request", async (status) => {
    const { manager, fetch } = setup();
    if (status === "unauthenticated") await manager.initialize();
    if (status === "mfa_required") await manager.signIn(credentials);
    fetch.mockClear();
    const state = manager.getState();
    await expect(manager.sendSMSCode()).rejects.toThrow("authenticated session");
    await expect(manager.submitSMSCode("123456")).rejects.toThrow("authenticated session");
    expect(fetch).not.toHaveBeenCalled();
    expect(manager.getState()).toBe(state);
  });

  it("sends and verifies with the existing token without issuing a session or auth notification", async () => {
    const { manager, routes, fetch, store, onStateChange } = setup(true);
    const sent = { sent: true, phone_masked: "+1***1234", expires_in_seconds: 300 };
    routes.set("POST /v1/mfa/sms/send", () => json(sent));
    routes.set("POST /v1/mfa/sms/verify", () => json({ verified: true, method: "sms" }));
    await manager.initialize();
    const state = manager.getState();
    const stored = store.get(SESSION_STORAGE_KEY);
    onStateChange.mockClear();
    await expect(manager.sendSMSCode()).resolves.toEqual(sent);
    await manager.submitSMSCode("123456");
    expect(fetch).toHaveBeenNthCalledWith(2, "https://api.test/v1/mfa/sms/send", expect.objectContaining({
      method: "POST", body: "{}", headers: expect.objectContaining({ Authorization: "Bearer existing-token" }),
    }));
    expect(fetch).toHaveBeenLastCalledWith("https://api.test/v1/mfa/sms/verify", expect.objectContaining({
      method: "POST", body: JSON.stringify({ code: "123456" }), headers: expect.objectContaining({ Authorization: "Bearer existing-token" }),
    }));
    expect(manager.getState()).toBe(state);
    expect(manager.getUser()).toEqual(user);
    expect(manager.getSessionToken()).toBe("existing-token");
    expect(store.get(SESSION_STORAGE_KEY)).toBe(stored);
    expect(onStateChange).not.toHaveBeenCalled();
  });

  it.each([
    [200, { verified: false, method: "sms" }],
    [200, { verified: true, method: "totp" }],
    [401, { error: "invalid code" }],
    [500, { error: "unavailable" }],
  ])("propagates verify failure %s %j without losing the session", async (status, body) => {
    const { manager, routes, store, onStateChange } = setup(true);
    routes.set("POST /v1/mfa/sms/verify", () => json(body, status));
    await manager.initialize();
    const state = manager.getState();
    const stored = store.get(SESSION_STORAGE_KEY);
    onStateChange.mockClear();
    await expect(manager.submitSMSCode("123456")).rejects.toThrow();
    expect(manager.getState()).toBe(state);
    expect(store.get(SESSION_STORAGE_KEY)).toBe(stored);
    expect(onStateChange).not.toHaveBeenCalled();
  });

  it.each([200, 500])("propagates send failure %s without changing the session", async (status) => {
    const { manager, routes, store } = setup(true);
    routes.set("POST /v1/mfa/sms/send", () => json({ sent: false }, status));
    await manager.initialize();
    const state = manager.getState();
    const stored = store.get(SESSION_STORAGE_KEY);
    await expect(manager.sendSMSCode()).rejects.toThrow();
    expect(manager.getState()).toBe(state);
    expect(store.get(SESSION_STORAGE_KEY)).toBe(stored);
  });

  it.each(["signout", "identity", "destroy", "rejection"])("does not restore or overwrite identity after in-flight %s", async (change) => {
    const { manager, routes, store, session, onStateChange } = setup(true);
    let finish!: (value: Response) => void;
    routes.set("POST /v1/mfa/sms/verify", () => new Promise((resolve) => { finish = resolve; }));
    await manager.initialize();
    const pending = manager.submitSMSCode("123456");
    const rejected = expect(pending).rejects.toThrow();
    if (change === "signout" || change === "rejection") await manager.signOut();
    if (change === "destroy") manager.destroy();
    if (change === "identity") {
      routes.set("POST /v1/signin", () => json({ ...session, session_token: "new-token", user: { ...user, id: "usr_2" } }));
      await manager.signIn(credentials);
    }
    const state = manager.getState();
    const stored = store.get(SESSION_STORAGE_KEY);
    onStateChange.mockClear();
    finish(change === "rejection" ? json({ error: "late failure" }, 401) : json({ verified: true, method: "sms" }));
    await rejected;
    expect(manager.getState()).toBe(state);
    expect(store.get(SESSION_STORAGE_KEY)).toBe(stored);
    expect(onStateChange).not.toHaveBeenCalled();
  });
});
