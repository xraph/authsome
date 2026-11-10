/**
 * GDPR purpose consent, the client half of authsome's `consent` server plugin.
 *
 * This is consent in the data-protection sense: a record that someone agreed
 * to a named purpose at a given version, with a timestamp and the IP it came
 * from. It has nothing to do with the OAuth "App X wants access to your
 * account" screen, which belongs to `oauth2provider` and lives in
 * `@authsome/plugin-oauth2`.
 *
 * ```ts
 * import { consent } from "@authsome/plugin-consent";
 *
 * const auth = new AuthManager({ baseURL, plugins: [consent()] });
 * await auth.plugin(consent).grant({ purpose: "marketing_email", version: "2026-01" });
 * ```
 */

import { definePlugin } from "@authsome/ui-core";

/** One consent record. */
export interface ConsentRecord {
  id: string;
  user_id: string;
  app_id: string;
  /** The thing agreed to, e.g. "marketing_email". */
  purpose: string;
  /** Which wording was agreed to. Bump it when the wording changes. */
  version: string;
  granted: boolean;
  granted_at: string;
  revoked_at?: string;
  ip_address: string;
  created_at: string;
  updated_at: string;
}

export interface ConsentListResult {
  consents: ConsentRecord[];
  next_cursor?: string;
}

export interface GrantConsentInput {
  purpose: string;
  /** Record which wording was agreed to. Strongly recommended. */
  version?: string;
  app_id?: string;
}

export interface RevokeConsentInput {
  purpose: string;
  app_id?: string;
}

export interface ConsentApi {
  /** Every consent record for the signed-in user, newest first. */
  list(opts?: {
    purpose?: string;
    cursor?: string;
    limit?: number;
  }): Promise<ConsentListResult>;
  grant(input: GrantConsentInput): Promise<ConsentRecord>;
  revoke(input: RevokeConsentInput): Promise<{ status: string }>;
}

export const consent = definePlugin<ConsentApi>("consent", (ctx) => ({
  list: (opts) =>
    ctx.request<ConsentListResult>("GET", "/v1/consent", {
      query: {
        purpose: opts?.purpose,
        cursor: opts?.cursor,
        limit: opts?.limit,
      },
    }),
  grant: (input) =>
    ctx.request<ConsentRecord>("POST", "/v1/consent/grant", { body: input }),
  revoke: (input) =>
    ctx.request<{ status: string }>("POST", "/v1/consent/revoke", {
      body: input,
    }),
}));
