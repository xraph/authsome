import { safeRedirectTarget } from "@authsome/ui-core";

/**
 * Where redirectAfterAuth sends a page served from `origin` whose query
 * string is `search`.
 *
 * `?redirect=` is attacker-controllable, and it's followed at the moment a
 * hostile page would be most convincing. safeRedirectTarget refuses anything
 * that doesn't resolve to this origin. The URL is then rebuilt on `origin`,
 * so the host always comes from the page you're on and only the path, query
 * and fragment come from the parameter.
 */
export function redirectAfterAuthTarget(origin: string, search: string): string {
  const requested = new URLSearchParams(search).get("redirect");
  const target = new URL(
    safeRedirectTarget(requested, "/", { currentOrigin: origin }),
    origin,
  );
  return origin + target.pathname + target.search + target.hash;
}

/**
 * Sends the browser on once sign-in, sign-up or SSO has finished: to the page
 * named by `?redirect=`, or to "/" when there isn't one or it points off-site.
 */
export function redirectAfterAuth(): void {
  window.location.href = redirectAfterAuthTarget(
    window.location.origin,
    window.location.search,
  );
}
