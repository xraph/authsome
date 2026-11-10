import { describe, it, expect } from "vitest";
import { oauth2, DEFAULT_SCOPE_LABELS, type OAuth2Api } from "./index";
import { fakeContext } from "./test-context";

describe("oauth2 plugin", () => {
  it("is named after the oauth2provider server plugin", () => {
    expect(oauth2.pluginName).toBe("oauth2provider");
  });

  it("reads the consent request by cid", async () => {
    const { ctx, request } = fakeContext({});
    await (oauth2().init(ctx) as OAuth2Api).getConsentRequest("cid_1");
    expect(request).toHaveBeenCalledWith("GET", "/v1/oauth/consent", {
      query: { cid: "cid_1" },
    });
  });

  it("sends the csrf back with the decision", async () => {
    const { ctx, request } = fakeContext({ redirect_url: "https://app/cb" });
    await (oauth2().init(ctx) as OAuth2Api).decide({
      cid: "cid_1",
      csrf: "tok",
      decision: "allow",
    });
    expect(request).toHaveBeenCalledWith("POST", "/v1/oauth/consent", {
      body: { cid: "cid_1", csrf: "tok", decision: "allow" },
    });
  });

  it("escapes a client id with awkward characters in it", async () => {
    const { ctx, request } = fakeContext(undefined);
    await (oauth2().init(ctx) as OAuth2Api).revokeGrant("client/one?x=1");
    expect(request).toHaveBeenCalledWith(
      "DELETE",
      "/v1/me/oauth/grants/client%2Fone%3Fx%3D1",
    );
  });

  it("ships readable labels for the standard scopes", () => {
    expect(DEFAULT_SCOPE_LABELS.openid).toBeTruthy();
    expect(DEFAULT_SCOPE_LABELS.offline_access).toBeTruthy();
  });
});
