import { describe, it, expect, vi } from "vitest";
import { phone, type PhoneApi } from "./index";

/** A PluginContext that records what the plugin asked for. */
function fakeContext(reply: unknown) {
  const request = vi.fn().mockResolvedValue(reply);
  return {
    ctx: {
      request,
      getState: () => ({ status: "unauthenticated" }) as never,
      getClientConfig: () => null,
      isServerEnabled: () => true,
      getActiveOrgId: () => null,
      on: () => () => {},
    },
    request,
  };
}

describe("phone plugin", () => {
  it("is named after the server plugin", () => {
    expect(phone.pluginName).toBe("phone");
    expect(phone().name).toBe("phone");
  });

  it("posts the number to the start route", async () => {
    const { ctx, request } = fakeContext({ status: "sent", expires_in: 300 });
    const api: PhoneApi = phone().init(ctx);

    const res = await api.start({ phone: "+351912345678" });

    expect(request).toHaveBeenCalledWith("POST", "/v1/phone/start", {
      body: { phone: "+351912345678" },
    });
    expect(res.expires_in).toBe(300);
  });

  it("posts the code to the verify route and returns the session", async () => {
    const session = {
      session_token: "s",
      refresh_token: "r",
      expires_at: "2026-01-01T00:00:00Z",
      new_user: true,
    };
    const { ctx, request } = fakeContext(session);
    const api: PhoneApi = phone().init(ctx);

    const res = await api.verify({ phone: "+351912345678", code: "482917" });

    expect(request).toHaveBeenCalledWith("POST", "/v1/phone/verify", {
      body: { phone: "+351912345678", code: "482917" },
    });
    expect(res.new_user).toBe(true);
  });

  it("passes app_id through when a deployment needs one", async () => {
    const { ctx, request } = fakeContext({ status: "sent", expires_in: 60 });
    await phone().init(ctx).start({ phone: "+15551234567", app_id: "app_2" });

    expect(request).toHaveBeenCalledWith("POST", "/v1/phone/start", {
      body: { phone: "+15551234567", app_id: "app_2" },
    });
  });

  it("lets a failure from the transport reach the caller", async () => {
    const boom = new Error("429");
    const ctx = {
      request: vi.fn().mockRejectedValue(boom),
      getState: () => ({ status: "unauthenticated" }) as never,
      getClientConfig: () => null,
      isServerEnabled: () => true,
      getActiveOrgId: () => null,
      on: () => () => {},
    };

    await expect(phone().init(ctx).start({ phone: "+1" })).rejects.toBe(boom);
  });
});
