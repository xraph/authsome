import { afterEach, describe, expect, it, vi } from "vitest";

import { createAuthMiddleware } from "./middleware";
import { SESSION_COOKIE } from "./session-cookie";

/** Minimal NextRequest stand-in: middleware only uses nextUrl and cookies. */
function req(pathname: string, cookies: Record<string, string> = {}) {
  const url = new URL(`https://app.test${pathname}`);
  return {
    nextUrl: Object.assign(url, { clone: () => new URL(url.toString()) }),
    cookies: {
      get(name: string) {
        const value = cookies[name];
        return value === undefined ? undefined : { value };
      },
    },
  } as never;
}

/** Stubs the auth API's verdict for /v1/me. */
function stubAuthAPI(reply: { status?: number; reject?: boolean }) {
  vi.stubGlobal("fetch", async () => {
    if (reply.reject) {
      throw new Error("network down");
    }
    const status = reply.status ?? 200;
    return { ok: status >= 200 && status < 300, status } as Response;
  });
}

const mw = (over: Partial<Parameters<typeof createAuthMiddleware>[0]> = {}) =>
  createAuthMiddleware({
    baseURL: "https://auth.test",
    publicPaths: ["/"],
    ...over,
  });

const withSession = () => req("/dashboard", { [SESSION_COOKIE]: "tok" });

function isRedirectToSignIn(res: Response): boolean {
  const loc = res.headers.get("location") ?? "";
  return res.status >= 300 && res.status < 400 && loc.includes("/sign-in");
}

afterEach(() => vi.unstubAllGlobals());

describe("createAuthMiddleware", () => {
  it("serves a protected route when the API confirms the session", async () => {
    stubAuthAPI({ status: 200 });
    const res = await mw()(withSession());
    expect(isRedirectToSignIn(res)).toBe(false);
  });

  it("redirects when the API rejects the session", async () => {
    stubAuthAPI({ status: 401 });
    expect(isRedirectToSignIn(await mw()(withSession()))).toBe(true);
  });

  // Only 401 used to redirect, so every one of these served protected routes
  // to anyone holding an arbitrary cookie value.
  describe("fails closed when no verdict is obtained", () => {
    it.each([
      ["403 forbidden", { status: 403 }],
      ["404 — wrong baseURL", { status: 404 }],
      ["500 — auth API erroring", { status: 500 }],
      ["502 — gateway down", { status: 502 }],
      ["network failure", { reject: true }],
    ])("%s", async (_label, reply) => {
      stubAuthAPI(reply);
      expect(isRedirectToSignIn(await mw()(withSession()))).toBe(true);
    });
  });

  it("serves the request on failure only when explicitly opted in", async () => {
    stubAuthAPI({ reject: true });
    const res = await mw({ onUnavailable: "allow" })(withSession());
    expect(isRedirectToSignIn(res)).toBe(false);
  });

  it("still allows public paths without any check", async () => {
    stubAuthAPI({ reject: true });
    const res = await mw()(req("/"));
    expect(isRedirectToSignIn(res)).toBe(false);
  });

  it("redirects a request carrying no session", async () => {
    stubAuthAPI({ status: 200 });
    expect(isRedirectToSignIn(await mw()(req("/dashboard")))).toBe(true);
  });

  // An unverifiable session must not eject someone from the page they would
  // use to sign in again. Note publicPaths here deliberately omits /sign-in:
  // failing closed turns a missing entry into an infinite redirect loop, so
  // the sign-in page is unconditionally reachable.
  it("does not bounce off the sign-in page when the API is unreachable", async () => {
    stubAuthAPI({ reject: true });
    const res = await mw()(req("/sign-in", { [SESSION_COOKIE]: "tok" }));
    expect(res.status).toBeLessThan(300);
  });

  it("never redirects the sign-in page to itself", async () => {
    stubAuthAPI({ status: 500 });
    const res = await mw({ publicPaths: [] })(
      req("/sign-in", { [SESSION_COOKIE]: "tok" }),
    );
    expect(res.status).toBeLessThan(300);
  });

  it("bounces a confirmed-signed-in user off the sign-in page", async () => {
    stubAuthAPI({ status: 200 });
    const res = await mw({ afterSignInUrl: "/home" })(
      req("/sign-in", { [SESSION_COOKIE]: "tok" }),
    );
    expect(res.status).toBeGreaterThanOrEqual(300);
    expect(res.headers.get("location")).toContain("/home");
  });
});

describe("path matching", () => {
  /** A path is "public" iff the request is served without a check. */
  async function isPublic(paths: string[], pathname: string): Promise<boolean> {
    stubAuthAPI({ reject: true }); // any check would fail closed
    const res = await mw({ publicPaths: paths })(req(pathname, {}));
    return !isRedirectToSignIn(res);
  }

  // The bug: one entry exposing /api/public also exposed every sibling route
  // sharing that prefix, invisibly.
  it("does not let a prefix leak into a sibling route", async () => {
    expect(await isPublic(["/api/public*"], "/api/publicadmin")).toBe(false);
    expect(await isPublic(["/api/public*"], "/api/public-internal")).toBe(false);
  });

  it("covers the prefix itself and everything beneath it", async () => {
    expect(await isPublic(["/api/public*"], "/api/public")).toBe(true);
    expect(await isPublic(["/api/public*"], "/api/public/keys")).toBe(true);
    expect(await isPublic(["/api/public*"], "/api/public/a/b")).toBe(true);
  });

  it("matches exact patterns exactly", async () => {
    expect(await isPublic(["/health"], "/health")).toBe(true);
    expect(await isPublic(["/health"], "/healthz")).toBe(false);
    expect(await isPublic(["/health"], "/health/deep")).toBe(false);
  });

  it("treats trailing slashes as the same route", async () => {
    expect(await isPublic(["/health"], "/health/")).toBe(true);
    expect(await isPublic(["/api/public*"], "/api/public/")).toBe(true);
  });

  it("keeps the root path exact, not a prefix for everything", async () => {
    expect(await isPublic(["/"], "/")).toBe(true);
    expect(await isPublic(["/"], "/dashboard")).toBe(false);
  });

  it("supports a root wildcard", async () => {
    expect(await isPublic(["/*"], "/anything/at/all")).toBe(true);
  });
});
