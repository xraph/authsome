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
  /** Abort an upstream request after this many ms (default: 30_000). */
  timeoutMs?: number;
}

/**
 * Request headers relayed to the backend beyond auth.
 *
 * The client-identity headers matter for security, not just telemetry: the
 * backend keys rate limiting, account lockout, geo/anomaly scoring, and its
 * audit log off the caller's IP and user agent. Drop them and every proxied
 * request arrives wearing this server's identity, collapsing all users into a
 * single rate-limit bucket and disabling brute-force protection.
 *
 * `x-forwarded-*` is relayed as received. That is correct when this app sits
 * behind a load balancer or CDN (Vercel, Cloudflare) that overwrites the
 * header with the true client IP. If you expose Next.js directly to the
 * internet, a caller can forge `X-Forwarded-For` and evade the backend's
 * limiter — terminate on a proxy that rewrites it, or have the backend trust
 * only its own peer address.
 */
const FORWARDED_REQUEST_HEADERS = [
  "user-agent",
  "accept-language",
  "x-forwarded-for",
  "x-forwarded-host",
  "x-forwarded-proto",
  "x-real-ip",
] as const;

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

    const headers: Record<string, string> = {};

    const authHeader = request.headers.get("Authorization");
    if (authHeader) {
      headers["Authorization"] = authHeader;
    }

    const cookie = request.headers.get("Cookie");
    if (cookie) {
      headers["Cookie"] = cookie;
    }

    // Relay client identity so backend rate limiting and lockout see the real
    // caller rather than this server. See FORWARDED_REQUEST_HEADERS.
    for (const name of FORWARDED_REQUEST_HEADERS) {
      const value = request.headers.get(name);
      if (value) {
        headers[name] = value;
      }
    }

    const hasBody = request.method !== "GET" && request.method !== "HEAD";
    if (hasBody) {
      // Pass the caller's own Content-Type through. Forcing application/json
      // here corrupted multipart uploads and form posts.
      headers["Content-Type"] =
        request.headers.get("Content-Type") ?? "application/json";
    }

    const timeoutMs = config.timeoutMs ?? 30_000;
    const abort = AbortSignal.timeout(timeoutMs);

    const res = await fetch(target, {
      method: request.method,
      headers,
      body: hasBody ? await request.text() : undefined,
      redirect: "manual",
      signal: abort,
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
    //
    // This route is mounted on the app's own origin, so anything the backend
    // returns as HTML executes with access to this origin's cookies and
    // storage. nosniff stops the browser from upgrading an unlabelled or
    // mislabelled body into HTML or script on its own.
    if (!contentType.includes("application/json")) {
      const response = new NextResponse(text, {
        status: res.status,
        headers: {
          "Content-Type": contentType || "text/plain; charset=utf-8",
          "X-Content-Type-Options": "nosniff",
        },
      });
      forwardSetCookies(res, response);
      return response;
    }

    try {
      const data = JSON.parse(text);
      const response = NextResponse.json(data, { status: res.status });
      response.headers.set("X-Content-Type-Options", "nosniff");
      forwardSetCookies(res, response);
      return response;
    } catch {
      // Labelled JSON that isn't JSON — serve it as inert text, never as
      // whatever the browser guesses it might be.
      const response = new NextResponse(text, {
        status: res.status,
        headers: {
          "Content-Type": "text/plain; charset=utf-8",
          "X-Content-Type-Options": "nosniff",
        },
      });
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
