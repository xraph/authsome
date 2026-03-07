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
  const [localUser, setLocalUser] = useState<User | null>(user);

  useEffect(() => {
    setLocalUser(user);
  }, [user]);

  const reload = useCallback(async () => {
    if (!session) return;
    const u = await client.getMe(session.session_token);
    setLocalUser(u);
  }, [client, session]);

  return { user: localUser, isLoading, reload };
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
  const [isLoading, setIsLoading] = useState(false);

  const load = useCallback(async () => {
    if (!session) return;
    setIsLoading(true);
    try {
      const res = await client.listOrganizations(session.session_token) as unknown as ListResponse<Organization>;
      setData(res);
    } finally {
      setIsLoading(false);
    }
  }, [client, session]);

  useEffect(() => {
    if (isAuthenticated) {
      void load();
    }
  }, [isAuthenticated, load]);

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
