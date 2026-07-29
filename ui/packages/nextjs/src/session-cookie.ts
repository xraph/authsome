/**
 * Resolving the session token from cookies, server-side.
 *
 * There are two ways a token reaches the browser's cookie jar, and middleware
 * and server components have to handle both:
 *
 *  1. The backend sets an httpOnly cookie (named by `authsome_session_token`
 *     by default) carrying the bare token. This is the preferred path — the
 *     token is unreadable to JavaScript, so XSS cannot exfiltrate it.
 *
 *  2. `createCookieStorage()` persists the whole Session as JSON under the
 *     ui-core storage key, so the value is an object, not a token, and the
 *     cookie name is the percent-encoded storage key.
 *
 * Previously only (1) was read, while the doc on `createCookieStorage` promised
 * it existed so "middleware and server components can read it". They could not:
 * the names never matched, and even forcing the name through `cookieName` would
 * have handed `Bearer <json>` to the API and redirect-looped a signed-in user.
 */

import { SESSION_STORAGE_KEY } from "@authsome/ui-core";

/** Default name of the backend-set httpOnly session cookie. */
export const SESSION_COOKIE = "authsome_session_token";

/**
 * Cookie name written by `createCookieStorage()`.
 *
 * It mirrors that helper's own key escaping — characters outside
 * [A-Za-z0-9_-] are percent-encoded — so ":" becomes "%3A".
 */
export const SESSION_STORAGE_COOKIE = SESSION_STORAGE_KEY.replace(
  /[^a-zA-Z0-9_-]/g,
  encodeURIComponent,
);

/** Minimal cookie accessor, satisfied by both NextRequest.cookies and the
 * `cookies()` store from next/headers. */
export interface CookieReader {
  get(name: string): { value: string } | undefined;
}

/**
 * Returns the session token from either cookie shape, or undefined.
 *
 * The httpOnly cookie wins when both are present: it is the one the backend
 * controls and the one JavaScript cannot have tampered with.
 */
export function readSessionToken(
  cookies: CookieReader,
  cookieName: string = SESSION_COOKIE,
): string | undefined {
  const direct = cookies.get(cookieName)?.value;
  if (direct) {
    return direct;
  }

  const stored = cookies.get(SESSION_STORAGE_COOKIE)?.value;
  if (!stored) {
    return undefined;
  }
  return parseStoredSessionToken(stored);
}

/**
 * Extracts session_token from a JSON Session persisted by createCookieStorage.
 * Returns undefined for anything that isn't a decodable session — a malformed
 * cookie must read as "no session", never as a token.
 */
function parseStoredSessionToken(raw: string): string | undefined {
  try {
    const parsed: unknown = JSON.parse(decodeURIComponent(raw));
    if (
      typeof parsed === "object" &&
      parsed !== null &&
      "session_token" in parsed &&
      typeof (parsed as { session_token: unknown }).session_token === "string"
    ) {
      const token = (parsed as { session_token: string }).session_token;
      return token || undefined;
    }
  } catch {
    // Not JSON, or not percent-decodable.
  }
  return undefined;
}
