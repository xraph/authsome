/**
 * Cookie-based TokenStorage for Next.js client components.
 *
 * Stores the session token in a cookie so that Next.js middleware
 * and server components can read it.
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
