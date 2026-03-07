/**
 * AuthClient adapter — thin wrapper over the auto-generated API client.
 *
 * The generated client lives in ./generated/api-client.ts and covers all
 * core + plugin endpoints.  This file extends it with a few convenience
 * overrides so that auth.ts (and downstream consumers) keep working
 * without changes.
 */

import {
  AuthClient as GeneratedClient,
  AuthClientError,
  type AuthClientConfig,
} from "./generated/api-client";
import type { ClientConfig } from "./types";

// ── Re-exports ───────────────────────────────────────

export { AuthClientError };
export type { AuthClientConfig };

// Re-export generated request types used by auth.ts
export type { SignInRequest, SignUpRequest } from "./generated/api-types";

// Re-export all generated API model types
export type {
  AuthResponse,
  Device,
  EnrollResponse,
  Invitation,
  KeyListItem,
  Member,
  Organization,
  TokenResponse,
  User,
} from "./generated/api-types";

// Backward-compatible type aliases for types renamed in the dynamic spec.
export type { KeyListItem as APIKey } from "./generated/api-types";
export type { EnrollResponse as MFAEnrollment } from "./generated/api-types";

// Re-export all generated request types
export type {
  AdminBanUserRequest,
  ChangePasswordRequest,
  ForgotPasswordRequest,
  CreateAPIKeyRequest,
  SendMagicLinkRequest,
  VerifyMagicLinkRequest,
  UpdateMeRequest,
  ChallengeMFARequest,
  EnrollMFARequest,
  SendSMSCodeRequest,
  VerifySMSCodeRequest,
  VerifyMFARequest,
  CreateOrganizationRequest,
  UpdateOrganizationRequest,
  CreateInvitationRequest,
  AddMemberRequest,
  RefreshRequest,
  ResetPasswordRequest,
  SsoACSRequest,
  SsoCallbackRequest,
  VerifyEmailRequest,
  VerifyRecoveryCodeRequest,
} from "./generated/api-types";

// Backward-compatible request type aliases
export type { SendMagicLinkRequest as MagicLinkSendRequest } from "./generated/api-types";
export type { VerifyMagicLinkRequest as MagicLinkVerifyRequest } from "./generated/api-types";
export type { ChallengeMFARequest as MfaChallengeRequest } from "./generated/api-types";
export type { EnrollMFARequest as MfaEnrollRequest } from "./generated/api-types";
export type { SendSMSCodeRequest as MfaSMSSendRequest } from "./generated/api-types";
export type { VerifySMSCodeRequest as MfaSMSVerifyRequest } from "./generated/api-types";
export type { VerifyMFARequest as MfaVerifyRequest } from "./generated/api-types";
export type { CreateOrganizationRequest as CreateOrgRequest } from "./generated/api-types";
export type { UpdateOrganizationRequest as UpdateOrgRequest } from "./generated/api-types";

// ── Manual types (not in the OpenAPI spec) ───────────

/** Options for paginated list endpoints. */
export interface ListOptions {
  limit?: number;
  offset?: number;
}

/** Paginated list response. */
export interface ListResponse<T> {
  items: T[];
  total: number;
}

// ── AuthClient adapter ──────────────────────────────

/**
 * AuthClient extends the auto-generated client with backward-compatible
 * convenience methods that auth.ts depends on.
 *
 * All 80+ generated endpoints are inherited as-is.  Only two methods
 * are overridden to preserve the simpler call-site signatures used by
 * the auth state machine.
 */
export class AuthClient extends GeneratedClient {
  /**
   * Refresh session tokens.
   *
   * Accepts a raw refresh-token string (used by auth.ts) or the full
   * RefreshRequest body.
   */
  override async refresh(body: { refresh_token: string } | string): Promise<any> {
    const req = typeof body === "string" ? { refresh_token: body } : body;
    return super.refresh(req);
  }

  /**
   * Sign out.
   *
   * Accepts a raw token string (used by auth.ts) or the full
   * (body, token) pair from the generated client.
   */
  override async signOut(bodyOrToken: unknown, token?: string): Promise<any> {
    if (typeof bodyOrToken === "string") {
      return super.signOut({} as any, bodyOrToken);
    }
    return super.signOut(bodyOrToken as any, token!);
  }

  /**
   * MFA challenge — bridge for auth.ts.
   *
   * auth.ts calls `mfaChallenge({ enrollment_id, code })` and expects
   * an AuthResponse back.  The generated client uses `challengeMFA`.
   */
  async mfaChallenge(body: { enrollment_id?: string; code: string }): Promise<any> {
    return super.challengeMFA(body as any);
  }

  /**
   * Verify an MFA recovery code.
   *
   * Accepts a raw code string (used by auth.ts) or the full request body.
   */
  override async verifyRecoveryCode(body: { code: string } | string): Promise<any> {
    const req = typeof body === "string" ? { code: body } : body;
    return super.verifyRecoveryCode(req);
  }

  /**
   * Send an SMS code for MFA — bridge for auth.ts.
   *
   * Simplifies the call-site by accepting just the session token.
   */
  async sendSMSCodeForMFA(token: string): Promise<any> {
    return super.sendSMSCode({}, token);
  }

  /**
   * Verify an SMS code for MFA — bridge for auth.ts.
   *
   * Simplifies the call-site by accepting code + token directly.
   */
  async verifySMSCodeForMFA(code: string, token: string): Promise<any> {
    return super.verifySMSCode({ code }, token);
  }

  /**
   * Fetch the client configuration from the backend.
   *
   * The config describes which auth methods are enabled so SDK
   * components can auto-configure without manual props.
   */
  async fetchClientConfig(publishableKey?: string): Promise<ClientConfig> {
    const url = new URL("/v1/auth/client-config", (this as any).config.baseURL);
    if (publishableKey) {
      url.searchParams.set("key", publishableKey);
    }
    const fetchFn = (this as any).config.fetch ?? globalThis.fetch;
    const res = await fetchFn(url.toString(), {
      method: "GET",
      headers: { "Content-Type": "application/json" },
    });
    if (!res.ok) {
      throw new AuthClientError("Failed to fetch client config", res.status);
    }
    return res.json();
  }
}
