import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { OrgSwitcher } from "./org-switcher";
import { makeUser, routedFetch, withProvider } from "../test-support";

const orgs = [
  { id: "org_1", name: "Acme", slug: "acme" },
  { id: "org_2", name: "Globex", slug: "globex" },
];

function open(name: string) {
  const trigger = screen.getByRole("button", { name: new RegExp(name) });
  fireEvent.pointerDown(trigger, { button: 0, ctrlKey: false, pointerType: "mouse" });
}

describe("OrgSwitcher", () => {
  it("shows and switches the session's active org when uncontrolled", async () => {
    const switched: string[] = [];
    const { fetchFn, calls } = routedFetch({
      "GET /v1/me": () => ({ ...makeUser(), active_org_id: "org_1" }),
      "GET /v1/orgs": () => ({ organizations: orgs }),
      "POST /v1/me/switch-org": () => {
        switched.push("org_2");
        return { session_id: "ses_1", org_id: "org_2" };
      },
    });

    render(withProvider(<OrgSwitcher />, { fetch: fetchFn }));
    await waitFor(() => expect(screen.getByRole("button", { name: /Acme/ })).toBeTruthy());

    open("Acme");
    const item = await screen.findByRole("menuitem", { name: /Globex/ });
    await act(async () => { fireEvent.click(item); });

    await waitFor(() => expect(calls).toContain("POST /v1/me/switch-org"));
    await waitFor(() => expect(screen.getByRole("button", { name: /Globex/ })).toBeTruthy());
  });

  it("tells the user when a switch fails", async () => {
    const onError = vi.fn();
    const { fetchFn } = routedFetch({
      "GET /v1/me": () => ({ ...makeUser(), active_org_id: "org_1" }),
      "GET /v1/orgs": () => ({ organizations: orgs }),
      "POST /v1/me/switch-org": () => new Response(JSON.stringify({ error: "conflict" }), { status: 409 }),
    });

    render(withProvider(<OrgSwitcher onError={onError} />, { fetch: fetchFn }));
    await waitFor(() => expect(screen.getByRole("button", { name: /Acme/ })).toBeTruthy());

    open("Acme");
    const item = await screen.findByRole("menuitem", { name: /Globex/ });
    await act(async () => { fireEvent.click(item); });

    await waitFor(() => expect(screen.getByRole("alert").textContent).toMatch(/couldn't switch/i));
    expect(onError).toHaveBeenCalledTimes(1);
    expect(screen.getByRole("button", { name: /Acme/ })).toBeTruthy();
  });

  it("leaves switching to the app when it passes onOrgChange", async () => {
    const onOrgChange = vi.fn();
    const { fetchFn, calls } = routedFetch({
      "GET /v1/orgs": () => ({ organizations: orgs }),
    });

    render(withProvider(<OrgSwitcher activeOrgId="org_2" onOrgChange={onOrgChange} />, { fetch: fetchFn }));
    await waitFor(() => expect(screen.getByRole("button", { name: /Globex/ })).toBeTruthy());

    open("Globex");
    const item = await screen.findByRole("menuitem", { name: /Acme/ });
    await act(async () => { fireEvent.click(item); });

    expect(onOrgChange).toHaveBeenCalledWith("org_1");
    expect(calls).not.toContain("POST /v1/me/switch-org");
  });
});
