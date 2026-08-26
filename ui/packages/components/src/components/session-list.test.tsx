import { render, screen, waitFor } from "@testing-library/react";
import { AuthClient } from "@authsome/ui-core";
import { describe, expect, it } from "vitest";

import { SessionList } from "./session-list";
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

function makeRow(device: string) {
  return {
    id: `sess_${device}`,
    device,
    browser: "Firefox",
    ip_address: "10.0.0.1",
    last_active: new Date().toISOString(),
    created_at: new Date(0).toISOString(),
    session_token: "other",
  };
}

describe("SessionList", () => {
  it("shows the loading skeletons until the sessions arrive", async () => {
    let release!: () => void;
    const gate = new Promise<void>((r) => (release = r));

    const { fetchFn } = routedFetch({
      "GET /v1/sessions": async () => {
        await gate;
        return { sessions: [makeRow("Thinkpad")] };
      },
    });

    const { container } = render(
      withProvider(<SessionList />, { fetch: fetchFn }),
    );

    await waitFor(() => expect(showsLoading(container)).toBe(true));
    expect(screen.queryByText(/Thinkpad/)).toBeNull();

    release();

    await waitFor(() => expect(screen.getByText(/Thinkpad/)).toBeTruthy());
    expect(showsLoading(container)).toBe(false);
  });

  it("reports loading again when the session token changes", async () => {
    // Family A regression guard: see device-list.test.tsx. A fix that only
    // sets loading after the await still passes the mount case and fails here.
    let release!: () => void;
    let gate = Promise.resolve();

    const { fetchFn } = routedFetch({
      "GET /v1/sessions": async ({ token }) => {
        await gate;
        return { sessions: [makeRow(`box-${token}`)] };
      },
    });

    const client = new AuthClient({ baseURL: BASE, fetch: fetchFn });
    const first = stubAuth({
      fetch: fetchFn,
      session: makeSession("tok-1"),
      client,
    });

    const { container, rerender } = render(withAuth(<SessionList />, first));
    await waitFor(() => expect(screen.getByText(/box-tok-1/)).toBeTruthy());
    expect(showsLoading(container)).toBe(false);

    gate = new Promise<void>((r) => (release = r));
    rerender(
      withAuth(
        <SessionList />,
        stubAuth({ fetch: fetchFn, session: makeSession("tok-2"), client }),
      ),
    );

    await waitFor(() => expect(showsLoading(container)).toBe(true));
    expect(screen.queryByText(/box-tok-1/)).toBeNull();

    release();
    await waitFor(() => expect(screen.getByText(/box-tok-2/)).toBeTruthy());
  });

  it("surfaces the server's message when the load fails", async () => {
    const { fetchFn } = routedFetch({
      "GET /v1/sessions": () => json({ error: "sessions unavailable" }, 500),
    });

    const { container } = render(
      withProvider(<SessionList />, { fetch: fetchFn }),
    );

    await waitFor(() =>
      expect(screen.getByRole("alert").textContent).toContain(
        "sessions unavailable",
      ),
    );
    expect(showsLoading(container)).toBe(false);
  });
});
