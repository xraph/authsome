import { describe, expect, it } from "vitest";

import { redirectAfterAuthTarget } from "./redirect-after-auth";

const origin = "https://app.example.com";

describe("redirectAfterAuthTarget", () => {
  it("keeps the path, query and fragment you asked for", () => {
    const search = "?redirect=" + encodeURIComponent("/dashboard?tab=2#billing");
    expect(redirectAfterAuthTarget(origin, search)).toBe(
      "https://app.example.com/dashboard?tab=2#billing",
    );
  });

  it("goes home when there is no redirect", () => {
    expect(redirectAfterAuthTarget(origin, "")).toBe("https://app.example.com/");
  });

  it.each([
    "https://evil.com/x",
    "//evil.com",
    "/\\evil.com",
    "javascript:alert(1)",
    "data:text/html,<script>alert(1)</script>",
  ])("goes home instead of to %s", (redirect) => {
    const search = "?redirect=" + encodeURIComponent(redirect);
    expect(redirectAfterAuthTarget(origin, search)).toBe("https://app.example.com/");
  });

  // A same-origin URL whose path starts with two slashes is accepted, and
  // as a bare path it would read as protocol-relative. Rebuilt on the
  // origin it can't leave the host.
  it("never lets a doubled slash in the path change the host", () => {
    const search = "?redirect=" + encodeURIComponent("https://app.example.com//evil.com/x");
    const target = new URL(redirectAfterAuthTarget(origin, search));
    expect(target.host).toBe("app.example.com");
    expect(target.pathname).toBe("//evil.com/x");
  });
});
