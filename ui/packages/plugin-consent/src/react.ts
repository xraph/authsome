/** React hooks and components for purpose consent. */

import { useCallback, useEffect, useState } from "react";
import { useAuth, usePlugin } from "@authsome/ui-react";
import {
  consent,
  type ConsentApi,
  type ConsentRecord,
  type GrantConsentInput,
  type RevokeConsentInput,
} from "./index";

/**
 * useConsents lists what the signed-in user has agreed to.
 *
 * ```tsx
 * const { consents, isLoading, reload } = useConsents();
 * ```
 */
export function useConsents(purpose?: string): {
  consents: ConsentRecord[];
  isLoading: boolean;
  error: Error | null;
  reload: () => Promise<void>;
} {
  const api = usePlugin<ConsentApi>(consent);
  const { session, isAuthenticated } = useAuth();
  const [consents, setConsents] = useState<ConsentRecord[]>([]);
  const [error, setError] = useState<Error | null>(null);
  const token = session?.session_token ?? null;

  // Which token the list we hold was fetched for. Comparing it against the
  // current one derives the load's progress instead of storing it, which is
  // what keeps setState out of the effect below.
  const [loadedFor, setLoadedFor] = useState<string | null>(null);
  const [isRefreshing, setIsRefreshing] = useState(false);
  const isLoading = (isAuthenticated && loadedFor !== token) || isRefreshing;

  const reload = useCallback(async () => {
    if (!token) return;
    setIsRefreshing(true);
    try {
      setConsents((await api.list({ purpose })).consents);
      setError(null);
      setLoadedFor(token);
    } catch (err) {
      setError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setIsRefreshing(false);
    }
  }, [api, purpose, token]);

  // Inline rather than calling reload(), which sets state before it awaits.
  // The ignore flag stops a response for a session the user already left
  // from overwriting a newer list.
  useEffect(() => {
    if (!isAuthenticated || !token) return;
    let ignore = false;
    void (async () => {
      try {
        const res = await api.list({ purpose });
        if (!ignore) {
          setConsents(res.consents);
          setError(null);
        }
      } catch (err) {
        if (!ignore) setError(err instanceof Error ? err : new Error(String(err)));
      } finally {
        if (!ignore) setLoadedFor(token);
      }
    })();
    return () => {
      ignore = true;
    };
  }, [api, isAuthenticated, purpose, token]);

  return { consents, isLoading, error, reload };
}

/** useGrantConsent records agreement to a purpose. */
export function useGrantConsent(): {
  grant: (input: GrantConsentInput) => Promise<ConsentRecord>;
  isSaving: boolean;
  error: Error | null;
} {
  const api = usePlugin<ConsentApi>(consent);
  const [isSaving, setIsSaving] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  const grant = useCallback(
    async (input: GrantConsentInput) => {
      setIsSaving(true);
      setError(null);
      try {
        return await api.grant(input);
      } catch (err) {
        setError(err instanceof Error ? err : new Error(String(err)));
        throw err;
      } finally {
        setIsSaving(false);
      }
    },
    [api],
  );

  return { grant, isSaving, error };
}

/** useRevokeConsent withdraws agreement to a purpose. */
export function useRevokeConsent(): {
  revoke: (input: RevokeConsentInput) => Promise<void>;
  isSaving: boolean;
  error: Error | null;
} {
  const api = usePlugin<ConsentApi>(consent);
  const [isSaving, setIsSaving] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  const revoke = useCallback(
    async (input: RevokeConsentInput) => {
      setIsSaving(true);
      setError(null);
      try {
        await api.revoke(input);
      } catch (err) {
        setError(err instanceof Error ? err : new Error(String(err)));
        throw err;
      } finally {
        setIsSaving(false);
      }
    },
    [api],
  );

  return { revoke, isSaving, error };
}

export { ConsentList, type ConsentListProps } from "./consent-list";
export { ConsentGate, type ConsentGateProps } from "./consent-gate";
