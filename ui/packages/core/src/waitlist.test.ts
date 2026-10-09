import { describe, expect, it, vi } from "vitest";
import { AuthClient } from "./client";

describe("waitlist client", () => {
  it.each(["pending", "approved", "rejected"])(
    "returns %s with app context and custom transport",
    async (status) => {
      const fetch = vi
        .fn<typeof globalThis.fetch>()
        .mockResolvedValue(
          new Response(JSON.stringify({ email: "ada@example.com", status })),
        );
      const client = new AuthClient({
        baseURL: "/api///",
        publishableKey: "pk_old",
        fetch,
      });
      client.setPublishableKey("pk_current");

      await expect(
        client.joinWaitlist({ email: " Ada@Example.com ", name: "Ada" }),
      ).resolves.toEqual({ email: "ada@example.com", status });
      expect(fetch).toHaveBeenCalledWith("/api/v1/waitlist/join", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Accept: "application/json",
          "X-Publishable-Key": "pk_current",
        },
        body: JSON.stringify({ email: "ada@example.com", name: "Ada" }),
      });
    },
  );

  it.each([
    {},
    { status: "approved" },
    { email: "ada@example.com", status: "unknown" },
  ])("rejects malformed responses %j", async (body) => {
    const client = new AuthClient({
      baseURL: "",
      fetch: vi.fn().mockResolvedValue(new Response(JSON.stringify(body))),
    });
    await expect(
      client.joinWaitlist({ email: "ada@example.com" }),
    ).rejects.toThrow("Unable to check waitlist status");
  });

  it("preserves errors and allows retries", async () => {
    const fetch = vi
      .fn<typeof globalThis.fetch>()
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({ error: "Try again later", type: "rate_limited" }),
          { status: 429 },
        ),
      )
      .mockResolvedValueOnce(new Response("unavailable", { status: 502 }))
      .mockRejectedValueOnce(new TypeError("offline"))
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({ email: "ada@example.com", status: "approved" }),
        ),
      );
    const client = new AuthClient({ baseURL: "https://api.test", fetch });
    await expect(
      client.joinWaitlist({ email: "ada@example.com" }),
    ).rejects.toMatchObject({
      code: 429,
      type: "rate_limited",
      message: "Try again later",
    });
    await expect(
      client.joinWaitlist({ email: "ada@example.com" }),
    ).rejects.toMatchObject({ code: 502 });
    await expect(
      client.joinWaitlist({ email: "ada@example.com" }),
    ).rejects.toThrow("offline");
    await expect(
      client.joinWaitlist({ email: "ada@example.com" }),
    ).resolves.toMatchObject({ status: "approved" });
  });
});
