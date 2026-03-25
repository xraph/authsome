import type { ClientConfig } from "@authsome/ui-core";

/**
 * All auth methods enabled: password, social (Google/GitHub/Apple),
 * passkey, MFA (totp/sms), magic link, SSO, and branding.
 */
export const CONFIG_ALL_ENABLED: ClientConfig = {
  version: "0.5.0",
  app_id: "aapp_storybook",
  branding: {
    app_name: "Storybook App",
  },
  password: { enabled: true },
  social: {
    enabled: true,
    providers: [
      { id: "google", name: "Google" },
      { id: "github", name: "GitHub" },
      { id: "apple", name: "Apple" },
    ],
  },
  passkey: { enabled: true },
  mfa: { enabled: true, methods: ["totp", "sms"] },
  magiclink: { enabled: true },
  sso: {
    enabled: true,
    connections: [
      { id: "okta", name: "Okta" },
      { id: "azure", name: "Azure AD" },
    ],
  },
  supported_plugins: [
    "password",
    "social",
    "passkey",
    "mfa",
    "magiclink",
    "sso",
    "email",
    "device",
  ],
};

/** Only social providers enabled (Google, GitHub). */
export const CONFIG_SOCIAL_ONLY: ClientConfig = {
  version: "0.5.0",
  app_id: "aapp_storybook",
  password: { enabled: true },
  social: {
    enabled: true,
    providers: [
      { id: "google", name: "Google" },
      { id: "github", name: "GitHub" },
    ],
  },
  passkey: { enabled: false },
  mfa: { enabled: false, methods: [] },
  magiclink: { enabled: false },
  sso: { enabled: false, connections: [] },
  supported_plugins: ["password", "social"],
};

/** Only passkey enabled. */
export const CONFIG_PASSKEY_ONLY: ClientConfig = {
  version: "0.5.0",
  app_id: "aapp_storybook",
  password: { enabled: true },
  social: { enabled: false, providers: [] },
  passkey: { enabled: true },
  mfa: { enabled: false, methods: [] },
  magiclink: { enabled: false },
  sso: { enabled: false, connections: [] },
  supported_plugins: ["password", "passkey"],
};

/** Only password enabled (most minimal setup). */
export const CONFIG_PASSWORD_ONLY: ClientConfig = {
  version: "0.5.0",
  app_id: "aapp_storybook",
  password: { enabled: true },
  social: { enabled: false, providers: [] },
  passkey: { enabled: false },
  mfa: { enabled: false, methods: [] },
  magiclink: { enabled: false },
  sso: { enabled: false, connections: [] },
  supported_plugins: ["password"],
};

/** Social + passkey (typical modern setup). */
export const CONFIG_SOCIAL_AND_PASSKEY: ClientConfig = {
  version: "0.5.0",
  app_id: "aapp_storybook",
  password: { enabled: true },
  social: {
    enabled: true,
    providers: [
      { id: "google", name: "Google" },
      { id: "github", name: "GitHub" },
    ],
  },
  passkey: { enabled: true },
  mfa: { enabled: false, methods: [] },
  magiclink: { enabled: false },
  sso: { enabled: false, connections: [] },
  supported_plugins: ["password", "social", "passkey"],
};

/** Password + MFA (totp, sms). */
export const CONFIG_MFA_ENABLED: ClientConfig = {
  version: "0.5.0",
  app_id: "aapp_storybook",
  password: { enabled: true },
  social: { enabled: false, providers: [] },
  passkey: { enabled: false },
  mfa: { enabled: true, methods: ["totp", "sms"] },
  magiclink: { enabled: false },
  sso: { enabled: false, connections: [] },
  supported_plugins: ["password", "mfa"],
};

/** Magic link enabled. */
export const CONFIG_MAGIC_LINK: ClientConfig = {
  version: "0.5.0",
  app_id: "aapp_storybook",
  password: { enabled: true },
  social: { enabled: false, providers: [] },
  passkey: { enabled: false },
  mfa: { enabled: false, methods: [] },
  magiclink: { enabled: true },
  sso: { enabled: false, connections: [] },
  supported_plugins: ["password", "magiclink"],
};

/** SSO with Okta + Azure AD connections. */
export const CONFIG_SSO: ClientConfig = {
  version: "0.5.0",
  app_id: "aapp_storybook",
  password: { enabled: true },
  social: { enabled: false, providers: [] },
  passkey: { enabled: false },
  mfa: { enabled: false, methods: [] },
  magiclink: { enabled: false },
  sso: {
    enabled: true,
    connections: [
      { id: "okta", name: "Okta" },
      { id: "azure-ad", name: "Azure AD" },
    ],
  },
  supported_plugins: ["password", "sso"],
};

/** MFA with TOTP only (no SMS). */
export const CONFIG_MFA_TOTP_ONLY: ClientConfig = {
  version: "0.5.0",
  app_id: "aapp_storybook",
  password: { enabled: true },
  social: { enabled: false, providers: [] },
  passkey: { enabled: false },
  mfa: { enabled: true, methods: ["totp"] },
  magiclink: { enabled: false },
  sso: { enabled: false, connections: [] },
  supported_plugins: ["password", "mfa"],
};

/** MFA with SMS only (no TOTP). */
export const CONFIG_MFA_SMS_ONLY: ClientConfig = {
  version: "0.5.0",
  app_id: "aapp_storybook",
  password: { enabled: true },
  social: { enabled: false, providers: [] },
  passkey: { enabled: false },
  mfa: { enabled: true, methods: ["sms"] },
  magiclink: { enabled: false },
  sso: { enabled: false, connections: [] },
  supported_plugins: ["password", "mfa"],
};

/** Empty config - nothing explicitly enabled. */
export const CONFIG_EMPTY: ClientConfig = {};

/** All methods + custom branding. */
export const CONFIG_WITH_BRANDING: ClientConfig = {
  ...CONFIG_ALL_ENABLED,
  branding: {
    app_name: "Acme Corp",
    logo_url: "https://placehold.co/120x40?text=ACME",
  },
};

/** Waitlist mode enabled. */
export const CONFIG_WAITLIST_ENABLED: ClientConfig = {
  ...CONFIG_ALL_ENABLED,
  waitlist: { enabled: true },
};
