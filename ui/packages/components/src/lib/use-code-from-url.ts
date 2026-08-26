import { useSyncExternalStore } from "react";

import { subscribeToPopState } from "./pop-state";

/**
 * Reads user_code or code from the current URL query params.
 * Supports both raw codes (`ABCDEFGH`) and dash-formatted (`ABCD-EFGH`).
 */
export function parseCodeFromSearch(search: string): string | undefined {
  const params = new URLSearchParams(search);
  const raw = params.get("user_code") ?? params.get("code");
  if (!raw) return undefined;
  const cleaned = raw.replace(/[^A-Z0-9]/gi, "").toUpperCase();
  return cleaned || undefined;
}

function getSnapshot(): string | undefined {
  return parseCodeFromSearch(window.location.search);
}

/**
 * The device code carried in the current URL, kept in step with history
 * navigation. Reads the location as the external store it is, so there is no
 * mirrored state for an effect to resync. Reports undefined during SSR, which
 * is what the previous useState initializer's window guard did.
 */
export function useCodeFromURL(): string | undefined {
  return useSyncExternalStore(subscribeToPopState, getSnapshot, () => undefined);
}
