/** Core type definitions for AuthSome UI Kit. */

import type { User } from "./generated/api-types";

// ── Re-export API types from generated code ──────────
//
// These were previously hand-maintained. They are now auto-generated
// by `make generate-ui-core` in sdkgen/.

export type {
  User,
  AuthResponse,
  Organization,
  Member,
  EnrollResponse,
} from "./generated/api-types";

// Backward-compatible alias for the renamed enrollment type.
export type { EnrollResponse as MFAEnrollment } from "./generated/api-types";

// ── Session ──────────────────────────────────────────
//
// Session tokens returned after authentication / refresh.
// The generated TokenResponse has all-optional fields, but we keep
// this interface with required fields because the auth state machine
// relies on them being present.

export interface Session {
  session_token: string;
  refresh_token: string;
  expires_at: string;
}

// ── Manual types (not in the OpenAPI spec) ────────────

/** API error response. */
export interface APIError {
  error: string;
  code?: number;
  /**
   * Machine-readable error type set by the backend (e.g. "email_not_verified").
   * Currently not part of the OpenAPI spec, so it's parsed manually from the
   * error response body in the generated client's request helper.
   */
  type?: string;
}

/** Authentication state. */
export type AuthState =
  | { status: "idle" }
  | { status: "loading" }
  | { status: "authenticated"; user: User; session: Session }
  | { status: "unauthenticated" }
  /**
   * Sign-in returned the MFA gate (HTTP 403 with type:"mfa_required").
   * The user must complete the second factor against the same ticket
   * via AuthManager.submitMFAChallenge(code) — the ticket is single-
   * use after a correct code and expires after 5 minutes server-side.
   *
   * `email` is preserved so the form can show "complete sign-in for
   * <email>" and so AuthManager.signIn(...) can be retried fully.
   *
   * `availableMethods` is what the backend reports as enrolled and
   * ready to validate — typically ["totp"] today.
   */
  | { status: "mfa_required"; email: string; mfaTicket: string; availableMethods: string[] }
  /**
   * Returned when a sign-in attempt is rejected because the user's email
   * has not yet been verified, or when sign-up succeeded with email
   * verification required. The `email` is preserved so a "resend" CTA can
   * target the same account.
   */
  | { status: "email_not_verified"; email: string }
  /**
   * Sign-up succeeded but the account requires email verification before
   * sign-in works. The session token returned by the API in this state is
   * non-functional and is intentionally discarded.
   */
  | { status: "verification_pending"; email: string }
  | { status: "error"; error: string };

/** Configuration for the AuthSome client. */
export interface AuthConfig {
  /** Base URL of the AuthSome API (e.g., "https://api.example.com"). */
  baseURL: string;

  /** Publishable key for auto-discovering enabled auth methods. */
  publishableKey?: string;

  /** Pre-fetched client config (useful for SSR). */
  initialClientConfig?: ClientConfig;

  /** Custom fetch implementation (defaults to global fetch). */
  fetch?: typeof fetch;

  /** Storage implementation for persisting tokens (defaults to localStorage). */
  storage?: TokenStorage;

  /** Callback invoked when the auth state changes. */
  onStateChange?: (state: AuthState) => void;

  /** Callback invoked on authentication error. */
  onError?: (error: APIError) => void;
}

// ── Client Config (auto-discovery) ─────────────────

/** Social provider info from the backend. */
export interface SocialProviderConfig {
  id: string;
  name: string;
}

/** SSO connection info from the backend. */
export interface SSOConnectionConfig {
  id: string;
  name: string;
}

/** A single option for select/radio/checkbox signup fields. */
export interface SignupFieldOption {
  label: string;
  value: string;
}

/** Validation rules for a signup field. */
export interface SignupFieldValidation {
  required?: boolean;
  min_len?: number;
  max_len?: number;
  pattern?: string;
  min?: number;
  max?: number;
}

/** A custom signup form field from the backend. */
export interface SignupFieldConfig {
  key: string;
  label: string;
  type: string;
  placeholder?: string;
  description?: string;
  options?: SignupFieldOption[];
  default?: string;
  validation?: SignupFieldValidation;
  order: number;
}

/**
 * Client configuration returned by the backend.
 *
 * Describes which auth methods are enabled so SDK components
 * can auto-configure their UI without manual props.
 */
export interface ClientConfig {
  version?: string;
  app_id?: string;
  branding?: {
    app_name?: string;
    logo_url?: string;
  };
  password?: { enabled: boolean };
  social?: { enabled: boolean; providers: SocialProviderConfig[] };
  passkey?: { enabled: boolean };
  mfa?: { enabled: boolean; methods: string[] };
  magiclink?: { enabled: boolean };
  sso?: { enabled: boolean; connections: SSOConnectionConfig[] };
  /** List of server-side plugin names that are installed and active. */
  supported_plugins?: string[];
  /** Custom signup form fields configured for this app. */
  signup_fields?: SignupFieldConfig[];
  /** Waitlist configuration. */
  waitlist?: { enabled: boolean };
  /** Email verification configuration. */
  email_verification?: { enabled: boolean; required: boolean };
  /** Device authorization (OAuth 2.0 device code flow). */
  device_authorization?: { enabled: boolean };
  /** Captcha challenge (e.g. Cloudflare Turnstile). */
  captcha?: {
    required: boolean;
    provider?: string;
    site_key?: string;
  };
}

/** Interface for persisting tokens across sessions. */
export interface TokenStorage {
  getItem(key: string): string | null | Promise<string | null>;
  setItem(key: string, value: string): void | Promise<void>;
  removeItem(key: string): void | Promise<void>;
}
