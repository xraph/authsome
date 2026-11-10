import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { CurrentPlanCard, ManageBillingButton, PricingTable, UsageMeter } from "./components";
import { harness, json, planJSON, subJSON } from "./test-support";

describe("PricingTable", () => {
  it("lists the plans with prices and marks the current one", async () => {
    await harness().mount(<PricingTable />);
    await waitFor(() => expect(screen.getByText("Team")).toBeTruthy());
    expect(screen.getByText("$99.00")).toBeTruthy();
    expect(screen.getByText("Free")).toBeTruthy();
    expect((screen.getByRole("button", { name: "Current plan" }) as HTMLButtonElement).disabled).toBe(true);
  });

  it("changes plan when there's a subscription", async () => {
    const h = harness();
    const patch = vi.fn(() => json({ subscription: subJSON({ plan: planJSON("team", { amount_minor: 9900 }) }) }));
    h.routes.set("PATCH /v1/billing/subscription", patch);
    await h.mount(<PricingTable />);
    const choose = await screen.findByRole("button", { name: "Choose Team" });
    await act(async () => { fireEvent.click(choose); });
    expect(patch).toHaveBeenCalledTimes(1);
    await waitFor(() => expect(screen.getAllByRole("button", { name: "Current plan" })).toHaveLength(1));
  });

  it("subscribes when there isn't one, and hands a checkout URL to onCheckout", async () => {
    const h = harness();
    h.routes.set("GET /v1/billing/subscription", () => json({ subscription: null, can_manage: true }));
    h.routes.set("POST /v1/billing/subscription", () => json({ checkout_url: "https://pay.example/s/1" }));
    const onCheckout = vi.fn();
    await h.mount(<PricingTable onCheckout={onCheckout} returnUrl="https://app/billing" />);
    const choose = await screen.findByRole("button", { name: "Choose Pro" });
    await act(async () => { fireEvent.click(choose); });
    expect(onCheckout).toHaveBeenCalledWith("https://pay.example/s/1");
  });

  it("lets the app take over selection", async () => {
    const onSelect = vi.fn();
    await harness().mount(<PricingTable onSelect={onSelect} />);
    const choose = await screen.findByRole("button", { name: "Choose Team" });
    fireEvent.click(choose);
    expect(onSelect).toHaveBeenCalledWith(expect.objectContaining({ slug: "team" }));
  });

  it("hides the buttons from someone who can't change the plan", async () => {
    const h = harness();
    h.routes.set("GET /v1/billing/subscription", () => json({ subscription: subJSON(), can_manage: false }));
    await h.mount(<PricingTable />);
    await waitFor(() => expect(screen.getByText("Team")).toBeTruthy());
    expect(screen.queryByRole("button", { name: "Choose Team" })).toBeNull();
  });
});

describe("CurrentPlanCard", () => {
  it("shows the plan, price, renewal date and seats", async () => {
    const h = harness();
    h.routes.set("GET /v1/billing/subscription", () => json({
      can_manage: true,
      subscription: subJSON({ seats: { feature_key: "seats", used: 3, limit: 10 } }),
    }));
    await h.mount(<CurrentPlanCard locale="en-US" timeZone="UTC" />);
    await waitFor(() => expect(screen.getByText("Pro")).toBeTruthy());
    expect(screen.getByText("$29.00 / monthly")).toBeTruthy();
    expect(screen.getByText(/Renews on Dec 1, 2026/)).toBeTruthy();
    expect(screen.getByText("3 of 10 seats")).toBeTruthy();
    expect(screen.getByRole("button", { name: "Cancel plan" })).toBeTruthy();
  });

  it("says when a canceled plan ends and offers to keep it", async () => {
    const h = harness();
    h.routes.set("GET /v1/billing/subscription", () => json({
      can_manage: true, subscription: subJSON({ cancel_at: "2026-12-01T00:00:00Z" }),
    }));
    const resume = vi.fn(() => json({ subscription: subJSON() }));
    h.routes.set("POST /v1/billing/subscription/resume", resume);
    await h.mount(<CurrentPlanCard locale="en-US" timeZone="UTC" />);
    await waitFor(() => expect(screen.getByText(/Ends on Dec 1, 2026/)).toBeTruthy());
    await act(async () => { fireEvent.click(screen.getByRole("button", { name: "Keep plan" })); });
    expect(resume).toHaveBeenCalledTimes(1);
    await waitFor(() => expect(screen.getByText(/Renews on/)).toBeTruthy());
  });

  it("shows no buttons to someone who can't manage billing", async () => {
    const h = harness();
    h.routes.set("GET /v1/billing/subscription", () => json({ subscription: subJSON(), can_manage: false }));
    await h.mount(<CurrentPlanCard />);
    await waitFor(() => expect(screen.getByText("Pro")).toBeTruthy());
    expect(screen.queryByRole("button")).toBeNull();
  });

  it("has an empty state", async () => {
    const h = harness();
    h.routes.set("GET /v1/billing/subscription", () => json({ subscription: null, can_manage: true }));
    await h.mount(<CurrentPlanCard />);
    await waitFor(() => expect(screen.getByText("No plan yet")).toBeTruthy());
  });
});

describe("UsageMeter", () => {
  it("shows use against the limit", async () => {
    await harness().mount(<UsageMeter feature="sso" label="SSO connections" />);
    await waitFor(() => expect(screen.getByText("2 of 10")).toBeTruthy());
    expect(screen.getByText("SSO connections")).toBeTruthy();
    expect(screen.getByRole("progressbar").getAttribute("aria-valuenow")).toBe("2");
  });
});

describe("ManageBillingButton", () => {
  it("opens the portal", async () => {
    const h = harness();
    h.routes.set("POST /v1/billing/portal", () => json({ url: "https://pay.example/portal" }));
    const onOpen = vi.fn();
    await h.mount(<ManageBillingButton onOpen={onOpen} />);
    const button = await screen.findByRole("button", { name: "Manage billing" });
    await act(async () => { fireEvent.click(button); });
    expect(onOpen).toHaveBeenCalledWith("https://pay.example/portal");
  });

  it("isn't shown to someone who can't manage billing", async () => {
    const h = harness();
    h.routes.set("GET /v1/billing/subscription", () => json({ subscription: subJSON(), can_manage: false }));
    await h.mount(<ManageBillingButton />);
    await waitFor(() => expect(h.calls).toContain("GET /v1/billing/subscription"));
    expect(screen.queryByRole("button")).toBeNull();
  });
});
