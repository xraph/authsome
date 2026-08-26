/**
 * Subscribes to history navigation.
 *
 * Shared by the hooks that read the current location. They read it through
 * useSyncExternalStore rather than mirroring it into state, because the
 * location is an external store and treating it as one is what keeps them
 * from having to resync it from an effect.
 */
export function subscribeToPopState(onChange: () => void): () => void {
  window.addEventListener("popstate", onChange);
  return () => window.removeEventListener("popstate", onChange);
}
