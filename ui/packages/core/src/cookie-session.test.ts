import { afterEach, describe, expect, it, vi } from "vitest";

import { AuthManager } from "./auth";
import type { TokenStorage } from "./types";

// session: "cookie" keeps the session in the backend's httpOnly cookie, so no
// token is stored, sent or kept, and a reload asks the backend who's signed
// in. These drive the real manager over a fake fetch that records every
// request.

const BASE = "https://api.test";
const user = { id: "ausr_1", email: "ada@example.com" };

type Sent = { url: string; init: RequestInit };

/** backend answers each path with a response, recording the calls. */
function backend(answers: Record<string, () => Response>) {
  const sent: Sent[] = [];
  const fetch = vi.fn(async (url: string | URL | Request, init?: RequestInit) => {
    const u = String(url);
    sent.push({ url: u, init: init ?? {} });
    const answer = answers[new URL(u).pathname];
    if (!answer) throw new TypeError(`no answer for ${u}`);
    return answer();
  });
  const to = (path: string) => sent.filter((s) => s.url.endsWith(path));
  return { fetch: fetch as unknown as typeof globalThis.fetch, sent, to };
}

const json = (status: number, body: unknown) => () =>
  new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });

/** storage that records every session write. */
function watchedStorage(): TokenStorage & { writes: string[] } {
  const writes: string[] = [];
  return {
    writes,
    getItem: () => null,
    setItem: (key: string) => {
      writes.push(key);
    },
    removeItem: () => {},
  };
}

function authorization(s: Sent): string | null {
  return new Headers(s.init.headers).get("Authorization");
}

const signInResponse = (expiresInMs: number) =>
  json(200, {
    user,
    session_token: "secret-session",
    refresh_token: "secret-refresh",
    expires_at: new Date(Date.now() + expiresInMs).toISOString(),
  });

afterEach(() => {
  vi.useRealTimers();
});

describe("cookie session", () => {
  it("restores a session by asking the backend, sending only the cookie", async () => {
    const { fetch, sent } = backend({ "/v1/me": json(200, user) });
    const auth = new AuthManager({ baseURL: BASE, fetch, session: "cookie" });
    await auth.initialize();

    expect(auth.getState().status).toBe("authenticated");
    expect(auth.getUser()).toMatchObject({ email: "ada@example.com" });
    expect(auth.getSessionToken()).toBeFalsy();
    expect(sent).toHaveLength(1);
    expect(sent[0].init.credentials).toBe("include");
    expect(authorization(sent[0])).toBeNull();
  });

  it.each([401, 403])("is signed out when the backend answers %i", async (status) => {
    const { fetch } = backend({ "/v1/me": json(status, { error: "unauthorized" }) });
    const auth = new AuthManager({ baseURL: BASE, fetch, session: "cookie" });
    await auth.initialize();
    expect(auth.getState().status).toBe("unauthenticated");
  });

  it("doesn't sign you out when the backend can't be reached, and asks again later", async () => {
    vi.useFakeTimers();
    let up = false;
    const { fetch, to } = backend({
      "/v1/me": () => {
        if (!up) throw new TypeError("fetch failed");
        return json(200, user)();
      },
    });
    const auth = new AuthManager({ baseURL: BASE, fetch, session: "cookie" });
    await auth.initialize();
    expect(auth.getState().status).toBe("unknown");

    up = true;
    await vi.advanceTimersByTimeAsync(30_000);
    expect(to("/v1/me")).toHaveLength(2);
    expect(auth.getState().status).toBe("authenticated");
    auth.destroy();
  });

  it("keeps no token from signing in, and stores nothing", async () => {
    const storage = watchedStorage();
    const { fetch, sent } = backend({ "/v1/signin": signInResponse(3_600_000) });
    const auth = new AuthManager({
      baseURL: BASE,
      fetch,
      storage,
      session: "cookie",
    });
    await auth.signIn({ email: "ada@example.com", password: "x" });

    expect(auth.getState().status).toBe("authenticated");
    expect(auth.getSessionToken()).toBeFalsy();
    expect(JSON.stringify(auth.getState())).not.toContain("secret");
    expect(storage.writes).toEqual([]);
    expect(sent[0].init.credentials).toBe("include");
    auth.destroy();
  });

  // The backend's auto-refresh middleware renews the cookie. A refresh from
  // here would have no refresh token to send.
  it("schedules no refresh, even for a session about to expire", async () => {
    vi.useFakeTimers();
    const { fetch, sent } = backend({ "/v1/signin": signInResponse(1_000) });
    const auth = new AuthManager({ baseURL: BASE, fetch, session: "cookie" });
    await auth.signIn({ email: "ada@example.com", password: "x" });

    await vi.advanceTimersByTimeAsync(3_600_000);
    expect(sent.map((s) => new URL(s.url).pathname)).toEqual(["/v1/signin"]);
    expect(auth.getState().status).toBe("authenticated");
    auth.destroy();
  });

  it("refreshes by checking the cookie still holds", async () => {
    let signedIn = true;
    const { fetch, to } = backend({
      "/v1/me": () => (signedIn ? json(200, user)() : json(401, {})()),
    });
    const auth = new AuthManager({ baseURL: BASE, fetch, session: "cookie" });
    await auth.initialize();

    signedIn = false;
    await auth.refreshNow();
    expect(to("/v1/me")).toHaveLength(2);
    expect(to("/v1/refresh")).toHaveLength(0);
    expect(auth.getState().status).toBe("unauthenticated");
  });

  it("signs out through the backend, which clears the cookie", async () => {
    const { fetch, to } = backend({
      "/v1/me": json(200, user),
      "/v1/signout": json(200, { status: "ok" }),
    });
    const auth = new AuthManager({ baseURL: BASE, fetch, session: "cookie" });
    await auth.initialize();
    await auth.signOut();

    expect(auth.getState().status).toBe("unauthenticated");
    const [out] = to("/v1/signout");
    expect(out.init.method).toBe("POST");
    expect(out.init.credentials).toBe("include");
    expect(authorization(out)).toBeNull();
  });

  it("signs out locally when the backend can't be reached", async () => {
    const { fetch } = backend({ "/v1/me": json(200, user) });
    const auth = new AuthManager({ baseURL: BASE, fetch, session: "cookie" });
    await auth.initialize();
    await auth.signOut();
    expect(auth.getState().status).toBe("unauthenticated");
  });

  it("sends an SMS code on the cookie alone", async () => {
    const { fetch, to } = backend({
      "/v1/me": json(200, user),
      "/v1/mfa/sms/send": json(200, {
        sent: true,
        phone_masked: "***1234",
        expires_in_seconds: 300,
      }),
    });
    const auth = new AuthManager({ baseURL: BASE, fetch, session: "cookie" });
    await auth.initialize();

    await expect(auth.sendSMSCode()).resolves.toMatchObject({ sent: true });
    const [send] = to("/v1/mfa/sms/send");
    expect(send.init.credentials).toBe("include");
    expect(authorization(send)).toBeNull();
  });

  it("wraps a custom fetch without dropping the request's own options", async () => {
    const { fetch, sent } = backend({ "/v1/me": json(200, user) });
    const auth = new AuthManager({
      baseURL: BASE,
      publishableKey: "pk_test",
      fetch,
      session: "cookie",
    });
    await auth.initialize();
    const me = sent.find((s) => s.url.endsWith("/v1/me"))!;
    expect(me.init.method).toBe("GET");
    expect(new Headers(me.init.headers).get("X-Publishable-Key")).toBe("pk_test");
    auth.destroy();
  });

  it("leaves the token mode as it was", async () => {
    const { fetch, sent } = backend({ "/v1/me": json(200, user) });
    const auth = new AuthManager({ baseURL: BASE, fetch });
    await auth.initialize();
    // Nothing stored means signed out, without asking the backend.
    expect(auth.getState().status).toBe("unauthenticated");
    expect(sent).toHaveLength(0);
  });
});
