import { describe, expect, it, vi } from "vitest";
import { AuthClient, AuthClientError } from "./client";

function clientWith(body: unknown, status = 200) {
  const fetch = vi.fn<typeof globalThis.fetch>().mockResolvedValue(
    new Response(JSON.stringify(body), { status }),
  );
  const client = new AuthClient({ baseURL: "https://api.test/", publishableKey: "pk_test", fetch });
  return { client, fetch };
}

describe("client ceremony adapters", () => {
  it("sends the MFA ticket and rejects enrollment-only or empty-ticket calls locally", async () => {
    const response = { user: { id: "usr_1" }, session_token: "issued", refresh_token: "refresh", expires_at: "2026-10-10T00:00:00Z" };
    const { client, fetch } = clientWith(response);
    await expect(client.mfaChallenge({ enrollment_id: "enr_1", code: "123456" })).rejects.toThrow("ticket");
    await expect(client.mfaChallenge({ mfa_ticket: "", code: "123456" })).rejects.toThrow("ticket");
    expect(fetch).not.toHaveBeenCalled();
    await expect(client.mfaChallenge({ mfa_ticket: "ticket", code: "123456" })).resolves.toEqual(response);
    expect(fetch).toHaveBeenCalledWith("https://api.test/v1/mfa/challenge", expect.objectContaining({
      method: "POST", body: JSON.stringify({ mfa_ticket: "ticket", code: "123456" }),
    }));
  });

  it.each(["recovery", { code: "recovery" }])("retains the non-session recovery verification result for %j", async (body) => {
    const response = { challenge_passed: true, codes_remaining: 7 };
    const { client, fetch } = clientWith(response);
    await expect(client.verifyRecoveryCode(body)).resolves.toEqual(response);
    expect(fetch).toHaveBeenCalledWith("https://api.test/v1/mfa/recovery/verify", expect.objectContaining({
      method: "POST", body: JSON.stringify({ code: "recovery" }),
    }));
  });

  it.each(["approve", "deny"] as const)("uses generated form transport for device %s", async (action) => {
    const { client, fetch } = clientWith({ status: action });
    await expect(client.completeDeviceAuthorization("ABCD EFGH", action, "token")).resolves.toEqual({ status: action });
    expect(fetch).toHaveBeenCalledWith("https://api.test/v1/oauth/device/complete", {
      method: "POST",
      credentials: "include",
      headers: {
        "Content-Type": "application/x-www-form-urlencoded",
        Accept: "application/json",
        Authorization: "Bearer token",
        "X-Publishable-Key": "pk_test",
      },
      body: new URLSearchParams({ user_code: "ABCD EFGH", action }).toString(),
    });
  });

  it.each(["https://api.test/base///", "/api///", ""])("preserves base URL %s", async (baseURL) => {
    const fetch = vi.fn<typeof globalThis.fetch>().mockResolvedValue(new Response(JSON.stringify({ status: "approved" })));
    const client = new AuthClient({ baseURL, fetch });
    await client.completeDeviceAuthorization("ABCDEFGH", "approve");
    expect(fetch).toHaveBeenCalledWith(`${baseURL.replace(/\/+$/, "")}/v1/oauth/device/complete`, expect.objectContaining({
      method: "POST", credentials: "include",
    }));
  });

  it("keeps optional cookie authentication and current publishable key", async () => {
    const { client, fetch } = clientWith({ status: "approved" });
    client.setPublishableKey("pk_new");
    await client.completeDeviceAuthorization("ABCDEFGH", "approve");
    const init = fetch.mock.calls[0][1];
    expect(init?.credentials).toBe("include");
    const headers = new Headers(init?.headers);
    expect(headers.get("Authorization")).toBeNull();
    expect(headers.get("X-Publishable-Key")).toBe("pk_new");
  });

  it("preserves generated errors and leaves unrelated transport options alone", async () => {
    const { client, fetch } = clientWith({ error: "expired code", type: "expired_code" }, 400);
    await expect(client.completeDeviceAuthorization("ABCDEFGH", "approve")).rejects.toMatchObject({
      message: "expired code", code: 400, type: "expired_code", details: { error: "expired code" },
    });
    await expect(client.sendSMSCodeForMFA("token")).rejects.toBeInstanceOf(AuthClientError);
    expect(fetch.mock.calls[1][1]?.credentials).toBeUndefined();
  });

  it("propagates non-JSON and network failures", async () => {
    const { client, fetch } = clientWith({});
    fetch.mockResolvedValueOnce(new Response("upstream unavailable", { status: 502 }));
    await expect(client.completeDeviceAuthorization("ABCDEFGH", "deny")).rejects.toMatchObject({ code: 502 });
    fetch.mockRejectedValueOnce(new TypeError("offline"));
    await expect(client.completeDeviceAuthorization("ABCDEFGH", "deny")).rejects.toThrow("offline");
  });
});
