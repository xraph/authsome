import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { SignInForm, type SSOResolution } from "./sign-in-form";
import { routedFetch, stubAuth, withAuth } from "../test-support";

/**
 * Home-realm discovery on the email step. `resolveSSO` lets the host app route
 * an email's domain to its IdP before a password is ever requested — the
 * Okta/Microsoft/Google "identifier-first" flow. These pin the branching so the
 * SSO step cannot regress into always asking for a password.
 */
describe("SignInForm SSO discovery", () => {
  const mount = (
    resolveSSO?: (email: string) => Promise<SSOResolution | null | undefined>,
  ) => {
    const { fetchFn } = routedFetch({});
    return render(
      withAuth(
        <SignInForm resolveSSO={resolveSSO} />,
        stubAuth({ fetch: fetchFn, session: null }),
      ),
    );
  };

  const submitEmail = (value: string) => {
    fireEvent.change(screen.getByLabelText("Email address"), {
      target: { value },
    });
    fireEvent.click(screen.getByRole("button", { name: "Continue" }));
  };

  it("routes an enforced domain straight to SSO, never showing a password field", async () => {
    const start = vi.fn();
    const resolveSSO = vi.fn(async () => ({ continue: start, enforced: true }));

    mount(resolveSSO);
    submitEmail("user@enforced.com");

    // The SSO step renders and password entry is not offered.
    await screen.findByRole("button", { name: /continue with sso/i });
    expect(resolveSSO).toHaveBeenCalledWith("user@enforced.com");
    expect(screen.queryByLabelText("Password")).toBeNull();
    expect(screen.queryByText(/password instead/i)).toBeNull();
    // Enforced auto-starts the IdP handoff.
    expect(start).toHaveBeenCalledTimes(1);
  });

  it("offers SSO alongside password when the domain is not enforced", async () => {
    const start = vi.fn();
    const resolveSSO = vi.fn(async () => ({
      continue: start,
      enforced: false,
      provider: "Okta",
    }));

    mount(resolveSSO);
    submitEmail("user@optional.com");

    // Provider name brands the SSO button; password remains reachable.
    const ssoButton = await screen.findByRole("button", {
      name: /continue with okta/i,
    });
    expect(start).not.toHaveBeenCalled(); // not auto-started when optional

    // The SSO button, when clicked, starts the IdP handoff (no-op stub keeps
    // the SSO step mounted so the password option is still reachable).
    fireEvent.click(ssoButton);
    expect(start).toHaveBeenCalledTimes(1);

    // Choosing "password instead" reveals the password field.
    fireEvent.click(screen.getByText(/password instead/i));
    expect(screen.getByLabelText("Password")).toBeTruthy();
  });

  it("falls through to the password step when the domain has no SSO", async () => {
    const resolveSSO = vi.fn(async () => null);

    mount(resolveSSO);
    submitEmail("user@nosso.com");

    await screen.findByLabelText("Password");
    expect(resolveSSO).toHaveBeenCalledWith("user@nosso.com");
  });

  it("fails open to password when the resolver throws", async () => {
    const resolveSSO = vi.fn(async () => {
      throw new Error("discovery unavailable");
    });

    mount(resolveSSO);
    submitEmail("user@flaky.com");

    // Discovery failure must never lock a user out of password login.
    await screen.findByLabelText("Password");
  });

  it("keeps the plain email→password flow when no resolver is supplied", async () => {
    mount(undefined);
    submitEmail("user@plain.com");

    await waitFor(() =>
      expect(screen.getByLabelText("Password")).toBeTruthy(),
    );
  });
});
