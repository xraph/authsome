import { describe, expect, it } from "vitest";

import {
  readSessionToken,
  SESSION_COOKIE,
  SESSION_STORAGE_COOKIE,
} from "./session-cookie";

/** Minimal stand-in for NextRequest.cookies / the next/headers cookie store. */
function jar(entries: Record<string, string>) {
  return {
    get(name: string) {
      const value = entries[name];
      return value === undefined ? undefined : { value };
    },
  };
}

describe("SESSION_STORAGE_COOKIE", () => {
  // createCookieStorage percent-encodes anything outside [A-Za-z0-9_-], so the
  // ":" in the storage key becomes %3A. If these drift apart the cookie written
  // by the client is invisible to the server again.
  it("matches the name createCookieStorage writes", () => {
    expect(SESSION_STORAGE_COOKIE).toBe("authsome%3Asession");
  });
});

describe("readSessionToken", () => {
  it("reads the backend httpOnly cookie", () => {
    expect(readSessionToken(jar({ [SESSION_COOKIE]: "tok_backend" }))).toBe("tok_backend");
  });

  // This is the case that was broken: the client persisted a session, and the
  // server saw nothing at all.
  it("reads a session persisted by createCookieStorage", () => {
    const stored = encodeURIComponent(
      JSON.stringify({
        session_token: "tok_stored",
        refresh_token: "refresh",
        expires_at: "2099-01-01T00:00:00.000Z",
      }),
    );
    expect(readSessionToken(jar({ [SESSION_STORAGE_COOKIE]: stored }))).toBe("tok_stored");
  });

  // The backend cookie is the one JavaScript cannot have tampered with.
  it("prefers the httpOnly cookie when both are present", () => {
    const stored = encodeURIComponent(JSON.stringify({ session_token: "tok_stored" }));
    expect(
      readSessionToken(
        jar({ [SESSION_COOKIE]: "tok_backend", [SESSION_STORAGE_COOKIE]: stored }),
      ),
    ).toBe("tok_backend");
  });

  it("honours a custom cookie name", () => {
    expect(readSessionToken(jar({ custom_session: "tok" }), "custom_session")).toBe("tok");
  });

  it("returns undefined with no cookies", () => {
    expect(readSessionToken(jar({}))).toBeUndefined();
  });

  // A malformed cookie must read as "no session" rather than yielding a value
  // that would be sent upstream as a bearer token.
  describe("refuses malformed stored sessions", () => {
    it.each([
      ["not json", "not-json-at-all"],
      ["json without a token", encodeURIComponent(JSON.stringify({ refresh_token: "r" }))],
      ["non-string token", encodeURIComponent(JSON.stringify({ session_token: 42 }))],
      ["empty token", encodeURIComponent(JSON.stringify({ session_token: "" }))],
      ["json array", encodeURIComponent(JSON.stringify(["session_token"]))],
      ["json null", encodeURIComponent(JSON.stringify(null))],
    ])("%s", (_label, value) => {
      expect(readSessionToken(jar({ [SESSION_STORAGE_COOKIE]: value }))).toBeUndefined();
    });
  });
});
