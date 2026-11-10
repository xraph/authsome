/**
 * SCIM administration, the client half of authsome's `scim` server plugin.
 *
 * This is the dashboard side: connect an identity provider, mint the bearer
 * token it will authenticate with, rotate it, revoke it. It is not the SCIM
 * 2.0 protocol itself, which the IdP calls server to server with one of
 * these tokens and which no browser ever touches.
 *
 * ```ts
 * import { scim } from "@authsome/plugin-scim";
 *
 * const auth = new AuthManager({ baseURL, plugins: [scim()] });
 * const { configs } = await auth.plugin(scim).listConfigs(orgId);
 * ```
 *
 * Every route here needs an organization owner or admin.
 */

import { definePlugin } from "@authsome/ui-core";

/** One identity provider connection. */
export interface ScimConfig {
  id: string;
  app_id: string;
  org_id: string;
  /** What this connection is called, usually the IdP's name. */
  name: string;
  /** Whether provisioning is accepted right now. */
  enabled: boolean;
  /** Create users the IdP pushes for the first time. */
  auto_create: boolean;
  /** Suspend users the IdP deactivates. */
  auto_suspend: boolean;
  /** Sync SCIM groups to teams. */
  group_sync: boolean;
  /** Org role given to provisioned users. */
  default_role: string;
  metadata?: Record<string, string>;
  created_at: string;
  updated_at: string;
}

/**
 * A token record. The token itself is not here: it exists once, in the
 * response that minted it, and the store keeps only a hash.
 */
export interface ScimToken {
  id: string;
  config_id: string;
  name: string;
  last_used_at?: string;
  expires_at?: string;
  created_at: string;
}

/** What comes back when a token is minted or rotated. */
export interface ScimTokenCreated {
  token: ScimToken;
  /** Shown once and never again. */
  plaintext: string;
  /** The SCIM 2.0 base URL to paste into the identity provider. */
  base_url: string;
}

export interface CreateScimConfigInput {
  name: string;
  enabled?: boolean;
  auto_create?: boolean;
  auto_suspend?: boolean;
  group_sync?: boolean;
  default_role?: string;
  metadata?: Record<string, string>;
}

/** Every field optional: anything left out is left alone. */
export type UpdateScimConfigInput = Partial<CreateScimConfigInput>;

export interface ScimApi {
  listConfigs(orgId: string): Promise<{ configs: ScimConfig[] }>;
  createConfig(orgId: string, input: CreateScimConfigInput): Promise<ScimConfig>;
  getConfig(configId: string): Promise<ScimConfig>;
  updateConfig(configId: string, input: UpdateScimConfigInput): Promise<ScimConfig>;
  deleteConfig(configId: string): Promise<{ status: string }>;
  listTokens(configId: string): Promise<{ tokens: ScimToken[] }>;
  createToken(
    configId: string,
    input: { name: string; expires_at?: string },
  ): Promise<ScimTokenCreated>;
  /**
   * Mints a replacement. `grace_seconds` keeps the old token working for a
   * while so an IdP can be updated without a gap in provisioning. Zero
   * revokes the old one at once.
   */
  rotateToken(
    tokenId: string,
    input?: { name?: string; grace_seconds?: number },
  ): Promise<ScimTokenCreated>;
  revokeToken(tokenId: string): Promise<{ status: string }>;
}

const enc = encodeURIComponent;

export const scim = definePlugin<ScimApi>("scim", (ctx) => ({
  listConfigs: (orgId) =>
    ctx.request<{ configs: ScimConfig[] }>(
      "GET",
      `/v1/orgs/${enc(orgId)}/scim/configs`,
    ),
  createConfig: (orgId, input) =>
    ctx.request<ScimConfig>("POST", `/v1/orgs/${enc(orgId)}/scim/configs`, {
      body: input,
    }),
  getConfig: (configId) =>
    ctx.request<ScimConfig>("GET", `/v1/scim/configs/${enc(configId)}`),
  updateConfig: (configId, input) =>
    ctx.request<ScimConfig>("PATCH", `/v1/scim/configs/${enc(configId)}`, {
      body: input,
    }),
  deleteConfig: (configId) =>
    ctx.request<{ status: string }>(
      "DELETE",
      `/v1/scim/configs/${enc(configId)}`,
    ),
  listTokens: (configId) =>
    ctx.request<{ tokens: ScimToken[] }>(
      "GET",
      `/v1/scim/configs/${enc(configId)}/tokens`,
    ),
  createToken: (configId, input) =>
    ctx.request<ScimTokenCreated>(
      "POST",
      `/v1/scim/configs/${enc(configId)}/tokens`,
      { body: input },
    ),
  rotateToken: (tokenId, input) =>
    ctx.request<ScimTokenCreated>(
      "POST",
      `/v1/scim/tokens/${enc(tokenId)}/rotate`,
      { body: input ?? {} },
    ),
  revokeToken: (tokenId) =>
    ctx.request<{ status: string }>(
      "DELETE",
      `/v1/scim/tokens/${enc(tokenId)}`,
    ),
}));

/** True when a token has an expiry that has already passed. */
export function isTokenExpired(token: ScimToken, now = Date.now()): boolean {
  if (!token.expires_at) return false;
  const at = Date.parse(token.expires_at);
  return !Number.isNaN(at) && at <= now;
}
