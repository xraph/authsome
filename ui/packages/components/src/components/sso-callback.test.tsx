import { render, screen, waitFor } from "@testing-library/react";
import { StrictMode } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { SSOCallback } from "./sso-callback";
import { makeUser, routedFetch, withProvider } from "../test-support";

function at(search: string): void {
  window.history.pushState({}, "", `/sso/callback${search}`);
}

function exchangeRoutes() {
  const tokens: (string | null)[] = [];
  const routed = routedFetch({
    "POST /v1/sso/exchange": ({ token }) => {
      tokens.push(token);
      return {
        session_token: "sso_tok",
        refresh_token: "sso_refresh",
        expires_at: new Date(Date.now() + 3_600_000).toISOString(),
        user: makeUser(),
        provider: "okta",
        is_new_user: false,
      };
    },
  });
  return { ...routed, tokens };
}

describe("SSOCallback", () => {
  afterEach(() => {
    window.history.pushState({}, "", "/");
  });

  it("exchanges the code exactly once, then calls onSuccess", async () => {
    at("?code=one-time-code");
    const { fetchFn, calls, tokens } = exchangeRoutes();
    const onSuccess = vi.fn();

    // StrictMode double-invokes effects; the single-use code must still be
    // sent only once.
    render(
      <StrictMode>
        {withProvider(<SSOCallback onSuccess={onSuccess} />, {
          fetch: fetchFn,
          session: null,
        })}
      </StrictMode>,
    );

    expect(screen.getByText("Signing you in…")).toBeTruthy();
    await waitFor(() => expect(onSuccess).toHaveBeenCalledTimes(1));
    expect(calls.filter((c) => c === "POST /v1/sso/exchange")).toHaveLength(1);
    // Publishable-key authed: no bearer token on the exchange.
    expect(tokens).toEqual([null]);
  });

  it("shows the reported sso_error without calling the exchange", async () => {
    at("?sso_error=access_denied");
    const { fetchFn, calls } = exchangeRoutes();
    const onError = vi.fn();

    render(
      withProvider(<SSOCallback onError={onError} />, {
        fetch: fetchFn,
        session: null,
      }),
    );

    await screen.findByText("Single sign-on failed");
    expect(screen.getByRole("alert").textContent).toContain("access_denied");
    expect(calls).not.toContain("POST /v1/sso/exchange");
    expect(onError).toHaveBeenCalledTimes(1);
  });

  it("treats a missing code as a failure", async () => {
    at("");
    const { fetchFn } = exchangeRoutes();

    render(withProvider(<SSOCallback />, { fetch: fetchFn, session: null }));

    await screen.findByText("Single sign-on failed");
    expect(screen.getByRole("alert").textContent).toMatch(/authorization code/i);
  });

  it("shows an exchange failure", async () => {
    const { fetchFn } = routedFetch({
      "POST /v1/sso/exchange": () =>
        new Response(JSON.stringify({ error: "code expired" }), {
          status: 400,
          headers: { "Content-Type": "application/json" },
        }),
    });
    const onSuccess = vi.fn();

    render(
      withProvider(<SSOCallback code="stale" onSuccess={onSuccess} />, {
        fetch: fetchFn,
        session: null,
      }),
    );

    await screen.findByText("Single sign-on failed");
    expect(screen.getByRole("alert").textContent).toContain("code expired");
    expect(onSuccess).not.toHaveBeenCalled();
  });
});
