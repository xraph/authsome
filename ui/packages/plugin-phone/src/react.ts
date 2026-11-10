/**
 * React hooks for phone sign-in.
 *
 * Imported from `@authsome/plugin-phone/react`, so a host that does not use
 * React never loads them.
 */

import { useCallback, useState } from "react";
import { useAuth, usePlugin } from "@authsome/ui-react";
import {
  phone,
  type PhoneApi,
  type PhoneStartInput,
  type PhoneVerifyInput,
  type PhoneVerifyResult,
} from "./index";

export interface UsePhoneAuth {
  /** Sends a code. Resolves to how long it is good for, in seconds. */
  start: (input: PhoneStartInput) => Promise<number>;
  /**
   * Checks the code. On success the session is adopted, so `useAuth()` reports
   * the person as signed in by the time this resolves.
   */
  verify: (input: PhoneVerifyInput) => Promise<PhoneVerifyResult>;
  isSending: boolean;
  isVerifying: boolean;
  /** The last failure from either call, cleared when the next one starts. */
  error: Error | null;
}

/**
 * usePhoneAuth drives the two-step phone flow.
 *
 * ```tsx
 * const { start, verify, isSending, error } = usePhoneAuth();
 * ```
 *
 * Needs `phone()` in the provider's plugins. Without it `usePlugin` throws
 * PluginNotRegisteredError, which names the plugin you forgot.
 */
export function usePhoneAuth(): UsePhoneAuth {
  const api = usePlugin<PhoneApi>(phone);
  const { manager } = useAuth();
  const [isSending, setIsSending] = useState(false);
  const [isVerifying, setIsVerifying] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  const start = useCallback(
    async (input: PhoneStartInput) => {
      setIsSending(true);
      setError(null);
      try {
        const res = await api.start(input);
        return res.expires_in;
      } catch (err) {
        setError(err instanceof Error ? err : new Error(String(err)));
        throw err;
      } finally {
        setIsSending(false);
      }
    },
    [api],
  );

  const verify = useCallback(
    async (input: PhoneVerifyInput) => {
      setIsVerifying(true);
      setError(null);
      try {
        const res = await api.verify(input);
        // Adopting here rather than leaving it to the caller is the whole
        // point: a correct code that does not sign anyone in is a bug every
        // host would otherwise have to fix for themselves.
        await manager.adoptSession({
          session_token: res.session_token,
          refresh_token: res.refresh_token,
          expires_at: res.expires_at,
          // The server returns the user on this response. The cast is here
          // because the generated type marks it optional for the one case
          // where a deployment suppresses it; adoptSession needs it.
          user: res.user as Parameters<typeof manager.adoptSession>[0]["user"],
        });
        return res;
      } catch (err) {
        setError(err instanceof Error ? err : new Error(String(err)));
        throw err;
      } finally {
        setIsVerifying(false);
      }
    },
    [api, manager],
  );

  return { start, verify, isSending, isVerifying, error };
}

export {
  PhoneSignInForm,
  type PhoneSignInFormProps,
} from "./phone-sign-in-form";
