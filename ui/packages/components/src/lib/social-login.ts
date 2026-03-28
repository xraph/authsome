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
  client: { startOAuth: (provider: string, body: { Provider: string; redirect_url?: string }) => Promise<{ auth_url: string }> },
  providerId: string,
  onComplete: () => void,
  onError?: (err: unknown) => void,
): Promise<void> {
  try {
    const { auth_url } = await client.startOAuth(providerId, {
      Provider: providerId,
      redirect_url: window.location.href,
    });

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
