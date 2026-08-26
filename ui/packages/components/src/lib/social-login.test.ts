import { AuthClient } from "@authsome/ui-core";
import { afterEach, describe, expect, it, vi } from "vitest";

import { handleSocialLogin } from "./social-login";
import { BASE, routedFetch } from "../test-support";

const realOpen = window.open;

afterEach(() => {
  window.open = realOpen;
});

/** A popup that never closes, so the completion poll stays parked. */
function stubPopup(): void {
  window.open = vi.fn(() => ({ closed: false }) as unknown as Window);
}

describe("handleSocialLogin", () => {
  it("sends the return target as redirect_url", async () => {
    // The backend reads redirect_url (query or body) and validates
    // frontend_url against the origin allowlist — see
    // plugins/social/plugin.go. Passing the target in the wrong slot means
    // the user is not returned where they started after signing in.
    stubPopup();
    window.history.pushState({}, "", "/dashboard?next=1");

    const { fetchFn, urls } = routedFetch({
      "POST /v1/social/github": () => ({
        auth_url: "https://github.test/login/oauth",
      }),
    });
    const client = new AuthClient({ baseURL: BASE, fetch: fetchFn });

    await handleSocialLogin(client, "github", () => {});

    const call = urls.find((u) => u.includes("/v1/social/github"));
    expect(call).toBeTruthy();
    const params = new URLSearchParams(call!.split("?")[1] ?? "");
    expect(params.get("redirect_url")).toBe(window.location.href);
  });

  it("never sends a stringified object as frontend_url", async () => {
    stubPopup();

    const { fetchFn, urls } = routedFetch({
      "POST /v1/social/github": () => ({
        auth_url: "https://github.test/login/oauth",
      }),
    });
    const client = new AuthClient({ baseURL: BASE, fetch: fetchFn });

    await handleSocialLogin(client, "github", () => {});

    const call = urls.find((u) => u.includes("/v1/social/github")) ?? "";
    const params = new URLSearchParams(call.split("?")[1] ?? "");
    // The generated client runs String() over whatever it is handed, so an
    // object passed into a string slot arrives as "[object Object]" — and
    // frontend_url is checked against the origin allowlist.
    expect(params.get("frontend_url")).not.toBe("[object Object]");
  });

  it("reports a failed start through onError", async () => {
    stubPopup();
    const { fetchFn } = routedFetch({});
    const client = new AuthClient({ baseURL: BASE, fetch: fetchFn });

    let seen: unknown = null;
    await handleSocialLogin(
      client,
      "github",
      () => {},
      (err) => (seen = err),
    );

    expect(seen).toBeTruthy();
  });
});
