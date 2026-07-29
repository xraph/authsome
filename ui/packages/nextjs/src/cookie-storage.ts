/**
 * Cookie-based TokenStorage for Next.js client components.
 *
 * Persists the whole Session as a cookie so Next.js middleware and server
 * components can read it via `readSessionToken`, which understands this shape
 * as well as the backend's own httpOnly cookie.
 *
 * Prefer the backend cookie where you can. A cookie written from JavaScript
 * cannot be httpOnly, so the refresh token it carries is readable by any
 * script on the page — use this only when the backend is not setting session
 * cookies for you.
 */

import type { TokenStorage } from "@authsome/ui-core";

/** Options for cookie storage. */
export interface CookieStorageOptions {
  /** Cookie path (default: "/"). */
  path?: string;
  /** SameSite attribute (default: "lax"). */
  sameSite?: "strict" | "lax" | "none";
  /** Whether to set the Secure flag (default: true in production). */
  secure?: boolean;
  /** Max-Age in seconds (default: 30 days). */
  maxAge?: number;
}

/**
 * Creates a TokenStorage backed by document.cookie.
 *
 * ```tsx
 * <AuthProvider
 *   baseURL="..."
 *   storage={createCookieStorage()}
 * >
 * ```
 */
export function createCookieStorage(opts: CookieStorageOptions = {}): TokenStorage {
  const path = opts.path ?? "/";
  const sameSite = opts.sameSite ?? "lax";
  const secure = opts.secure ?? (typeof location !== "undefined" && location.protocol === "https:");
  const maxAge = opts.maxAge ?? 30 * 24 * 60 * 60;

  return {
    getItem(key: string): string | null {
      if (typeof document === "undefined") return null;
      const match = document.cookie.match(new RegExp(`(?:^|;\\s*)${escapeKey(key)}=([^;]*)`));
      return match ? decodeURIComponent(match[1]) : null;
    },

    setItem(key: string, value: string): void {
      if (typeof document === "undefined") return;
      const parts = [
        `${escapeKey(key)}=${encodeURIComponent(value)}`,
        `path=${path}`,
        `max-age=${maxAge}`,
        `samesite=${sameSite}`,
      ];
      if (secure) parts.push("secure");
      document.cookie = parts.join("; ");
    },

    removeItem(key: string): void {
      if (typeof document === "undefined") return;
      document.cookie = `${escapeKey(key)}=; path=${path}; max-age=0`;
    },
  };
}

function escapeKey(key: string): string {
  return key.replace(/[^a-zA-Z0-9_-]/g, encodeURIComponent);
}
