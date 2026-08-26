import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { AuthClient } from "@authsome/ui-core";
import { describe, expect, it } from "vitest";

import { DeviceList } from "./device-list";
import {
  BASE,
  json,
  makeSession,
  routedFetch,
  stubAuth,
  withAuth,
  withProvider,
} from "../test-support";

/** Skeleton is the only thing in this tree with animate-pulse. */
function showsLoading(container: HTMLElement): boolean {
  return container.querySelector(".animate-pulse") !== null;
}

const laptop = {
  id: "dev_1",
  name: "Ada's laptop",
  browser: "Firefox",
  os: "Linux",
  last_seen_at: new Date().toISOString(),
  trusted: true,
  type: "desktop",
  created_at: new Date(0).toISOString(),
  user_id: "usr_1",
  app_id: "app_1",
  updated_at: new Date(0).toISOString(),
};

describe("DeviceList", () => {
  it("shows the loading skeletons until the devices arrive", async () => {
    let release!: () => void;
    const gate = new Promise<void>((r) => (release = r));

    const { fetchFn } = routedFetch({
      "GET /v1/devices": async () => {
        await gate;
        return { devices: [laptop] };
      },
    });

    const { container } = render(
      withProvider(<DeviceList />, { fetch: fetchFn }),
    );

    // In flight: skeletons, no device row.
    await waitFor(() => expect(showsLoading(container)).toBe(true));
    expect(screen.queryByText("Ada's laptop")).toBeNull();

    release();

    await waitFor(() => expect(screen.getByText("Ada's laptop")).toBeTruthy());
    expect(showsLoading(container)).toBe(false);
  });

  it("reports loading again when the session token changes", async () => {
    // The regression guard. A rewrite that only sets isLoading after the await
    // still passes the mount case, because isLoading starts true. It fails
    // here: the second load has to put the component back into loading, or the
    // user stares at the previous account's devices with no spinner.
    const seen: string[] = [];
    let release!: () => void;
    let gate = Promise.resolve();

    const { fetchFn } = routedFetch({
      "GET /v1/devices": async ({ token }) => {
        seen.push(token ?? "none");
        await gate;
        return { devices: [{ ...laptop, name: `laptop for ${token}` }] };
      },
    });

    const client = new AuthClient({ baseURL: BASE, fetch: fetchFn });
    const first = stubAuth({ fetch: fetchFn, session: makeSession("tok-1"), client });

    const { container, rerender } = render(
      withAuth(<DeviceList />, first),
    );

    await waitFor(() =>
      expect(screen.getByText("laptop for tok-1")).toBeTruthy(),
    );
    expect(showsLoading(container)).toBe(false);

    // Second load, held open so the in-flight state is observable.
    gate = new Promise<void>((r) => (release = r));
    const second = stubAuth({
      fetch: fetchFn,
      session: makeSession("tok-2"),
      client,
    });
    rerender(withAuth(<DeviceList />, second));

    await waitFor(() => expect(showsLoading(container)).toBe(true));
    expect(screen.queryByText("laptop for tok-1")).toBeNull();

    release();
    await waitFor(() =>
      expect(screen.getByText("laptop for tok-2")).toBeTruthy(),
    );
    expect(seen).toEqual(["tok-1", "tok-2"]);
  });

  it("surfaces the server's message when the load fails", async () => {
    const { fetchFn } = routedFetch({
      "GET /v1/devices": () => json({ error: "devices unavailable" }, 500),
    });

    const { container } = render(
      withProvider(<DeviceList />, { fetch: fetchFn }),
    );

    await waitFor(() =>
      expect(screen.getByRole("alert").textContent).toContain(
        "devices unavailable",
      ),
    );
    // The error clears the loading state rather than leaving both on screen.
    expect(showsLoading(container)).toBe(false);
  });

  it("reloads through the loading state after trusting a device, then reports", async () => {
    // Pins two things the fix must not disturb: the refetch a handler triggers
    // goes through the same loading state as any other load, and onTrust fires
    // after that refetch rather than before it.
    const order: string[] = [];
    let trusted = false;
    let release!: () => void;
    let gate = Promise.resolve();

    const { fetchFn } = routedFetch({
      "GET /v1/devices": async () => {
        await gate;
        order.push("list");
        return { devices: [{ ...laptop, trusted }] };
      },
      "PATCH /v1/devices/dev_1/trust": () => {
        order.push("trust");
        trusted = true;
        return { ...laptop, trusted: true };
      },
    });

    const seen: string[] = [];
    const { container } = render(
      withProvider(<DeviceList onTrust={(id) => seen.push(id)} />, {
        fetch: fetchFn,
      }),
    );

    await waitFor(() => expect(screen.getByText("Untrusted")).toBeTruthy());

    gate = new Promise<void>((r) => (release = r));
    fireEvent.click(screen.getByTitle("Trust device"));

    await waitFor(() => expect(showsLoading(container)).toBe(true));
    expect(seen).toEqual([]);

    release();
    await waitFor(() => expect(seen).toEqual(["dev_1"]));
    expect(order).toEqual(["list", "trust", "list"]);
  });
});
