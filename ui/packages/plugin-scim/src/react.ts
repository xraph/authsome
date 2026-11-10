/** React hooks and components for SCIM administration. */

import { useCallback, useEffect, useState } from "react";
import { useAuth, usePlugin } from "@authsome/ui-react";
import {
  scim,
  type CreateScimConfigInput,
  type ScimApi,
  type ScimConfig,
  type ScimToken,
  type ScimTokenCreated,
  type UpdateScimConfigInput,
} from "./index";

/** useScimConfigs lists an organization's identity provider connections. */
export function useScimConfigs(orgId: string | null): {
  configs: ScimConfig[];
  isLoading: boolean;
  error: Error | null;
  reload: () => Promise<void>;
} {
  const api = usePlugin<ScimApi>(scim);
  const { session, isAuthenticated } = useAuth();
  const [configs, setConfigs] = useState<ScimConfig[]>([]);
  const [error, setError] = useState<Error | null>(null);
  const token = session?.session_token ?? null;
  const key = token && orgId ? `${token}:${orgId}` : null;
  const [loadedFor, setLoadedFor] = useState<string | null>(null);
  const [isRefreshing, setIsRefreshing] = useState(false);
  const isLoading =
    (isAuthenticated && Boolean(key) && loadedFor !== key) || isRefreshing;

  const reload = useCallback(async () => {
    if (!orgId || !key) return;
    setIsRefreshing(true);
    try {
      setConfigs((await api.listConfigs(orgId)).configs);
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
        const res = await api.listConfigs(orgId);
        if (!ignore) {
          setConfigs(res.configs);
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

  return { configs, isLoading, error, reload };
}

/** useScimTokens lists the tokens for one connection. */
export function useScimTokens(configId: string | null): {
  tokens: ScimToken[];
  isLoading: boolean;
  error: Error | null;
  reload: () => Promise<void>;
} {
  const api = usePlugin<ScimApi>(scim);
  const { session, isAuthenticated } = useAuth();
  const [tokens, setTokens] = useState<ScimToken[]>([]);
  const [error, setError] = useState<Error | null>(null);
  const token = session?.session_token ?? null;
  const key = token && configId ? `${token}:${configId}` : null;
  const [loadedFor, setLoadedFor] = useState<string | null>(null);
  const [isRefreshing, setIsRefreshing] = useState(false);
  const isLoading =
    (isAuthenticated && Boolean(key) && loadedFor !== key) || isRefreshing;

  const reload = useCallback(async () => {
    if (!configId || !key) return;
    setIsRefreshing(true);
    try {
      setTokens((await api.listTokens(configId)).tokens);
      setError(null);
      setLoadedFor(key);
    } catch (err) {
      setError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setIsRefreshing(false);
    }
  }, [api, configId, key]);

  useEffect(() => {
    if (!isAuthenticated || !configId || !key) return;
    let ignore = false;
    void (async () => {
      try {
        const res = await api.listTokens(configId);
        if (!ignore) {
          setTokens(res.tokens);
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
  }, [api, configId, isAuthenticated, key]);

  return { tokens, isLoading, error, reload };
}

/** One shape for the writes, because they all behave the same way. */
function useScimMutation<Args extends unknown[], R>(
  run: (api: ScimApi, ...args: Args) => Promise<R>,
): { mutate: (...args: Args) => Promise<R>; isSaving: boolean; error: Error | null } {
  const api = usePlugin<ScimApi>(scim);
  const [isSaving, setIsSaving] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  const mutate = useCallback(
    async (...args: Args) => {
      setIsSaving(true);
      setError(null);
      try {
        return await run(api, ...args);
      } catch (err) {
        setError(err instanceof Error ? err : new Error(String(err)));
        throw err;
      } finally {
        setIsSaving(false);
      }
    },
    [api, run],
  );

  return { mutate, isSaving, error };
}

export function useCreateScimConfig() {
  const { mutate, isSaving, error } = useScimMutation(
    (api, orgId: string, input: CreateScimConfigInput) =>
      api.createConfig(orgId, input),
  );
  return { create: mutate, isSaving, error };
}

export function useUpdateScimConfig() {
  const { mutate, isSaving, error } = useScimMutation(
    (api, configId: string, input: UpdateScimConfigInput) =>
      api.updateConfig(configId, input),
  );
  return { update: mutate, isSaving, error };
}

export function useDeleteScimConfig() {
  const { mutate, isSaving, error } = useScimMutation(
    (api, configId: string) => api.deleteConfig(configId),
  );
  return { remove: mutate, isSaving, error };
}

export function useGenerateScimToken() {
  const { mutate, isSaving, error } = useScimMutation(
    (api, configId: string, input: { name: string; expires_at?: string }) =>
      api.createToken(configId, input),
  );
  return { generate: mutate as (...a: [string, { name: string; expires_at?: string }]) => Promise<ScimTokenCreated>, isSaving, error };
}

export function useRotateScimToken() {
  const { mutate, isSaving, error } = useScimMutation(
    (api, tokenId: string, input?: { name?: string; grace_seconds?: number }) =>
      api.rotateToken(tokenId, input),
  );
  return { rotate: mutate, isSaving, error };
}

export function useRevokeScimToken() {
  const { mutate, isSaving, error } = useScimMutation(
    (api, tokenId: string) => api.revokeToken(tokenId),
  );
  return { revoke: mutate, isSaving, error };
}

export {
  ScimConnectionCard,
  type ScimConnectionCardProps,
} from "./scim-connection-card";
export { ScimTokenList, type ScimTokenListProps } from "./scim-token-list";
