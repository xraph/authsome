/**
 * Opens a centered popup window for OAuth authentication.
 * Returns the popup Window reference, or null if blocked by the browser.
 */
export function openOAuthPopup(
  url: string,
  name = "oauth-login",
  width = 500,
  height = 600,
): Window | null {
  const left = window.screenX + (window.outerWidth - width) / 2;
  const top = window.screenY + (window.outerHeight - height) / 2;
  return window.open(
    url,
    name,
    `width=${width},height=${height},left=${left},top=${top},popup=yes`,
  );
}

/**
 * Initiates a social OAuth login flow using the Authsome API.
 *
 * 1. Calls `startOAuth` to get the provider's authorization URL.
 * 2. Opens the URL in a popup window (falls back to full-page redirect
 *    if the popup is blocked).
 * 3. Polls for the popup to close, then invokes `onComplete`.
 *
 * The backend sets an httpOnly session cookie during the OAuth callback,
 * so the caller typically just needs to redirect/reload after completion.
 */
export async function handleSocialLogin(
  // The codegen mis-types this endpoint's response: the social-OAuth
  // Start{Request,Response} Go structs collide with the phone-auth pair and
  // the wrong shape wins, so the declared response loses `auth_url`. It is
  // cast at the assertion site below.
  //
  // The request is not a body. startOAuth takes frontend_url and redirect_url
  // positionally and puts them in the query string, which is what
  // plugins/social/plugin.go reads (both are `query:` tagged there).
  client: {
    startOAuth: (
      provider: string,
      frontendUrl?: string,
      redirectUrl?: string,
    ) => Promise<unknown>;
  },
  providerId: string,
  onComplete: () => void,
  onError?: (err: unknown) => void,
): Promise<void> {
  try {
    // redirect_url is the post-auth return target. frontend_url is left unset
    // so the backend falls back to the Origin/Referer it already trusts,
    // rather than a value this function would be asserting.
    const res = (await client.startOAuth(
      providerId,
      undefined,
      window.location.href,
    )) as { auth_url: string };
    const { auth_url } = res;

    const popup = openOAuthPopup(auth_url);

    if (!popup) {
      // Popup was blocked — fall back to full-page redirect.
      window.location.href = auth_url;
      return;
    }

    const poll = setInterval(() => {
      if (!popup.closed) return;
      clearInterval(poll);
      // The backend set the httpOnly session cookie during the
      // callback. Just invoke onComplete so the consumer can
      // redirect or reload.
      onComplete();
    }, 500);
  } catch (err) {
    onError?.(err);
  }
}
