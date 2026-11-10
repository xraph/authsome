/**
 * OAuth2 consent and connected apps, the client half of authsome's
 * `oauth2provider` server plugin.
 *
 * This is the "App X wants access to your account" screen and the settings
 * list of apps you have authorised. Purpose consent in the data-protection
 * sense is a different plugin, `@authsome/plugin-consent`.
 *
 * ```ts
 * import { oauth2 } from "@authsome/plugin-oauth2";
 *
 * const auth = new AuthManager({ baseURL, plugins: [oauth2()] });
 * const view = await auth.plugin(oauth2).getConsentRequest(cid);
 * ```
 */

import { definePlugin } from "@authsome/ui-core";

/** What the authorize screen has to show. */
export interface ConsentRequest {
  /** Opaque id for this consent round. Goes back with the decision. */
  cid: string;
  client_id: string;
  client_name: string;
  /** Raw scope strings, e.g. "openid", "profile", "offline_access". */
  scopes: string[];
  /** Must go back with the decision. */
  csrf: string;
  action: string;
}

/** Where to send the browser once a decision is recorded. */
export interface ConsentDecisionResult {
  redirect_url: string;
}

/** An app you have authorised. */
export interface OAuthGrant {
  client_id: string;
  client_name: string;
  scopes: string[];
  created_at: string;
  updated_at: string;
}

export interface OAuth2Api {
  /** Loads the consent screen's content for a consent id. */
  getConsentRequest(cid: string): Promise<ConsentRequest>;
  /** Records allow or deny. Returns where to send the browser next. */
  decide(input: {
    cid: string;
    csrf: string;
    decision: "allow" | "deny";
  }): Promise<ConsentDecisionResult>;
  /** Apps the signed-in user has authorised. */
  listGrants(): Promise<{ grants: OAuthGrant[] }>;
  /** Withdraws an app's access. */
  revokeGrant(clientId: string): Promise<void>;
}

export const oauth2 = definePlugin<OAuth2Api>("oauth2provider", (ctx) => ({
  getConsentRequest: (cid) =>
    ctx.request<ConsentRequest>("GET", "/v1/oauth/consent", { query: { cid } }),
  decide: (input) =>
    ctx.request<ConsentDecisionResult>("POST", "/v1/oauth/consent", {
      body: input,
    }),
  listGrants: () =>
    ctx.request<{ grants: OAuthGrant[] }>("GET", "/v1/me/oauth/grants"),
  revokeGrant: (clientId) =>
    ctx.request<void>(
      "DELETE",
      `/v1/me/oauth/grants/${encodeURIComponent(clientId)}`,
    ),
}));

/**
 * Readable names for the scopes authsome issues. A host passes its own map
 * for anything custom, and anything unknown falls back to the raw string so
 * nobody is ever asked to approve a blank line.
 */
export const DEFAULT_SCOPE_LABELS: Record<string, string> = {
  openid: "Confirm who you are",
  profile: "See your name and profile details",
  email: "See your email address",
  phone: "See your phone number",
  address: "See your address",
  offline_access: "Stay signed in when you are away",
};
