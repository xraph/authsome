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
  // The slashes are trimmed by scanning rather than with a regex. /\/+$/ is
  // polynomial on a run of slashes that does not end the string: the engine
  // consumes the run from every start position, fails the $, and backtracks
  // through the whole run before trying the next one. It cost 46ms at 10k
  // slashes and 4.2s at 100k. basePath is the caller's `path` prop, so it is
  // input this module does not control, which is why CodeQL reports it as
  // js/polynomial-redos. Scanning does the same job in linear time.
  let end = basePath.length;
  while (end > 0 && basePath[end - 1] === "/") end--;
  const normalized = basePath.slice(0, end);

  if (!pathname.startsWith(normalized)) return undefined;

  let start = normalized.length;
  while (start < pathname.length && pathname[start] === "/") start++;

  return pathname.slice(start) || undefined;
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
