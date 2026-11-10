/** React hooks and components for organization invitations. */

import { useCallback, useEffect, useState } from "react";
import { useAuth, usePlugin } from "@authsome/ui-react";
import {
  organization,
  type Invitation,
  type InvitationPreview,
  type Member,
  type OrganizationApi,
} from "./index";

/**
 * useInvitation reads an invitation by its token.
 *
 * No session is needed, which is the point: the person deciding whether to
 * accept usually has no account yet.
 */
export function useInvitation(token: string | null): {
  invitation: InvitationPreview | null;
  isLoading: boolean;
  error: Error | null;
} {
  const api = usePlugin<OrganizationApi>(organization);
  const [invitation, setInvitation] = useState<InvitationPreview | null>(null);
  const [error, setError] = useState<Error | null>(null);
  const [loadedFor, setLoadedFor] = useState<string | null>(null);
  const isLoading = Boolean(token) && loadedFor !== token;

  useEffect(() => {
    if (!token) return;
    let ignore = false;
    void (async () => {
      try {
        const res = await api.getInvitation(token);
        if (!ignore) {
          setInvitation(res);
          setError(null);
        }
      } catch (err) {
        if (!ignore) {
          setInvitation(null);
          setError(err instanceof Error ? err : new Error(String(err)));
        }
      } finally {
        if (!ignore) setLoadedFor(token);
      }
    })();
    return () => {
      ignore = true;
    };
  }, [api, token]);

  return { invitation, isLoading, error };
}

/** useAcceptInvitation joins the org. Needs a session. */
export function useAcceptInvitation(): {
  accept: (token: string) => Promise<Member>;
  isSaving: boolean;
  error: Error | null;
} {
  const api = usePlugin<OrganizationApi>(organization);
  const [isSaving, setIsSaving] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  const accept = useCallback(
    async (token: string) => {
      setIsSaving(true);
      setError(null);
      try {
        return await api.acceptInvitation(token);
      } catch (err) {
        setError(err instanceof Error ? err : new Error(String(err)));
        throw err;
      } finally {
        setIsSaving(false);
      }
    },
    [api],
  );

  return { accept, isSaving, error };
}

/** useDeclineInvitation turns it down. */
export function useDeclineInvitation(): {
  decline: (token: string) => Promise<void>;
  isSaving: boolean;
  error: Error | null;
} {
  const api = usePlugin<OrganizationApi>(organization);
  const [isSaving, setIsSaving] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  const decline = useCallback(
    async (token: string) => {
      setIsSaving(true);
      setError(null);
      try {
        await api.declineInvitation(token);
      } catch (err) {
        setError(err instanceof Error ? err : new Error(String(err)));
        throw err;
      } finally {
        setIsSaving(false);
      }
    },
    [api],
  );

  return { decline, isSaving, error };
}

/** useInvitations lists an org's pending invitations. Admin side. */
export function useInvitations(orgId: string | null): {
  invitations: Invitation[];
  isLoading: boolean;
  error: Error | null;
  reload: () => Promise<void>;
} {
  const api = usePlugin<OrganizationApi>(organization);
  const { session, isAuthenticated } = useAuth();
  const [invitations, setInvitations] = useState<Invitation[]>([]);
  const [error, setError] = useState<Error | null>(null);
  const token = session?.session_token ?? null;
  const key = token && orgId ? `${token}:${orgId}` : null;
  const [loadedFor, setLoadedFor] = useState<string | null>(null);
  const [isRefreshing, setIsRefreshing] = useState(false);
  const isLoading = (isAuthenticated && Boolean(key) && loadedFor !== key) || isRefreshing;

  const reload = useCallback(async () => {
    if (!orgId || !key) return;
    setIsRefreshing(true);
    try {
      setInvitations((await api.listInvitations(orgId)).invitations);
      setError(null);
      setLoadedFor(key);
    } catch (err) {
      setError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setIsRefreshing(false);
    }
  }, [api, key, orgId]);

  useEffect(() => {
    if (!isAuthenticated || !orgId || !key) return;
    let ignore = false;
    void (async () => {
      try {
        const res = await api.listInvitations(orgId);
        if (!ignore) {
          setInvitations(res.invitations);
          setError(null);
        }
      } catch (err) {
        if (!ignore) setError(err instanceof Error ? err : new Error(String(err)));
      } finally {
        if (!ignore) setLoadedFor(key);
      }
    })();
    return () => {
      ignore = true;
    };
  }, [api, isAuthenticated, key, orgId]);

  return { invitations, isLoading, error, reload };
}

/** useCreateInvitation invites someone. */
export function useCreateInvitation(orgId: string | null): {
  invite: (input: { email: string; role?: string }) => Promise<Invitation>;
  isSaving: boolean;
  error: Error | null;
} {
  const api = usePlugin<OrganizationApi>(organization);
  const [isSaving, setIsSaving] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  const invite = useCallback(
    async (input: { email: string; role?: string }) => {
      if (!orgId) throw new Error("useCreateInvitation: no organization id");
      setIsSaving(true);
      setError(null);
      try {
        return await api.createInvitation(orgId, input);
      } catch (err) {
        setError(err instanceof Error ? err : new Error(String(err)));
        throw err;
      } finally {
        setIsSaving(false);
      }
    },
    [api, orgId],
  );

  return { invite, isSaving, error };
}

export { InvitationCard, type InvitationCardProps } from "./invitation-card";
export { InvitationList, type InvitationListProps } from "./invitation-list";
