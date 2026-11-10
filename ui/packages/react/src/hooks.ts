/** Convenience hooks built on top of useAuth. */

import { useCallback, useEffect, useState, useSyncExternalStore } from "react";
import { readOrganizationList } from "@authsome/ui-core";
import type { ClientConfig, Organization, OrganizationList, PluginKey, User } from "@authsome/ui-core";
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
  /** How many organizations were loaded. */
  total: number;
  /** Cursor for the next page, when the server has more. */
  nextCursor?: string;
  isLoading: boolean;
  reload: () => Promise<void>;
} {
  const { client, session, isAuthenticated } = useAuth();
  const [data, setData] = useState<OrganizationList>({ organizations: [], total: 0 });
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
      const res = await client.listOrganizations(session.session_token);
      setData(readOrganizationList(res));
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
        const res = await client.listOrganizations(session.session_token);
        if (!ignore) setData(readOrganizationList(res));
      } finally {
        if (!ignore) setLoadedFor(session.session_token);
      }
    })();
    return () => {
      ignore = true;
    };
  }, [isAuthenticated, client, session]);

  return {
    organizations: data.organizations,
    total: data.total,
    nextCursor: data.nextCursor,
    isLoading,
    reload: load,
  };
}

/**
 * usePlugin returns a client plugin's API. The plugin has to be in the
 * `plugins` of the surrounding `<AuthProvider>`; if it isn't, this throws an
 * error that says which one to add.
 *
 * ```tsx
 * <AuthProvider baseURL={url} plugins={[subscription()]}>...</AuthProvider>
 * const billing = usePlugin(subscription);
 * ```
 */
export function usePlugin<Api>(plugin: PluginKey<Api>): Api {
  const { manager } = useAuth();
  return manager.plugin(plugin);
}

/**
 * useActiveOrg returns the session's active organization and a way to
 * switch it. It re-renders when the organization changes, including on
 * sign-in (restored from the session) and sign-out (cleared).
 *
 * ```tsx
 * const { orgId, switchOrg } = useActiveOrg();
 * ```
 */
export function useActiveOrg(): {
  orgId: string | null;
  switchOrg: (orgId: string) => Promise<void>;
  isSwitching: boolean;
} {
  const { manager } = useAuth();
  const subscribe = useCallback(
    (onChange: () => void) => {
      const offs = (["orgChanged", "signedIn", "signedOut"] as const).map((e) =>
        manager.on(e, onChange),
      );
      return () => offs.forEach((off) => off());
    },
    [manager],
  );
  const orgId = useSyncExternalStore(
    subscribe,
    () => manager.getActiveOrgId(),
    () => manager.getActiveOrgId(),
  );

  const [isSwitching, setIsSwitching] = useState(false);
  const switchOrg = useCallback(
    async (next: string) => {
      setIsSwitching(true);
      try {
        await manager.switchOrg(next);
      } finally {
        setIsSwitching(false);
      }
    },
    [manager],
  );

  return { orgId, switchOrg, isSwitching };
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

// ── Last sign-in ───────────────────────────────────────

/**
 * What the account summary shows. This is public API, because
 * `<AccountSummaryCard>` takes it as a prop so a host can render the card from
 * data of its own.
 */
export interface LastSignInSummary {
  /** When the current session began, ISO 8601, or null when unknown. */
  lastSignInAt: string | null;
  /** The IP that session came from. There is no geo lookup server-side yet. */
  ipAddress: string | null;
  /** Browser and OS of the device most recently seen, when known. */
  device: string | null;
  /** How many devices are marked trusted. */
  trustedDeviceCount: number;
  /**
   * Whether a second factor is enrolled. null means "not known yet": there is
   * no endpoint that answers it, so this stays null until one exists.
   */
  twoStepEnabled: boolean | null;
}

/** Narrows one untyped session record. `sessions` is Record<string, unknown>[]. */
function readSessionRecord(raw: Record<string, unknown>): {
  createdAt: string | null;
  ip: string | null;
} {
  const str = (k: string): string | null => {
    const v = raw[k];
    return typeof v === "string" && v.length > 0 ? v : null;
  };
  return { createdAt: str("created_at"), ip: str("ip_address") };
}

/**
 * useLastSignIn reports when and where this account last signed in.
 *
 * ```tsx
 * const { summary, isLoading } = useLastSignIn();
 * ```
 *
 * It is a reassurance surface, not an authorization one. Nothing here decides
 * what the user can do, so a host that cannot load it should carry on.
 */
export function useLastSignIn(): {
  summary: LastSignInSummary | null;
  isLoading: boolean;
  reload: () => Promise<void>;
} {
  const { client, session, isAuthenticated } = useAuth();
  const [summary, setSummary] = useState<LastSignInSummary | null>(null);
  const token = session?.session_token ?? null;

  // Same shape as useOrganizations: which token the data we hold was fetched
  // for, compared against the current one, so the automatic load's progress is
  // derived rather than stored and no setState happens inside the effect.
  const [loadedFor, setLoadedFor] = useState<string | null>(null);
  const [isRefreshing, setIsRefreshing] = useState(false);

  const isLoading = (isAuthenticated && loadedFor !== token) || isRefreshing;

  const fetchSummary = useCallback(
    async (sessionToken: string): Promise<LastSignInSummary> => {
      // Both calls are independent and neither blocks the other. A failure in
      // either is not worth failing the card over, so each falls back.
      const [sessions, devices] = await Promise.all([
        client.listSessions(sessionToken, 1).catch(() => null),
        client.listDevices(sessionToken).catch(() => null),
      ]);

      const newest = sessions?.sessions?.[0]
        ? readSessionRecord(sessions.sessions[0])
        : null;

      const seen = [...(devices?.devices ?? [])].sort((a, b) =>
        (b.last_seen_at ?? "").localeCompare(a.last_seen_at ?? ""),
      )[0];

      return {
        lastSignInAt: newest?.createdAt ?? seen?.last_seen_at ?? null,
        ipAddress: newest?.ip ?? seen?.ip_address ?? null,
        device:
          [seen?.browser, seen?.os].filter(Boolean).join(" on ") || seen?.name || null,
        trustedDeviceCount: (devices?.devices ?? []).filter((d) => d.trusted).length,
        twoStepEnabled: null,
      };
    },
    [client],
  );

  const load = useCallback(async () => {
    if (!session) return;
    setIsRefreshing(true);
    try {
      setSummary(await fetchSummary(session.session_token));
      setLoadedFor(session.session_token);
    } finally {
      setIsRefreshing(false);
    }
  }, [fetchSummary, session]);

  // Inline rather than calling load(), which sets state before it awaits. The
  // ignore flag stops a response for a session the user already left from
  // overwriting a newer one.
  useEffect(() => {
    if (!isAuthenticated || !session) return;
    let ignore = false;
    void (async () => {
      try {
        const next = await fetchSummary(session.session_token);
        if (!ignore) setSummary(next);
      } finally {
        if (!ignore) setLoadedFor(session.session_token);
      }
    })();
    return () => {
      ignore = true;
    };
  }, [isAuthenticated, session, fetchSummary]);

  return { summary, isLoading, reload: load };
}
