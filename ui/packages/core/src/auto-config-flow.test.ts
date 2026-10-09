import { afterEach, describe, expect, it, vi } from "vitest";
import { AuthManager } from "./auth";
import { AuthClientError } from "./client";
import type { ClientConfig, TokenStorage, User } from "./types";

const managers: AuthManager[] = [];
afterEach(() => managers.splice(0).forEach((manager) => manager.destroy()));
function manager(
  config?: ClientConfig,
  storage?: TokenStorage,
  key = "pk_a",
  baseURL = "https://api.test",
) {
  const auth = new AuthManager({
    baseURL,
    publishableKey: key,
    initialClientConfig: config,
    storage,
  });
  managers.push(auth);
  return auth;
}
function storage(): TokenStorage {
  const data = new Map<string, string>();
  return {
    getItem: (key) => data.get(key) ?? null,
    setItem: (key, value) => void data.set(key, value),
    removeItem: (key) => void data.delete(key),
  };
}
const user = { id: "u_1", email: "ada@example.com" } as User;
const response = {
  user,
  session_token: "token",
  refresh_token: "refresh",
  expires_at: "2099-01-01T00:00:00Z",
};

describe("AutoConfig signup completion", () => {
  it("waits for configuration and refuses closed signup before creating an account", async () => {
    const auth = manager();
    const client = auth.getClient();
    vi.spyOn(client, "fetchClientConfig").mockResolvedValue({
      signup_enabled: false,
    });
    const signup = vi.spyOn(client, "signUp");
    await expect(
      auth.signUp({ email: user.email, password: "pw" }),
    ).rejects.toThrow("Signup isn't available");
    expect(signup).not.toHaveBeenCalled();
  });
  it.each([undefined, { enabled: true, required: true }])(
    "keeps verification pending for policy %j",
    async (email_verification) => {
      const auth = manager({ email_verification });
      vi.spyOn(auth.getClient(), "signUp").mockResolvedValue(response);
      const profile = vi.spyOn(auth.getClient(), "getMe");
      await auth.signUp({ email: user.email, password: "pw" });
      expect(auth.getState()).toEqual({
        status: "verification_pending",
        email: user.email,
      });
      expect(auth.getSessionToken()).toBeNull();
      expect(profile).not.toHaveBeenCalled();
    },
  );
  it.each([
    { enabled: true, required: false },
    { enabled: false, required: true },
  ])(
    "authenticates a validated session for policy %j",
    async (email_verification) => {
      const auth = manager({ email_verification });
      vi.spyOn(auth.getClient(), "signUp").mockResolvedValue(response);
      const profile = vi
        .spyOn(auth.getClient(), "getMe")
        .mockResolvedValue({ ...user, roles: [] });
      await auth.signUp({ email: user.email, password: "pw" });
      expect(profile).toHaveBeenCalledWith("token");
      expect(auth.getState().status).toBe("authenticated");
    },
  );
  it("keeps synthetic duplicate signup sessions out of storage", async () => {
    const auth = manager({
      email_verification: { enabled: true, required: false },
    });
    vi.spyOn(auth.getClient(), "signUp").mockResolvedValue(response);
    vi.spyOn(auth.getClient(), "getMe").mockRejectedValue(
      new AuthClientError("invalid session", 401),
    );
    await auth.signUp({ email: user.email, password: "pw" });
    expect(auth.getState().status).toBe("verification_pending");
    expect(auth.getSessionToken()).toBeNull();
  });
  it("reports profile transport failures instead of claiming authentication", async () => {
    const auth = manager({
      email_verification: { enabled: false, required: false },
    });
    vi.spyOn(auth.getClient(), "signUp").mockResolvedValue(response);
    vi.spyOn(auth.getClient(), "getMe").mockRejectedValue(
      new Error("network unavailable"),
    );
    await expect(
      auth.signUp({ email: user.email, password: "pw" }),
    ).rejects.toThrow("network unavailable");
    expect(auth.getSessionToken()).toBeNull();
  });
});

describe("AutoConfig cache isolation", () => {
  it("shares a cached config only for the same server and publishable key", async () => {
    const shared = storage();
    const first = manager(undefined, shared);
    vi.spyOn(first.getClient(), "fetchClientConfig").mockResolvedValue({
      app_id: "app_a",
    });
    await first.fetchClientConfig();
    for (const [key, host, expected] of [
      ["pk_a", "https://api.test/", "app_a"],
      ["pk_b", "https://api.test", "fresh"],
      ["pk_a", "https://other.test", "fresh"],
    ]) {
      const next = manager(undefined, shared, key, host);
      const fetch = vi
        .spyOn(next.getClient(), "fetchClientConfig")
        .mockResolvedValue({ app_id: "fresh" });
      expect((await next.fetchClientConfig()).app_id).toBe(expected);
      expect(fetch).toHaveBeenCalledTimes(expected === "fresh" ? 1 : 0);
    }
  });
});
