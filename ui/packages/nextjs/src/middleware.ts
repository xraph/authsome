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

const SESSION_COOKIE = "authsome:session_token";

/** Configuration for the auth middleware. */
export interface AuthMiddlewareConfig {
  /** Base URL of the AuthSome API. */
  baseURL: string;
  /** Path to redirect unauthenticated users (default: "/sign-in"). */
  signInPage?: string;
  /** Paths that do not require authentication. Supports glob-like prefixes (e.g. "/api/public"). */
  publicPaths?: string[];
  /** Cookie name for the session token (default: "authsome:session_token"). */
  cookieName?: string;
}

/**
 * Creates a Next.js Edge middleware that protects routes behind authentication.
 *
 * - Public paths are served without checks.
 * - Other paths require a valid session token (stored in a cookie).
 * - If the token is missing or invalid the user is redirected to `signInPage`.
 */
export function createAuthMiddleware(config: AuthMiddlewareConfig) {
  const signInPage = config.signInPage ?? "/sign-in";
  const publicPaths = config.publicPaths ?? ["/", signInPage];
  const cookieName = config.cookieName ?? SESSION_COOKIE;

  return async function authMiddleware(request: NextRequest) {
    const { pathname } = request.nextUrl;

    // Allow public paths.
    if (isPublicPath(pathname, publicPaths)) {
      return NextResponse.next();
    }

    // Check for session cookie.
    const sessionToken = request.cookies.get(cookieName)?.value;
    if (!sessionToken) {
      return redirectToSignIn(request, signInPage);
    }

    // Validate the token against the API.
    try {
      const res = await fetch(`${config.baseURL}/v1/me`, {
        headers: { Authorization: `Bearer ${sessionToken}` },
      });

      if (!res.ok) {
        return redirectToSignIn(request, signInPage);
      }
    } catch {
      // Network error — let the request through so the page can handle it.
      return NextResponse.next();
    }

    return NextResponse.next();
  };
}

function isPublicPath(pathname: string, publicPaths: string[]): boolean {
  return publicPaths.some((p) => {
    if (p.endsWith("*")) {
      return pathname.startsWith(p.slice(0, -1));
    }
    return pathname === p;
  });
}

function redirectToSignIn(request: NextRequest, signInPage: string): NextResponse {
  const url = request.nextUrl.clone();
  url.pathname = signInPage;
  url.searchParams.set("redirect", request.nextUrl.pathname);
  return NextResponse.redirect(url);
}
