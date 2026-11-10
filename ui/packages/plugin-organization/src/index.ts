/**
 * Organization invitations, the client half of authsome's `organization`
 * server plugin.
 *
 * ```ts
 * import { organization } from "@authsome/plugin-organization";
 *
 * const auth = new AuthManager({ baseURL, plugins: [organization()] });
 * const invite = await auth.plugin(organization).getInvitation(token);
 * ```
 *
 * `getInvitation` is unauthenticated on purpose: someone deciding whether to
 * accept has not signed in yet, and often has no account at all.
 */

import { definePlugin } from "@authsome/ui-core";

/** An invitation as the org's admins see it. */
export interface Invitation {
  id: string;
  org_id: string;
  email: string;
  role: string;
  inviter_id: string;
  /** "pending", "accepted", "declined", "expired". */
  status: string;
  expires_at: string;
  created_at: string;
}

/**
 * What the person holding the token is shown before they accept.
 *
 * Deliberately not the stored row. It carries the org's name and the
 * inviter's, which the row does not have, and leaves out the member list and
 * anything else the recipient has no business seeing yet.
 */
export interface InvitationPreview {
  org_id: string;
  org_name: string;
  org_slug?: string;
  /** The inviter's display name, or their email when they have no name. */
  inviter_name: string;
  role: string;
  /** Partially masked, so the page can say who it is for without leaking it. */
  email_masked: string;
  expires_at: string;
}

export interface InvitationListResult {
  invitations: Invitation[];
  next_cursor?: string;
}

export interface Member {
  id: string;
  org_id: string;
  user_id: string;
  role: string;
  created_at: string;
  updated_at: string;
}

export interface OrganizationApi {
  /** Reads an invitation by its token. No session needed. */
  getInvitation(token: string): Promise<InvitationPreview>;
  /** Accepts it. Needs a session, so sign in or sign up first. */
  acceptInvitation(token: string): Promise<Member>;
  declineInvitation(token: string): Promise<{ status: string }>;
  listInvitations(
    orgId: string,
    opts?: { limit?: number; cursor?: string },
  ): Promise<InvitationListResult>;
  createInvitation(
    orgId: string,
    input: { email: string; role?: string },
  ): Promise<Invitation>;
}

export const organization = definePlugin<OrganizationApi>(
  "organization",
  (ctx) => ({
    getInvitation: (token) =>
      ctx.request<InvitationPreview>(
        "GET",
        `/v1/orgs/invitations/${encodeURIComponent(token)}`,
      ),
    acceptInvitation: (token) =>
      ctx.request<Member>("POST", "/v1/orgs/invitations/accept", {
        body: { token },
      }),
    declineInvitation: (token) =>
      ctx.request<{ status: string }>("POST", "/v1/orgs/invitations/decline", {
        body: { token },
      }),
    listInvitations: (orgId, opts) =>
      ctx.request<InvitationListResult>(
        "GET",
        `/v1/orgs/${encodeURIComponent(orgId)}/invitations`,
        { query: { limit: opts?.limit, cursor: opts?.cursor } },
      ),
    createInvitation: (orgId, input) =>
      ctx.request<Invitation>(
        "POST",
        `/v1/orgs/${encodeURIComponent(orgId)}/invitations`,
        { body: input },
      ),
  }),
);

/** "in 6 days", "today", "expired". */
export function expiresIn(iso: string): string {
  const at = new Date(iso).getTime();
  if (Number.isNaN(at)) return "";
  const days = Math.ceil((at - Date.now()) / 86_400_000);
  if (days < 0) return "expired";
  if (days === 0) return "today";
  if (days === 1) return "in 1 day";
  return `in ${days} days`;
}
