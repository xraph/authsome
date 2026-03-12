/**
 * Next.js App Router proxy handler for AuthSome.
 *
 * Proxies all requests from `/api/authsome/[...path]` to the AuthSome
 * backend, forwarding headers, cookies, and request bodies.
 *
 * Usage in `app/api/authsome/[...path]/route.ts`:
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
  // Prefer getSetCookie() (Node 18.14+) which correctly handles
  // multiple Set-Cookie headers without merging them.
  const cookies = backendRes.headers.getSetCookie?.() ?? [];
  if (cookies.length > 0) {
    for (const cookie of cookies) {
      nextRes.headers.append("Set-Cookie", cookie);
    }
    return;
  }

  // Fallback: read raw set-cookie header via get() which may merge
  // multiple values with ", " — but for a single cookie it works fine.
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
 * - Proxies request bodies for non-GET/HEAD methods
 * - Forwards `Set-Cookie` response headers back to the browser
 * - Parses JSON responses with a text fallback
 */
export function createProxyHandler(config: ProxyHandlerConfig) {
  return async function handler(
    request: NextRequest,
    { params }: { params: Promise<{ path: string[] }> },
  ) {
    const { path } = await params;
    const target = `${config.baseURL}/${path.join("/")}`;

    const headers: Record<string, string> = {
      "Content-Type": "application/json",
    };

    // Forward authorization header if present.
    const authHeader = request.headers.get("Authorization");
    if (authHeader) {
      headers["Authorization"] = authHeader;
    }

    // Forward cookies for session-based auth.
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
    });

    const text = await res.text();

    try {
      const data = JSON.parse(text);
      const response = NextResponse.json(data, { status: res.status });
      forwardSetCookies(res, response);
      return response;
    } catch {
      const fallbackResp = new NextResponse(text, { status: res.status });
      forwardSetCookies(res, fallbackResp);
      return fallbackResp;
    }
  };
}
