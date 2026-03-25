/**
 * Next.js App Router proxy handler for AuthSome.
 *
 * Proxies all requests from your chosen catch-all route to the AuthSome
 * backend, forwarding headers, cookies, query parameters, and request bodies.
 * Handles OAuth redirect responses and non-JSON content (e.g. HTML callback pages).
 *
 * Usage in `app/api/auth/[...path]/route.ts`:
 * ```ts
 * export { GET, POST, PUT, DELETE, PATCH } from "@authsome/ui-nextjs/proxy";
 * ```
 *
 * Or with custom config:
 * ```ts
 * import { createProxyHandler } from "@authsome/ui-nextjs/proxy";
 *
 * const handler = createProxyHandler({
 *   baseURL: process.env.NEXT_PUBLIC_AUTHSOME_API_URL!,
 * });
 *
 * export { handler as GET, handler as POST, handler as PUT, handler as DELETE, handler as PATCH };
 * ```
 */

import { type NextRequest, NextResponse } from "next/server";

/** Configuration for the proxy handler. */
export interface ProxyHandlerConfig {
  /** Base URL of the AuthSome backend API (e.g. "http://localhost:7900"). */
  baseURL: string;
}

/**
 * Forward Set-Cookie headers from the backend response to the Next.js
 * response so httpOnly session cookies reach the browser.
 */
function forwardSetCookies(
  backendRes: Response,
  nextRes: NextResponse | Response,
): void {
  const cookies = backendRes.headers.getSetCookie?.() ?? [];
  if (cookies.length > 0) {
    for (const cookie of cookies) {
      nextRes.headers.append("Set-Cookie", cookie);
    }
    return;
  }

  const raw = backendRes.headers.get("set-cookie");
  if (raw) {
    nextRes.headers.append("Set-Cookie", raw);
  }
}

/**
 * Creates a Next.js App Router catch-all route handler that proxies
 * requests to the AuthSome backend API.
 *
 * The handler:
 * - Forwards `Authorization` and `Cookie` headers
 * - Forwards query parameters (needed for OAuth callbacks)
 * - Proxies request bodies for non-GET/HEAD methods
 * - Forwards `Set-Cookie` response headers back to the browser
 * - Handles redirect responses (OAuth flows) with `redirect: 'manual'`
 * - Preserves non-JSON content types (e.g. HTML callback pages)
 */
export function createProxyHandler(config: ProxyHandlerConfig) {
  return async function handler(
    request: NextRequest,
    { params }: { params: Promise<{ path: string[] }> },
  ) {
    const { path } = await params;
    const queryString = request.nextUrl.search;
    const target = `${config.baseURL}/${path.join("/")}${queryString}`;

    const headers: Record<string, string> = {
      "Content-Type": "application/json",
    };

    const authHeader = request.headers.get("Authorization");
    if (authHeader) {
      headers["Authorization"] = authHeader;
    }

    const cookie = request.headers.get("Cookie");
    if (cookie) {
      headers["Cookie"] = cookie;
    }

    const res = await fetch(target, {
      method: request.method,
      headers,
      body:
        request.method !== "GET" && request.method !== "HEAD"
          ? await request.text()
          : undefined,
      redirect: "manual",
    });

    // Forward redirect responses (e.g. OAuth social login redirects).
    if (res.status >= 300 && res.status < 400) {
      const location = res.headers.get("Location");
      if (location) {
        const response = NextResponse.redirect(
          location,
          res.status as 301 | 302 | 303 | 307 | 308,
        );
        forwardSetCookies(res, response);
        return response;
      }
    }

    const contentType = res.headers.get("Content-Type") ?? "";
    const text = await res.text();

    // Pass through non-JSON responses (e.g. HTML callback pages) with
    // the original Content-Type so the browser renders them correctly.
    if (!contentType.includes("application/json")) {
      const response = new NextResponse(text, {
        status: res.status,
        headers: { "Content-Type": contentType },
      });
      forwardSetCookies(res, response);
      return response;
    }

    try {
      const data = JSON.parse(text);
      const response = NextResponse.json(data, { status: res.status });
      forwardSetCookies(res, response);
      return response;
    } catch {
      const response = new NextResponse(text, { status: res.status });
      forwardSetCookies(res, response);
      return response;
    }
  };
}

/**
 * Default proxy handler using `NEXT_PUBLIC_AUTHSOME_API_URL` env var.
 * Export this directly from your route file for zero-config setup:
 *
 * ```ts
 * // app/api/auth/[...path]/route.ts
 * export { GET, POST, PUT, DELETE, PATCH } from "@authsome/ui-nextjs/proxy";
 * ```
 */
const defaultHandler = createProxyHandler({
  baseURL: (typeof process !== "undefined" ? process.env?.NEXT_PUBLIC_AUTHSOME_API_URL : undefined) ?? "",
});

export {
  defaultHandler as GET,
  defaultHandler as POST,
  defaultHandler as PUT,
  defaultHandler as DELETE,
  defaultHandler as PATCH,
};
