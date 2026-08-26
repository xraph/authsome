import { act, render, screen, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, it } from "vitest";

import { AuthProvider, useAuth } from "./context";

// These run against the real AuthManager rather than a mock. Without a
// publishableKey it never fetches config, and with no storage configured it
// falls back to in-memory storage, so a mounted provider settles to
// "unauthenticated" without touching the network.
const BASE = "https://api.example.test";

/**
 * Renders a provider and records the context on every render.
 *
 * mount and rerender are async and wrapped in act because the provider starts
 * an async initialize on mount. Letting that resolve outside act is what
 * produces React's "not wrapped in act" warning, and a test suite that prints
 * warnings on a good run is one nobody reads on a bad one.
 */
function capture() {
  const seen: ReturnType<typeof useAuth>[] = [];
  function Probe() {
    seen.push(useAuth());
    return <span data-testid="status">{seen[seen.length - 1].state.status}</span>;
  }
  const tree = (extra?: Record<string, unknown>, children: ReactNode = <Probe />) => (
    <AuthProvider baseURL={BASE} {...extra}>
      {children}
    </AuthProvider>
  );

  let rerenderFn: ((ui: ReactNode) => void) | null = null;
  let unmountFn: (() => void) | null = null;

  return {
    seen,
    async mount(extra?: Record<string, unknown>) {
      await act(async () => {
        const r = render(tree(extra));
        rerenderFn = r.rerender;
        unmountFn = r.unmount;
      });
    },
    async rerender(extra?: Record<string, unknown>) {
      await act(async () => rerenderFn?.(tree(extra)));
    },
    unmount() {
      unmountFn?.();
    },
  };
}

describe("AuthProvider", () => {
  // The manager owns the session, the refresh timer and every subscriber. If a
  // re-render replaced it, in-flight state would be dropped and the mount
  // effect would tear down and re-subscribe for nothing. This is the invariant
  // the old ref-during-render construction existed to provide.
  it("keeps one manager across re-renders", async () => {
    const c = capture();
    await c.mount();
    await c.rerender();
    await c.rerender();

    expect(c.seen.length).toBeGreaterThanOrEqual(3);
    expect(new Set(c.seen.map((x) => x.manager)).size).toBe(1);
  });

  // A changing prop must not rebuild it either. Construction closes over the
  // first render's config on purpose.
  it("keeps the same manager when props change", async () => {
    const c = capture();
    await c.mount();
    await c.rerender({ onError: () => {} });

    expect(new Set(c.seen.map((x) => x.manager)).size).toBe(1);
  });

  // Each provider owns its own manager, so a fresh mount must not reuse the
  // last one. This catches a manager hoisted to module scope.
  it("builds a new manager for a separate mount", async () => {
    const first = capture();
    await first.mount();
    const managerA = first.seen[0].manager;
    first.unmount();

    const second = capture();
    await second.mount();

    expect(second.seen[0].manager).not.toBe(managerA);
  });

  it("takes its first state from the manager rather than a placeholder", async () => {
    const c = capture();
    await c.mount();
    expect(c.seen[0].state.status).toBe("idle");
    expect(c.seen[0].isAuthenticated).toBe(false);
  });

  // Proves the mount effect really subscribed and called initialize: the
  // manager resolves storage, finds nothing, and reports unauthenticated.
  it("subscribes to the manager and follows it out of idle", async () => {
    const c = capture();
    await c.mount();
    await waitFor(() =>
      expect(screen.getByTestId("status").textContent).toBe("unauthenticated"),
    );
  });

  it("exposes the client and the action callbacks", async () => {
    const c = capture();
    await c.mount();
    const ctx = c.seen[0];
    expect(ctx.client).toBeTruthy();
    for (const fn of ["signIn", "signUp", "signOut", "submitMFACode"] as const) {
      expect(typeof ctx[fn]).toBe("function");
    }
  });
});

describe("useAuth", () => {
  it("refuses to run outside a provider", () => {
    function Orphan() {
      useAuth();
      return null;
    }
    expect(() => render(<Orphan />)).toThrow(/within an <AuthProvider>/);
  });
});
