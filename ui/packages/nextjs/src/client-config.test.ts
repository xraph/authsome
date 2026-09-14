import { afterEach, describe, expect, it, vi } from "vitest";

import { getClientConfig } from "./server";

afterEach(() => vi.unstubAllGlobals());

// Same contract as the browser client: a path-prefixed base URL keeps its
// prefix. The server helper is what a Next app calls at render time to seed
// the provider, so a wrong URL here means a null config on every request.
describe("getClientConfig", () => {
  it("keeps the base URL's path prefix and appends the publishable key", async () => {
    const fetchFn = vi.fn((_input: string | URL | Request, _init?: RequestInit) => Promise.resolve(Response.json({ methods: [] })));
    vi.stubGlobal("fetch", fetchFn);

    const config = await getClientConfig({ baseURL: "https://gw.test/identity/authsome", publishableKey: "pk_x" });

    expect(config).toEqual({ methods: [] });
    expect(fetchFn.mock.calls[0]![0]).toBe("https://gw.test/identity/authsome/v1/client-config?key=pk_x");
  });
});
