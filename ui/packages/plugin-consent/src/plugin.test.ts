import { describe, it, expect } from "vitest";
import { consent, type ConsentApi } from "./index";
import { fakeContext } from "./test-context";

describe("consent plugin", () => {
  it("is named after the server plugin", () => {
    expect(consent.pluginName).toBe("consent");
  });

  it("lists with the filters it was given", async () => {
    const { ctx, request } = fakeContext({ consents: [] });
    await (consent().init(ctx) as ConsentApi).list({ purpose: "marketing", limit: 10 });
    expect(request).toHaveBeenCalledWith("GET", "/v1/consent", {
      query: { purpose: "marketing", cursor: undefined, limit: 10 },
    });
  });

  it("lists with no filters at all", async () => {
    const { ctx, request } = fakeContext({ consents: [] });
    await (consent().init(ctx) as ConsentApi).list();
    expect(request).toHaveBeenCalledWith("GET", "/v1/consent", {
      query: { purpose: undefined, cursor: undefined, limit: undefined },
    });
  });

  it("records the version when granting", async () => {
    const { ctx, request } = fakeContext({});
    await (consent().init(ctx) as ConsentApi).grant({
      purpose: "marketing",
      version: "2026-01",
    });
    expect(request).toHaveBeenCalledWith("POST", "/v1/consent/grant", {
      body: { purpose: "marketing", version: "2026-01" },
    });
  });

  it("revokes by purpose", async () => {
    const { ctx, request } = fakeContext({ status: "ok" });
    await (consent().init(ctx) as ConsentApi).revoke({ purpose: "marketing" });
    expect(request).toHaveBeenCalledWith("POST", "/v1/consent/revoke", {
      body: { purpose: "marketing" },
    });
  });
});
