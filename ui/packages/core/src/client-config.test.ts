import { describe, expect, it, vi } from "vitest";

import { AuthClient } from "./client";

// Guards the base URL contract: an AuthSome API mounted under a path prefix
// (a gateway routes it at /identity/authsome) must keep that prefix on the
// client-config request. `new URL("/v1/...", base)` resolves against the
// origin and silently drops the path, so every page asked the gateway root
// and got a 404 plus an unhandled rejection.
describe("fetchClientConfig", () => {
  it("keeps the base URL's path prefix and appends the publishable key", async () => {
    const fetchFn = vi.fn((_input: string | URL | Request, _init?: RequestInit) => Promise.resolve(Response.json({ methods: [] })));
    const client = new AuthClient({
      baseURL: "https://gw.test/identity/authsome/",
      publishableKey: "pk_x",
      fetch: fetchFn,
    });

    await client.fetchClientConfig("pk_x");

    expect(fetchFn).toHaveBeenCalledTimes(1);
    expect(fetchFn.mock.calls[0]![0]).toBe("https://gw.test/identity/authsome/v1/client-config?key=pk_x");
    expect(fetchFn.mock.calls[0]![1]?.headers).toMatchObject({
      "X-Publishable-Key": "pk_x",
    });
  });
});
