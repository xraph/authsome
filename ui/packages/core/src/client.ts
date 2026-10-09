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
import type {
  ApiStatusResponse,
  ApiTokenResponse,
  ChallengeMFARequest,
  ChallengeResponse,
  DeviceCompleteResponse,
  RecoveryVerifyResponse,
  SMSSendResponse,
  SMSVerifyResponse,
  VerifyRecoveryCodeRequest,
  RefreshRequest,
} from "./generated/api-types";
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
  User,
} from "./generated/api-types";

// Backward-compatible type aliases for types renamed in the dynamic spec.
export type { KeyListItem as APIKey } from "./generated/api-types";
export type { EnrollResponse as MFAEnrollment } from "./generated/api-types";
export type { ApiTokenResponse as TokenResponse } from "./generated/api-types";

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
  PasskeyLoginFinishRequest,
  PasskeyRegisterFinishRequest,
  RefreshRequest,
  ResetPasswordRequest,
  VerifyEmailRequest,
  VerifyRecoveryCodeRequest,
} from "./generated/api-types";

// SsoACSRequest and SsoCallbackRequest used to be re-exported here. Both
// endpoints now carry their input in the path and query string rather than a
// body, so the spec no longer describes a request schema for either and there
// is nothing left to alias. Call ssoACS(provider) and ssoCallback(provider,
// state, code, error) directly.

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

/** The signup decision returned by the app's waitlist. */
export type WaitlistStatus = "pending" | "approved" | "rejected";

export interface WaitlistJoinResponse {
  email: string;
  status: WaitlistStatus;
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
  private readonly waitlistURL: string;
  private readonly waitlistFetch: typeof globalThis.fetch;

  constructor(config: AuthClientConfig) {
    const fetchFn = config.fetch ?? globalThis.fetch.bind(globalThis);
    let baseURL = config.baseURL;
    while (baseURL.endsWith("/")) baseURL = baseURL.slice(0, -1);
    const deviceCompletionURL = `${baseURL}/v1/oauth/device/complete`;
    super({
      ...config,
      fetch: (input, init) => {
        // Device approval supports cookie sessions as well as bearer tokens.
        // Keep that transport option while using the generated form contract.
        const isDeviceCompletion =
          input === deviceCompletionURL && init?.method === "POST";
        return fetchFn(
          input,
          isDeviceCompletion ? { ...init, credentials: "include" } : init,
        );
      },
    });
    this.waitlistURL = `${baseURL}/v1/waitlist/join`;
    this.waitlistFetch = fetchFn;
  }

  /** Join the waitlist, or return the existing entry and its approval status. */
  async joinWaitlist(body: {
    email: string;
    name?: string;
  }): Promise<WaitlistJoinResponse> {
    const headers: Record<string, string> = {
      "Content-Type": "application/json",
      Accept: "application/json",
    };
    const key = this.getPublishableKey();
    if (key) headers["X-Publishable-Key"] = key;

    const response = await this.waitlistFetch(this.waitlistURL, {
      method: "POST",
      headers,
      body: JSON.stringify({ ...body, email: body.email.trim().toLowerCase() }),
    });
    const data = await response.json().catch(() => null);
    if (!response.ok) {
      throw new AuthClientError(
        data?.error ?? `Request failed with status ${response.status}`,
        response.status,
        data?.type,
        data ?? undefined,
      );
    }
    if (
      typeof data?.email !== "string" ||
      !["pending", "approved", "rejected"].includes(data?.status)
    ) {
      throw new AuthClientError(
        "Unable to check waitlist status. Please try again.",
      );
    }
    return data as WaitlistJoinResponse;
  }

  /**
   * Refresh session tokens.
   *
   * Accepts a raw refresh-token string (used by auth.ts) or the full
   * RefreshRequest body.
   */
  override async refresh(
    body: RefreshRequest | string,
  ): Promise<ApiTokenResponse> {
    const req = typeof body === "string" ? { refresh_token: body } : body;
    return super.refresh(req);
  }

  /**
   * Sign out.
   *
   * Accepts a raw token string (used by auth.ts). The generated client used
   * to take a (body, token) pair; the endpoint no longer reads a body, so the
   * two-argument form is still accepted and its first argument ignored rather
   * than breaking callers outside this package.
   */
  override async signOut(
    bodyOrToken: unknown,
    token?: string,
  ): Promise<ApiStatusResponse> {
    if (typeof bodyOrToken === "string") {
      return super.signOut(bodyOrToken);
    }
    return super.signOut(token!);
  }

  /** @deprecated Use challengeMFA with the ticket from sign-in. */
  async mfaChallenge(
    body: ChallengeMFARequest | { enrollment_id?: string; code: string },
  ): Promise<ChallengeResponse> {
    if (!("mfa_ticket" in body) || !body.mfa_ticket) {
      throw new Error(
        "MFA challenge requires an authentication ticket, not an enrollment ID",
      );
    }
    return super.challengeMFA(body);
  }

  /** Verify a recovery code in an existing cookie-authenticated session. */
  override async verifyRecoveryCode(
    body: VerifyRecoveryCodeRequest | string,
  ): Promise<RecoveryVerifyResponse> {
    return super.verifyRecoveryCode(
      typeof body === "string" ? { code: body } : body,
    );
  }

  /** Send an SMS code for the authenticated session. */
  async sendSMSCodeForMFA(token: string): Promise<SMSSendResponse> {
    return super.sendSMSCode({}, token);
  }

  /** Verify an SMS code without issuing a session. */
  async verifySMSCodeForMFA(
    code: string,
    token: string,
  ): Promise<SMSVerifyResponse> {
    return super.verifySMSCode({ code }, token);
  }

  /** Approve or deny a device using a bearer token or the existing session cookie. */
  async completeDeviceAuthorization(
    userCode: string,
    action: "approve" | "deny",
    token?: string,
  ): Promise<DeviceCompleteResponse> {
    return super.oauth2DeviceComplete(
      { user_code: userCode, action },
      token ?? "",
    );
  }

  /**
   * Fetch the client configuration from the backend.
   *
   * The config describes which auth methods are enabled so SDK
   * components can auto-configure without manual props.
   */
  async fetchClientConfig(publishableKey?: string): Promise<ClientConfig> {
    return super.getClientConfig("", publishableKey);
  }
}
