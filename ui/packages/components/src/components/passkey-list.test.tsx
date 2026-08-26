import { render, screen, waitFor } from "@testing-library/react";
import { AuthClient } from "@authsome/ui-core";
import { describe, expect, it } from "vitest";

import { PasskeyList } from "./passkey-list";
import {
  BASE,
  json,
  makeSession,
  routedFetch,
  stubAuth,
  withAuth,
  withProvider,
} from "../test-support";

function showsLoading(container: HTMLElement): boolean {
  return container.querySelector(".animate-pulse") !== null;
}

function makeCredential(name: string) {
  return {
    id: `cred_${name}`,
    display_name: name,
    transport: ["internal"],
    created_at: new Date(0).toISOString(),
  };
}

describe("PasskeyList", () => {
  it("shows the loading skeletons until the passkeys arrive", async () => {
    let release!: () => void;
    const gate = new Promise<void>((r) => (release = r));

    const { fetchFn } = routedFetch({
      "GET /v1/passkeys": async () => {
        await gate;
        return { credentials: [makeCredential("Yubikey")] };
      },
    });

    const { container } = render(
      withProvider(<PasskeyList />, { fetch: fetchFn }),
    );

    await waitFor(() => expect(showsLoading(container)).toBe(true));
    release();

    await waitFor(() => expect(screen.getByText("Yubikey")).toBeTruthy());
    expect(showsLoading(container)).toBe(false);
  });

  it("settles on the empty state rather than a spinner when signed out", async () => {
    // The branch that makes this component different from the other two
    // lists: with no token it stops loading instead of waiting forever. A fix
    // that derives loading purely from "have I loaded for this token" would
    // spin here for good.
    const { fetchFn, calls } = routedFetch({});

    const { container } = render(
      withProvider(<PasskeyList />, { fetch: fetchFn, session: null }),
    );

    await waitFor(() =>
      expect(screen.getByText("No passkeys registered")).toBeTruthy(),
    );
    expect(showsLoading(container)).toBe(false);
    expect(calls).not.toContain("GET /v1/passkeys");
  });

  it("reports loading again when the session token changes", async () => {
    let release!: () => void;
    let gate = Promise.resolve();

    const { fetchFn } = routedFetch({
      "GET /v1/passkeys": async ({ token }) => {
        await gate;
        return { credentials: [makeCredential(`key-${token}`)] };
      },
    });

    const client = new AuthClient({ baseURL: BASE, fetch: fetchFn });
    const { container, rerender } = render(
      withAuth(
        <PasskeyList />,
        stubAuth({ fetch: fetchFn, session: makeSession("tok-1"), client }),
      ),
    );

    await waitFor(() => expect(screen.getByText("key-tok-1")).toBeTruthy());
    expect(showsLoading(container)).toBe(false);

    gate = new Promise<void>((r) => (release = r));
    rerender(
      withAuth(
        <PasskeyList />,
        stubAuth({ fetch: fetchFn, session: makeSession("tok-2"), client }),
      ),
    );

    await waitFor(() => expect(showsLoading(container)).toBe(true));
    expect(screen.queryByText("key-tok-1")).toBeNull();

    release();
    await waitFor(() => expect(screen.getByText("key-tok-2")).toBeTruthy());
  });

  it("surfaces the server's message when the load fails", async () => {
    const { fetchFn } = routedFetch({
      "GET /v1/passkeys": () => json({ error: "passkeys unavailable" }, 500),
    });

    const { container } = render(
      withProvider(<PasskeyList />, { fetch: fetchFn }),
    );

    await waitFor(() =>
      expect(screen.getByRole("alert").textContent).toContain(
        "passkeys unavailable",
      ),
    );
    expect(showsLoading(container)).toBe(false);
  });
});
