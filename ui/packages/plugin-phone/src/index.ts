/**
 * Phone sign-in, the client half of authsome's `phone` server plugin.
 *
 * ```ts
 * import { phone } from "@authsome/plugin-phone";
 *
 * const auth = new AuthManager({ baseURL, plugins: [phone()] });
 * const sms = auth.plugin(phone);
 *
 * await sms.start({ phone: "+351912345678" });
 * const res = await sms.verify({ phone: "+351912345678", code: "482917" });
 * await auth.adoptSession(res);
 * ```
 *
 * Both endpoints are unauthenticated, because this is how someone signs in
 * for the first time. `verify` returns a real session, which the manager
 * adopts: the plugin does not sign anyone in by itself, so a host can decide
 * what happens between the code being right and the session being live.
 */

import { definePlugin } from "@authsome/ui-core";

/** What `start` sends. `app_id` is only needed for a multi-app deployment. */
export interface PhoneStartInput {
  phone: string;
  app_id?: string;
}

/** What the server says when a code is on its way. */
export interface PhoneStartResult {
  /** Server-side status string, e.g. "sent". */
  status: string;
  /** Seconds until the code stops working. */
  expires_in: number;
}

export interface PhoneVerifyInput {
  phone: string;
  code: string;
  app_id?: string;
}

/**
 * A real session. Hand it to `AuthManager.adoptSession` to sign the person
 * in. `new_user` is true when this number had no account until now, which is
 * the hook a host needs for an onboarding step.
 */
export interface PhoneVerifyResult {
  session_token: string;
  refresh_token: string;
  expires_at: string;
  new_user: boolean;
  user?: unknown;
}

export interface PhoneApi {
  /** Sends a code to the number. */
  start(input: PhoneStartInput): Promise<PhoneStartResult>;
  /** Exchanges a code for a session. */
  verify(input: PhoneVerifyInput): Promise<PhoneVerifyResult>;
}

/**
 * The phone plugin. Register it on the manager, then look it up with
 * `auth.plugin(phone)`.
 */
export const phone = definePlugin<PhoneApi>("phone", (ctx) => ({
  start: (input) =>
    ctx.request<PhoneStartResult>("POST", "/v1/phone/start", { body: input }),
  verify: (input) =>
    ctx.request<PhoneVerifyResult>("POST", "/v1/phone/verify", { body: input }),
}));
