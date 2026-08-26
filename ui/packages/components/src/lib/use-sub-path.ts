import { useCallback, useSyncExternalStore } from "react";

import { subscribeToPopState } from "./pop-state";

/**
 * Extracts the sub-path segment after the base path.
 *
 * E.g. for base="/sign-in" and pathname="/sign-in/forgot-password", returns
 * "forgot-password".
 */
export function extractSubPath(
  pathname: string,
  basePath: string,
): string | undefined {
  const normalized = basePath.replace(/\/+$/, "");
  if (!pathname.startsWith(normalized)) return undefined;
  const rest = pathname.slice(normalized.length).replace(/^\/+/, "");
  return rest || undefined;
}

/**
 * The sub-path of the current location relative to `basePath`, kept in step
 * with history navigation.
 *
 * The location is an external store, so this reads it as one rather than
 * mirroring it into state and resyncing from an effect. That version had to
 * re-derive the value at the top of every effect run, which reads as dead code
 * next to the useState initializer and is the obvious line to delete — but
 * deleting it silently stops the hook resyncing when `basePath` changes.
 * useSyncExternalStore has no such line to lose: a new basePath is a new
 * snapshot function, and React re-reads it.
 *
 * The third argument is the server snapshot. There is no location to read
 * during SSR, so it reports undefined, which is what the useState initializer's
 * `typeof window === "undefined"` guard did.
 */
export function useSubPath(basePath: string): string | undefined {
  const getSnapshot = useCallback(
    () => extractSubPath(window.location.pathname, basePath),
    [basePath],
  );

  return useSyncExternalStore(subscribeToPopState, getSnapshot, () => undefined);
}
