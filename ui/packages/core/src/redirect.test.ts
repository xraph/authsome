import { describe, expect, it } from "vitest";

import { isSafeRedirect, safeRedirectTarget } from "./redirect";

const ORIGIN = "https://app.example.com";
const opts = { currentOrigin: ORIGIN };

describe("isSafeRedirect", () => {
  describe("accepts same-origin destinations", () => {
    it.each([
      "/dashboard",
      "/",
      "/settings/profile",
      "/a/b?x=1#frag",
      "https://app.example.com/ok",
      // Percent-encoded slashes stay a path segment, so this never leaves the origin.
      "/%2F%2Fevil.com",
    ])("%j", (candidate) => {
      expect(isSafeRedirect(candidate, opts)).toBe(true);
    });
  });

  describe("refuses cross-origin destinations", () => {
    it.each(["https://evil.com", "http://evil.com"])("%j", (candidate) => {
      expect(isSafeRedirect(candidate, opts)).toBe(false);
    });

    // A hostname that merely starts with the trusted origin is a different host.
    it("refuses a suffix-confusion host", () => {
      expect(isSafeRedirect("https://app.example.com.evil.com/x", opts)).toBe(false);
    });
  });

  // These are the cases a `candidate.startsWith("/")` guard lets through. Each
  // one resolves to https://evil.com in a browser. Do not replace the
  // URL-resolution check with string inspection.
  describe("refuses host swaps that defeat prefix matching", () => {
    it.each([
      "//evil.com",
      "///evil.com",
      "/\\evil.com",
      "\\\\evil.com",
      "/\\/evil.com",
      "https:/\\evil.com",
    ])("%j", (candidate) => {
      expect(isSafeRedirect(candidate, opts)).toBe(false);
    });
  });

  // URL parsing strips the control characters used to disguise these, so
  // "java\nscript:" normalizes to "javascript:" before the protocol check.
  describe("refuses non-http(s) schemes", () => {
    it.each([
      "javascript:alert(1)",
      "JaVaScRiPt:alert(1)",
      "java\nscript:alert(1)",
      " javascript:alert(1)",
      "\tjavascript:alert(1)",
      "data:text/html,<script>alert(1)</script>",
      "blob:https://app.example.com/x",
    ])("%j", (candidate) => {
      expect(isSafeRedirect(candidate, opts)).toBe(false);
    });
  });

  describe("refuses leading-whitespace absolute URLs", () => {
    it.each(["\thttps://evil.com", " https://evil.com"])("%j", (candidate) => {
      expect(isSafeRedirect(candidate, opts)).toBe(false);
    });
  });

  // Same host, different origin: a downgraded scheme or a non-default port is
  // a distinct security context and must not inherit trust.
  describe("refuses same-host origin changes", () => {
    it.each(["http://app.example.com/x", "https://app.example.com:8443/x"])(
      "%j",
      (candidate) => {
        expect(isSafeRedirect(candidate, opts)).toBe(false);
      },
    );
  });

  describe("refuses empty and non-string input", () => {
    it.each([["", "empty"], ["   ", "whitespace"]])("%s (%s)", (candidate) => {
      expect(isSafeRedirect(candidate, opts)).toBe(false);
    });

    it("refuses null", () => {
      expect(isSafeRedirect(null, opts)).toBe(false);
    });

    it("refuses undefined", () => {
      expect(isSafeRedirect(undefined, opts)).toBe(false);
    });
  });

  describe("allowedOrigins", () => {
    const allow = {
      currentOrigin: ORIGIN,
      allowedOrigins: ["https://portal.example.com"],
    };

    it("accepts an exactly matching origin", () => {
      expect(isSafeRedirect("https://portal.example.com/x", allow)).toBe(true);
    });

    // Entries are normalized to an origin so config formatting can't cause a
    // silent miss that would look like a broken redirect rather than a typo.
    describe("tolerates entry formatting", () => {
      it.each([
        "https://portal.example.com",
        "https://portal.example.com/",
        "https://portal.example.com/some/path",
        "https://PORTAL.Example.COM",
        "https://portal.example.com:443",
      ])("%j", (entry) => {
        expect(
          isSafeRedirect("https://portal.example.com/x", {
            currentOrigin: ORIGIN,
            allowedOrigins: [entry],
          }),
        ).toBe(true);
      });
    });

    // Matching is exact by design — see SafeRedirectOptions.allowedOrigins.
    describe("does not widen to related hosts", () => {
      it.each([
        ["sibling subdomain", "https://other.example.com/x"],
        ["nested subdomain", "https://a.portal.example.com/x"],
        ["suffix confusion", "https://portal.example.com.evil.com/x"],
        ["prefix confusion", "https://evilportal.example.com/x"],
        ["scheme downgrade", "http://portal.example.com/x"],
        ["non-default port", "https://portal.example.com:8443/x"],
      ])("refuses %s", (_label, candidate) => {
        expect(isSafeRedirect(candidate, allow)).toBe(false);
      });
    });

    it("treats a wildcard entry as inert rather than permissive", () => {
      const wild = {
        currentOrigin: ORIGIN,
        allowedOrigins: ["*.example.com", "https://*.example.com"],
      };
      expect(isSafeRedirect("https://anything.example.com/x", wild)).toBe(false);
      // The unusable entry must not disturb same-origin redirects.
      expect(isSafeRedirect("/dashboard", wild)).toBe(true);
    });

    // An unparseable entry must match nothing rather than everything.
    describe("ignores malformed entries", () => {
      it.each([
        "",
        "   ",
        "portal.example.com",
        "not a url",
        "javascript:alert(1)",
        "data:text/html,x",
      ])("%j", (entry) => {
        expect(
          isSafeRedirect("https://evil.com", {
            currentOrigin: ORIGIN,
            allowedOrigins: [entry],
          }),
        ).toBe(false);
      });
    });

    // URL.origin stringifies opaque origins to "null", so a javascript: entry
    // and a javascript: candidate would compare equal without the guard.
    it("does not let an opaque-origin entry admit an opaque-origin candidate", () => {
      expect(
        isSafeRedirect("javascript:alert(1)", {
          currentOrigin: ORIGIN,
          allowedOrigins: ["javascript:alert(1)"],
        }),
      ).toBe(false);
    });
  });
});

describe("safeRedirectTarget", () => {
  it("returns the candidate when it is safe", () => {
    expect(safeRedirectTarget("/dashboard", "/", opts)).toBe("/dashboard");
  });

  it("falls back when the candidate is unsafe", () => {
    expect(safeRedirectTarget("https://evil.com", "/", opts)).toBe("/");
  });

  it("falls back when the candidate is absent", () => {
    expect(safeRedirectTarget(null, "/home", opts)).toBe("/home");
  });
});
