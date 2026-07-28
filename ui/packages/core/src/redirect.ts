/**
 * Redirect-target validation.
 *
 * Sign-in and sign-up flows accept a `?redirect=` parameter so a user who was
 * bounced to the auth page lands back where they started. That parameter is
 * attacker-controlled: without validation, `?redirect=https://evil.com` sends
 * the user to a hostile page *immediately after they authenticate*, which is
 * the most convincing possible moment to present a fake "confirm your
 * password" screen. It can also carry `javascript:` payloads.
 *
 * The check here resolves the candidate against the current origin and
 * compares origins, rather than inspecting the string. String checks are the
 * usual approach and they leak: `startsWith("/")` accepts `//evil.com` and
 * `/\evil.com`, both of which browsers resolve to a different host.
 */

/** Schemes a redirect may use. Everything else (javascript:, data:, blob:) is refused. */
const ALLOWED_PROTOCOLS = new Set(["http:", "https:"]);

/** Options controlling which redirect targets are acceptable. */
export interface SafeRedirectOptions {
  /**
   * Origin the app is currently served from — `window.location.origin` in a
   * browser. Redirects to this origin are always allowed.
   */
  currentOrigin: string;
  /**
   * Extra origins accepted as redirect targets, for deployments where the auth
   * UI and the app live on different hosts. Empty by default: same-origin only.
   *
   * Entries are matched **exactly**, by origin. Wildcards and subdomain
   * patterns are deliberately unsupported — every tenant host must be listed
   * in full. Pattern matching here is a well-known source of account-takeover
   * bugs, and an explicit list is worth the verbosity.
   *
   * Each entry must be an absolute URL ("https://portal.example.com"); it is
   * reduced to its origin, so a trailing slash, a path, or a capitalized host
   * are all tolerated. Entries that cannot be parsed are ignored and will
   * never match.
   */
  allowedOrigins?: string[];
}

/**
 * Reports whether `candidate` is safe to navigate to after authentication.
 *
 * Accepts relative paths ("/dashboard", "/a/b?x=1#f") and absolute URLs whose
 * origin is the current origin or an allowed one. Refuses everything else,
 * including a malformed candidate.
 */
export function isSafeRedirect(
  candidate: string | null | undefined,
  options: SafeRedirectOptions,
): boolean {
  if (typeof candidate !== "string" || candidate.trim() === "") {
    return false;
  }

  let resolved: URL;
  try {
    // Resolving against currentOrigin collapses the evasion cases —
    // "//evil.com", "/\evil.com", "\thttps://evil.com" — onto their real
    // origin, where the comparison below catches them.
    resolved = new URL(candidate, options.currentOrigin);
  } catch {
    return false;
  }

  // Opaque-origin schemes (javascript:, data:) never reach the comparison.
  if (!ALLOWED_PROTOCOLS.has(resolved.protocol)) {
    return false;
  }

  if (resolved.origin === options.currentOrigin) {
    return true;
  }

  // Exact origin match against the allowlist — no wildcards by design.
  // Entries are normalized so config formatting can't cause a silent miss.
  return (options.allowedOrigins ?? []).some(
    (allowed) => toOrigin(allowed) === resolved.origin,
  );
}

/**
 * Reduces an allowlist entry to its canonical origin, or `null` when it isn't
 * a parseable absolute URL. Returning `null` keeps a malformed entry inert:
 * it matches nothing rather than matching everything.
 */
function toOrigin(value: string): string | null {
  try {
    const { origin } = new URL(value);
    // Opaque-origin schemes (data:, javascript:) stringify to "null" and must
    // never be treated as a real origin to compare against.
    return origin === "null" ? null : origin;
  } catch {
    return null;
  }
}

/**
 * Returns `candidate` when it passes {@link isSafeRedirect}, otherwise
 * `fallback`. Use this at navigation sites so an unsafe value degrades to a
 * known-good destination instead of blocking the user.
 */
export function safeRedirectTarget(
  candidate: string | null | undefined,
  fallback: string,
  options: SafeRedirectOptions,
): string {
  return isSafeRedirect(candidate, options) ? (candidate as string) : fallback;
}

