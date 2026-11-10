/** React hooks and components for OAuth consent and connected apps. */

import { useCallback, useEffect, useState } from "react";
import { useAuth, usePlugin } from "@authsome/ui-react";
import {
  oauth2,
  type ConsentDecisionResult,
  type ConsentRequest,
  type OAuth2Api,
  type OAuthGrant,
} from "./index";

/**
 * useConsentRequest loads what an authorize screen has to show.
 *
 * The consent id comes from the `cid` query parameter the authorize endpoint
 * redirected with.
 */
export function useConsentRequest(cid: string | null): {
  request: ConsentRequest | null;
  isLoading: boolean;
  error: Error | null;
} {
  const api = usePlugin<OAuth2Api>(oauth2);
  const [request, setRequest] = useState<ConsentRequest | null>(null);
  const [error, setError] = useState<Error | null>(null);
  const [loadedFor, setLoadedFor] = useState<string | null>(null);

  const isLoading = Boolean(cid) && loadedFor !== cid;

  useEffect(() => {
    if (!cid) return;
    let ignore = false;
    void (async () => {
      try {
        const res = await api.getConsentRequest(cid);
        if (!ignore) {
          setRequest(res);
          setError(null);
        }
      } catch (err) {
        if (!ignore) setError(err instanceof Error ? err : new Error(String(err)));
      } finally {
        if (!ignore) setLoadedFor(cid);
      }
    })();
    return () => {
      ignore = true;
    };
  }, [api, cid]);

  return { request, isLoading, error };
}

/** useConsentDecision records allow or deny. */
export function useConsentDecision(): {
  decide: (input: {
    cid: string;
    csrf: string;
    decision: "allow" | "deny";
  }) => Promise<ConsentDecisionResult>;
  isDeciding: boolean;
  error: Error | null;
} {
  const api = usePlugin<OAuth2Api>(oauth2);
  const [isDeciding, setIsDeciding] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  const decide = useCallback(
    async (input: { cid: string; csrf: string; decision: "allow" | "deny" }) => {
      setIsDeciding(true);
      setError(null);
      try {
        return await api.decide(input);
      } catch (err) {
        setError(err instanceof Error ? err : new Error(String(err)));
        throw err;
      } finally {
        setIsDeciding(false);
      }
    },
    [api],
  );

  return { decide, isDeciding, error };
}

/** useMyGrants lists the apps the signed-in user has authorised. */
export function useMyGrants(): {
  grants: OAuthGrant[];
  isLoading: boolean;
  error: Error | null;
  reload: () => Promise<void>;
} {
  const api = usePlugin<OAuth2Api>(oauth2);
  const { session, isAuthenticated } = useAuth();
  const [grants, setGrants] = useState<OAuthGrant[]>([]);
  const [error, setError] = useState<Error | null>(null);
  const token = session?.session_token ?? null;
  const [loadedFor, setLoadedFor] = useState<string | null>(null);
  const [isRefreshing, setIsRefreshing] = useState(false);
  const isLoading = (isAuthenticated && loadedFor !== token) || isRefreshing;

  const reload = useCallback(async () => {
    if (!token) return;
    setIsRefreshing(true);
    try {
      setGrants((await api.listGrants()).grants);
      setError(null);
      setLoadedFor(token);
    } catch (err) {
      setError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setIsRefreshing(false);
    }
  }, [api, token]);

  useEffect(() => {
    if (!isAuthenticated || !token) return;
    let ignore = false;
    void (async () => {
      try {
        const res = await api.listGrants();
        if (!ignore) {
          setGrants(res.grants);
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
  }, [api, isAuthenticated, token]);

  return { grants, isLoading, error, reload };
}

/** useRevokeGrant withdraws an app's access. */
export function useRevokeGrant(): {
  revoke: (clientId: string) => Promise<void>;
  isSaving: boolean;
  error: Error | null;
} {
  const api = usePlugin<OAuth2Api>(oauth2);
  const [isSaving, setIsSaving] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  const revoke = useCallback(
    async (clientId: string) => {
      setIsSaving(true);
      setError(null);
      try {
        await api.revokeGrant(clientId);
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

export {
  OAuthConsentScreen,
  type OAuthConsentScreenProps,
} from "./oauth-consent-screen";
export {
  ConnectedAppsList,
  type ConnectedAppsListProps,
} from "./connected-apps-list";
