import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import type { ClientConfig } from "@authsome/ui-core";
import { afterEach, describe, expect, it, vi } from "vitest";

import { SignInForm, type SignInFormComponentProps } from "./sign-in-form";
import { routedFetch, withProvider } from "../test-support";

const ssoOn: ClientConfig = {
  sso: {
    enabled: true,
    connections: [
      { id: "okta", name: "Okta" },
      { id: "azure-ad", name: "Azure AD" },
    ],
  },
};

async function mount(props: SignInFormComponentProps, clientConfig: ClientConfig) {
  const routed = routedFetch({
    // A hash URL, because jsdom implements same-document navigation and
    // nothing else. That makes window.location.assign observable here.
    "POST /v1/sso/okta/login": () => ({
      login_url: "#idp-okta",
      state: "st_1",
    }),
  });
  await act(async () => render(
    withProvider(<SignInForm {...props} />, {
      fetch: routed.fetchFn,
      session: null,
      clientConfig,
    }),
  ));
  return routed;
}

/**
 * SSO connections from the client config render as one button each, separate
 * from the `resolveSSO` home-realm discovery covered in sign-in-form.sso.test.
 */
describe("SignInForm SSO connections", () => {
  afterEach(() => {
    window.history.pushState({}, "", "/");
  });

  it("renders a button per configured connection", async () => {
    await mount({}, ssoOn);
    expect(screen.getByRole("button", { name: "Continue with Okta" })).toBeTruthy();
    expect(
      screen.getByRole("button", { name: "Continue with Azure AD" }),
    ).toBeTruthy();
  });

  it("hides the buttons when sso.enabled is false", async () => {
    await mount({}, { sso: { ...ssoOn.sso!, enabled: false } });
    expect(screen.queryByRole("button", { name: /continue with okta/i })).toBeNull();
  });

  it("starts the login with the connection id and return URL, then redirects", async () => {
    const { urls } = await mount(
      { ssoReturnUrl: "https://app.example.test/sso/callback" },
      ssoOn,
    );

    fireEvent.click(screen.getByRole("button", { name: "Continue with Okta" }));

    await waitFor(() => expect(window.location.hash).toBe("#idp-okta"));
    expect(urls).toContain(
      "POST /v1/sso/okta/login?return_url=" +
        encodeURIComponent("https://app.example.test/sso/callback"),
    );
  });

  it("lets onSSOLogin replace the built-in handler", async () => {
    const onSSOLogin = vi.fn();
    const { calls } = await mount({ onSSOLogin }, ssoOn);

    fireEvent.click(
      screen.getByRole("button", { name: "Continue with Azure AD" }),
    );

    expect(onSSOLogin).toHaveBeenCalledWith("azure-ad");
    expect(calls.some((c) => c.includes("/v1/sso/"))).toBe(false);
  });

  it("shows SSO buttons in the password-disabled view", async () => {
    await mount({}, { ...ssoOn, password: { enabled: false } });
    expect(screen.getByRole("button", { name: "Continue with Okta" })).toBeTruthy();
    expect(screen.queryByText(/no sign-in methods/i)).toBeNull();
  });
});
