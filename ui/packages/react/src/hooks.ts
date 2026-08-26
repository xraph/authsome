/** Convenience hooks built on top of useAuth. */

import { useCallback, useEffect, useState } from "react";
import type { ClientConfig, Organization, User, ListResponse } from "@authsome/ui-core";
import { useAuth } from "./context";

/**
 * useUser returns the current user and a function to reload the profile.
 *
 * ```tsx
 * const { user, reload } = useUser();
 * ```
 */
export function useUser(): {
  user: User | null;
  isLoading: boolean;
  reload: () => Promise<void>;
} {
  const { user, isLoading, client, session } = useAuth();

  // reload() can put a freshly fetched profile in front of whatever the
  // context is holding. That override lasts until the context user itself
  // changes, at which point it is stale and gets dropped.
  //
  // The comparison happens during render rather than in an effect. Syncing
  // state to a prop through an effect renders once with the old value, then
  // again with the new one, and react-hooks/set-state-in-effect exists to
  // point that out. Adjusting during render is what React documents instead:
  // the extra pass happens before anything is committed to the screen.
  const [override, setOverride] = useState<User | null>(null);
  const [lastSeen, setLastSeen] = useState<User | null>(user);
  if (user !== lastSeen) {
    setLastSeen(user);
    setOverride(null);
  }

  const reload = useCallback(async () => {
    if (!session) return;
    setOverride(await client.getMe(session.session_token));
  }, [client, session]);

  return { user: override ?? user, isLoading, reload };
}

/**
 * useOrganizations fetches the list of organizations for the current user.
 *
 * ```tsx
 * const { organizations, isLoading } = useOrganizations();
 * ```
 */
export function useOrganizations(): {
  organizations: Organization[];
  total: number;
  isLoading: boolean;
  reload: () => Promise<void>;
} {
  const { client, session, isAuthenticated } = useAuth();
  const [data, setData] = useState<ListResponse<Organization>>({ items: [], total: 0 });
  const token = session?.session_token ?? null;

  // Which token the list we are holding was fetched for. Comparing it against
  // the current one derives the automatic load's progress instead of storing
  // it, which is what keeps setState out of the effect below. A new session
  // makes them differ again, so switching accounts reports loading too.
  const [loadedFor, setLoadedFor] = useState<string | null>(null);
  // A manual reload is not covered by that comparison, because the list it is
  // replacing was already fetched for this token. It gets its own flag, set
  // from a callback rather than an effect.
  const [isRefreshing, setIsRefreshing] = useState(false);

  const isLoading = (isAuthenticated && loadedFor !== token) || isRefreshing;

  const load = useCallback(async () => {
    if (!session) return;
    setIsRefreshing(true);
    try {
      const res = (await client.listOrganizations(
        session.session_token,
      )) as unknown as ListResponse<Organization>;
      setData(res);
      setLoadedFor(session.session_token);
    } finally {
      setIsRefreshing(false);
    }
  }, [client, session]);

  // The fetch is inline rather than a call to load(), because load() sets state
  // before it awaits anything and doing that inside an effect costs a render
  // pass for no gain. The ignore flag is the other half: without it a response
  // for a session the user has already left can overwrite a newer list.
  useEffect(() => {
    if (!isAuthenticated || !session) return;
    let ignore = false;
    void (async () => {
      try {
        const res = (await client.listOrganizations(
          session.session_token,
        )) as unknown as ListResponse<Organization>;
        if (!ignore) setData(res);
      } finally {
        if (!ignore) setLoadedFor(session.session_token);
      }
    })();
    return () => {
      ignore = true;
    };
  }, [isAuthenticated, client, session]);

  return {
    organizations: data.items,
    total: data.total,
    isLoading,
    reload: load,
  };
}

/**
 * useSessionToken returns the current session token (or null).
 *
 * Useful for passing to custom API calls.
 */
export function useSessionToken(): string | null {
  const { session } = useAuth();
  return session?.session_token ?? null;
}

/**
 * useClientConfig returns the auto-discovered client configuration.
 *
 * Requires a `publishableKey` on `AuthProvider` to fetch config from the backend.
 *
 * ```tsx
 * const { config, isLoaded } = useClientConfig();
 * if (config?.social?.enabled) {
 *   // Render social login buttons
 * }
 * ```
 */
export function useClientConfig(): {
  config: ClientConfig | null;
  isLoaded: boolean;
} {
  const { clientConfig, isConfigLoaded } = useAuth();
  return { config: clientConfig, isLoaded: isConfigLoaded };
}
