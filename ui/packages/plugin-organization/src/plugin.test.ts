import { describe, it, expect, vi, afterEach } from "vitest";
import { organization, expiresIn, type OrganizationApi } from "./index";
import { fakeContext } from "./test-context";

afterEach(() => vi.useRealTimers());

describe("organization plugin", () => {
  it("is named after the server plugin", () => {
    expect(organization.pluginName).toBe("organization");
  });

  it("reads an invitation by token without a body", async () => {
    const { ctx, request } = fakeContext({});
    await (organization().init(ctx) as OrganizationApi).getInvitation("tok_1");
    expect(request).toHaveBeenCalledWith("GET", "/v1/orgs/invitations/tok_1");
  });

  it("escapes a token with url characters in it", async () => {
    const { ctx, request } = fakeContext({});
    await (organization().init(ctx) as OrganizationApi).getInvitation("a/b+c");
    expect(request).toHaveBeenCalledWith("GET", "/v1/orgs/invitations/a%2Fb%2Bc");
  });

  it("posts the token when accepting", async () => {
    const { ctx, request } = fakeContext({});
    await (organization().init(ctx) as OrganizationApi).acceptInvitation("tok_1");
    expect(request).toHaveBeenCalledWith("POST", "/v1/orgs/invitations/accept", {
      body: { token: "tok_1" },
    });
  });

  it("lists an org's invitations with paging", async () => {
    const { ctx, request } = fakeContext({ invitations: [] });
    await (organization().init(ctx) as OrganizationApi).listInvitations("org_1", {
      limit: 20,
    });
    expect(request).toHaveBeenCalledWith("GET", "/v1/orgs/org_1/invitations", {
      query: { limit: 20, cursor: undefined },
    });
  });
});

describe("expiresIn", () => {
  it("counts whole days ahead", () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-11-10T12:00:00Z"));
    expect(expiresIn("2026-11-16T12:00:00Z")).toBe("in 6 days");
  });

  it("says expired once the moment has passed", () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-11-10T12:00:00Z"));
    expect(expiresIn("2026-11-09T12:00:00Z")).toBe("expired");
  });

  it("uses the singular for one day", () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-11-10T12:00:00Z"));
    expect(expiresIn("2026-11-11T11:00:00Z")).toBe("in 1 day");
  });

  it("is empty for something that is not a date", () => {
    expect(expiresIn("not a date")).toBe("");
  });
});
