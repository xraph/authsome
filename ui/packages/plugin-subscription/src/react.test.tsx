import { act, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Entitled, usePlans, useSubscription } from "./react";
import { harness, json, planJSON, subJSON } from "./test-support";

describe("useSubscription", () => {
  it("loads on mount and re-renders after a plan change", async () => {
    const h = harness();
    h.routes.set("PATCH /v1/billing/subscription", () => json({ subscription: subJSON({ plan: planJSON("team") }) }));
    let seen = null as unknown as ReturnType<typeof useSubscription>;
    function Probe() {
      seen = useSubscription();
      return <span data-testid="plan">{seen.isLoading ? "loading" : seen.plan?.name ?? "none"}</span>;
    }
    await h.mount(<Probe />);
    await waitFor(() => expect(screen.getByTestId("plan").textContent).toBe("Pro"));
    expect(seen.canManage).toBe(true);

    await act(async () => {
      const result = await seen.changePlan("team");
      expect(result.type).toBe("changed");
    });
    expect(screen.getByTestId("plan").textContent).toBe("Team");
    expect(seen.isPending).toBe(false);
  });

  it("reports pending while a write is in flight", async () => {
    const h = harness();
    let release!: () => void;
    const gate = new Promise<void>((r) => (release = r));
    h.routes.set("POST /v1/billing/subscription/cancel", async () => {
      await gate;
      return json({ subscription: subJSON({ cancel_at: "2026-12-01T00:00:00Z" }) });
    });
    let seen = null as unknown as ReturnType<typeof useSubscription>;
    function Probe() {
      seen = useSubscription();
      return null;
    }
    await h.mount(<Probe />);
    await waitFor(() => expect(seen.subscription).not.toBeNull());

    let canceling!: Promise<unknown>;
    act(() => {
      canceling = seen.cancel();
    });
    await waitFor(() => expect(seen.isPending).toBe(true));
    await act(async () => {
      release();
      await canceling;
    });
    expect(seen.isPending).toBe(false);
    expect(seen.subscription?.willCancel).toBe(true);
  });
});

describe("sharing", () => {
  it("makes one request pair however many components read the subscription", async () => {
    const h = harness();
    function Probe() {
      useSubscription();
      return null;
    }
    await h.mount(
      <>
        <Probe />
        <Probe />
        <Entitled feature="sso"><span /></Entitled>
        <Entitled feature="audit"><span /></Entitled>
      </>,
    );
    await waitFor(() => expect(h.calls.filter((c) => c === "GET /v1/billing/subscription")).toHaveLength(1));
    expect(h.calls.filter((c) => c === "GET /v1/billing/entitlements")).toHaveLength(1);
  });

  it("keeps the write functions stable between renders", async () => {
    const h = harness();
    const seen: ReturnType<typeof useSubscription>[] = [];
    function Probe() {
      seen.push(useSubscription());
      return null;
    }
    await h.mount(<Probe />);
    await waitFor(() => expect(seen.at(-1)?.subscription).not.toBeNull());
    const first = seen[0];
    const last = seen.at(-1)!;
    expect(last.changePlan).toBe(first.changePlan);
    expect(last.cancel).toBe(first.cancel);
    expect(last.refresh).toBe(first.refresh);
  });
});

describe("usePlans", () => {
  it("loads the plans on mount", async () => {
    let seen = null as unknown as ReturnType<typeof usePlans>;
    function Probe() {
      seen = usePlans();
      return null;
    }
    await harness().mount(<Probe />);
    await waitFor(() => expect(seen.plans.map((p) => p.slug)).toEqual(["free", "pro", "team"]));
    expect(seen.isLoading).toBe(false);
  });
});

describe("Entitled", () => {
  it("shows the loading slot, then the children when the plan allows it", async () => {
    const h = harness();
    let release!: () => void;
    const gate = new Promise<void>((r) => (release = r));
    const original = h.routes.get("GET /v1/billing/entitlements")!;
    h.routes.set("GET /v1/billing/entitlements", async () => {
      await gate;
      return original();
    });
    await h.mount(
      <Entitled feature="sso" fallback={<span>upgrade</span>} loading={<span>checking</span>}>
        <span>sso-settings</span>
      </Entitled>,
    );
    expect(screen.queryByText("checking")).not.toBeNull();
    expect(screen.queryByText("sso-settings")).toBeNull();
    await act(async () => release());
    await waitFor(() => expect(screen.queryByText("sso-settings")).not.toBeNull());
  });

  it("falls back for a feature the plan doesn't have", async () => {
    await harness().mount(
      <Entitled feature="audit_log" fallback={<span>upgrade</span>}>
        <span>audit</span>
      </Entitled>,
    );
    await waitFor(() => expect(screen.queryByText("upgrade")).not.toBeNull());
    expect(screen.queryByText("audit")).toBeNull();
  });

  it("falls back when the server has no subscription plugin", async () => {
    const h = harness({ serverPlugins: ["organization"] });
    await h.mount(
      <Entitled feature="sso" fallback={<span>upgrade</span>}>
        <span>sso-settings</span>
      </Entitled>,
    );
    await waitFor(() => expect(screen.queryByText("upgrade")).not.toBeNull());
    expect(h.calls.some((c) => c.includes("/v1/billing"))).toBe(false);
  });

  it("falls back on an error", async () => {
    const h = harness();
    h.routes.set("GET /v1/billing/entitlements", () => json({ error: "down" }, 500));
    await h.mount(
      <Entitled feature="sso" fallback={<span>upgrade</span>}>
        <span>sso-settings</span>
      </Entitled>,
    );
    await waitFor(() => expect(screen.queryByText("upgrade")).not.toBeNull());
  });
});
