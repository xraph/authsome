import { describe, it, expect } from "vitest";
import { scim, isTokenExpired, type ScimApi, type ScimToken } from "./index";
import { fakeContext } from "./test-context";

const token = (over: Partial<ScimToken> = {}): ScimToken => ({
  id: "t1",
  config_id: "c1",
  name: "Okta",
  created_at: "2026-01-01T00:00:00Z",
  ...over,
});

describe("scim plugin", () => {
  it("is named after the server plugin", () => {
    expect(scim.pluginName).toBe("scim");
  });

  it("lists an org's configurations", async () => {
    const { ctx, request } = fakeContext({ configs: [] });
    await (scim().init(ctx) as ScimApi).listConfigs("org_1");
    expect(request).toHaveBeenCalledWith("GET", "/v1/orgs/org_1/scim/configs");
  });

  it("escapes ids in paths", async () => {
    const { ctx, request } = fakeContext({});
    await (scim().init(ctx) as ScimApi).getConfig("c/1");
    expect(request).toHaveBeenCalledWith("GET", "/v1/scim/configs/c%2F1");
  });

  it("sends a partial body when updating", async () => {
    const { ctx, request } = fakeContext({});
    await (scim().init(ctx) as ScimApi).updateConfig("c1", { enabled: false });
    expect(request).toHaveBeenCalledWith("PATCH", "/v1/scim/configs/c1", {
      body: { enabled: false },
    });
  });

  it("rotates with a grace window", async () => {
    const { ctx, request } = fakeContext({});
    await (scim().init(ctx) as ScimApi).rotateToken("t1", { grace_seconds: 60 });
    expect(request).toHaveBeenCalledWith("POST", "/v1/scim/tokens/t1/rotate", {
      body: { grace_seconds: 60 },
    });
  });

  it("sends an empty body when rotating with no options", async () => {
    const { ctx, request } = fakeContext({});
    await (scim().init(ctx) as ScimApi).rotateToken("t1");
    expect(request).toHaveBeenCalledWith("POST", "/v1/scim/tokens/t1/rotate", {
      body: {},
    });
  });
});

describe("isTokenExpired", () => {
  it("is false for a token with no expiry", () => {
    expect(isTokenExpired(token())).toBe(false);
  });

  it("is true once the expiry has passed", () => {
    expect(
      isTokenExpired(token({ expires_at: "2026-01-02T00:00:00Z" }),
        Date.parse("2026-02-01T00:00:00Z")),
    ).toBe(true);
  });

  it("is false before the expiry", () => {
    expect(
      isTokenExpired(token({ expires_at: "2026-03-01T00:00:00Z" }),
        Date.parse("2026-02-01T00:00:00Z")),
    ).toBe(false);
  });

  it("treats an unparseable expiry as not expired", () => {
    expect(isTokenExpired(token({ expires_at: "soon" }))).toBe(false);
  });
});
