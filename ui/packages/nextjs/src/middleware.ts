/**
 * Next.js Edge middleware for AuthSome authentication.
 *
 * Usage in `middleware.ts`:
 * ```ts
 * import { createAuthMiddleware } from "@authsome/ui-nextjs/middleware";
 *
 * export default createAuthMiddleware({
 *   baseURL: process.env.AUTHSOME_API_URL!,
 *   signInPage: "/sign-in",
 *   publicPaths: ["/", "/sign-in", "/sign-up", "/api/public"],
 * });
 *
 * export const config = { matcher: ["/((?!_next/static|_next/image|favicon.ico).*)"] };
 * ```
 */

import type { NextRequest } from "next/server";
import { NextResponse } from "next/server";

import { readSessionToken, SESSION_COOKIE } from "./session-cookie";

/** Configuration for the auth middleware. */
export interface AuthMiddlewareConfig {
  /** Base URL of the AuthSome API. */
  baseURL: string;
  /** Path to redirect unauthenticated users (default: "/sign-in"). */
  signInPage?: string;
  /** Paths that do not require authentication. Supports glob-like prefixes (e.g. "/api/public*"). */
  publicPaths?: string[];
  /** Cookie name for the session token (default: "authsome_session_token"). */
  cookieName?: string;
  /**
   * Paths that should redirect authenticated users away (e.g. sign-in, sign-up pages).
   * Supports glob-like prefixes. By default includes `signInPage` and "/sign-up".
   */
  authPaths?: string[];
  /** Where to redirect authenticated users visiting auth pages (default: "/"). */
  afterSignInUrl?: string;
  /**
   * What to do when the session cannot be validated because the auth API is
   * unreachable or errored — as distinct from it rejecting the session.
   *
   * "deny" (default) redirects to sign-in. "allow" serves the request.
   *
   * "allow" trades authentication for availability, and the trade is worse
   * than it looks: a caller can set any cookie value they like, so during an
   * outage — or against a misconfigured baseURL that 404s every request —
   * "allow" serves protected routes to anyone. Choose it only if you have a
   * second enforcement point behind this one.
   */
  onUnavailable?: "deny" | "allow";
  /**
   * Milliseconds to wait for the validation request (default: 5000).
   *
   * Middleware runs on every matched request, so an auth API that hangs
   * without this stalls the whole site rather than failing a check.
   */
  timeoutMs?: number;
}

/** Outcome of validating a session token against the auth API. */
type SessionCheck = "valid" | "rejected" | "unavailable";

/**
 * Creates a Next.js Edge middleware that protects routes behind authentication.
 *
 * - Auth pages redirect authenticated users to `afterSignInUrl`.
 * - Public paths are served without checks.
 * - Other paths require a session the auth API confirms.
 * - A missing, rejected, or unverifiable session redirects to `signInPage`.
 *
 * Only an explicit 2xx from the auth API counts as authenticated. Previously
 * any response other than 401 was let through, so a 404 from a wrong baseURL,
 * a 500, or a network error served every protected route to anyone holding an
 * arbitrary cookie value. Set `onUnavailable: "allow"` to restore that
 * behaviour deliberately.
 */
export function createAuthMiddleware(config: AuthMiddlewareConfig) {
  const signInPage = config.signInPage ?? "/sign-in";
  const publicPaths = config.publicPaths ?? ["/", signInPage];
  const cookieName = config.cookieName ?? SESSION_COOKIE;
  const afterSignInUrl = config.afterSignInUrl ?? "/";
  const authPaths = config.authPaths ?? [signInPage, "/sign-up"];

  const onUnavailable = config.onUnavailable ?? "deny";
  const timeoutMs = config.timeoutMs ?? 5000;

  /**
   * Asks the auth API whether this token is currently good.
   *
   * A verdict requires an answer: 2xx is valid, 401/403 is rejected, and
   * anything else — other statuses, a network failure, a timeout — is
   * "unavailable", meaning no verdict was obtained. Reading a non-401 as
   * success is what let a wrong baseURL authenticate everyone.
   */
  async function checkSession(token: string): Promise<SessionCheck> {
    try {
      const res = await fetch(`${config.baseURL}/v1/me`, {
        headers: { Authorization: `Bearer ${token}` },
        signal: AbortSignal.timeout(timeoutMs),
      });
      if (res.ok) {
        return "valid";
      }
      if (res.status === 401 || res.status === 403) {
        return "rejected";
      }
      return "unavailable";
    } catch {
      return "unavailable";
    }
  }

  return async function authMiddleware(request: NextRequest) {
    const { pathname } = request.nextUrl;
    // Accepts either the backend's httpOnly cookie or a session persisted by
    // createCookieStorage — see readSessionToken.
    const sessionToken = readSessionToken(request.cookies, cookieName);

    // Auth pages: bounce a confirmed-signed-in user away. Only a positive
    // verdict redirects — an unverifiable session must not eject someone from
    // the very page they would use to sign in again.
    if (sessionToken && matchesPath(pathname, authPaths)) {
      if ((await checkSession(sessionToken)) === "valid") {
        const url = request.nextUrl.clone();
        url.pathname = afterSignInUrl;
        url.search = "";
        return NextResponse.redirect(url);
      }
    }

    // The sign-in page is always reachable, whether or not the caller listed
    // it in publicPaths. Redirecting it to itself is an infinite loop, and it
    // is the one page a user with an unusable session must be able to reach.
    if (pathname === signInPage) {
      return NextResponse.next();
    }

    // Allow public paths.
    if (matchesPath(pathname, publicPaths)) {
      return NextResponse.next();
    }

    if (!sessionToken) {
      return redirectToSignIn(request, signInPage);
    }

    switch (await checkSession(sessionToken)) {
      case "valid":
        return NextResponse.next();
      case "rejected":
        return redirectToSignIn(request, signInPage);
      case "unavailable":
        // No verdict. Fail closed unless the operator opted out.
        return onUnavailable === "allow"
          ? NextResponse.next()
          : redirectToSignIn(request, signInPage);
    }
  };
}

/**
 * Reports whether pathname is covered by one of the configured patterns.
 *
 * A trailing "*" matches the prefix and anything beneath it, but only at a
 * path-segment boundary: "/api/public*" covers "/api/public" and
 * "/api/public/keys", and does NOT cover "/api/publicadmin".
 *
 * Plain string containment was the earlier behaviour, and it silently widened
 * publicPaths — one entry meant to expose "/api/public" also exposed every
 * sibling route sharing that prefix, with no way to tell from the config that
 * it had happened.
 *
 * Trailing slashes are normalized so "/dashboard" and "/dashboard/" are the
 * same route, which is how Next.js resolves them.
 */
function matchesPath(pathname: string, paths: string[]): boolean {
  const target = normalizePath(pathname);
  return paths.some((pattern) => {
    if (pattern.endsWith("*")) {
      const prefix = normalizePath(pattern.slice(0, -1));
      // A root wildcard ("*" or "/*") covers every path.
      if (prefix === "/") {
        return true;
      }
      return target === prefix || target.startsWith(prefix + "/");
    }
    return target === normalizePath(pattern);
  });
}

/** Strips trailing slashes, preserving the root path as "/". */
function normalizePath(p: string): string {
  const trimmed = p.replace(/\/+$/, "");
  return trimmed === "" ? "/" : trimmed;
}

function redirectToSignIn(request: NextRequest, signInPage: string): NextResponse {
  const url = request.nextUrl.clone();
  url.pathname = signInPage;
  url.searchParams.set("redirect", request.nextUrl.pathname);
  return NextResponse.redirect(url);
}
