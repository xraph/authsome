import { act, render, screen } from "@testing-library/react";
import type { ClientConfig } from "@authsome/ui-core";
import { describe, expect, it } from "vitest";

import { AuthCard } from "./auth-card";
import { SignInForm } from "./sign-in-form";
import { SignUpForm } from "./sign-up-form";
import { routedFetch, withProvider } from "../test-support";

const branded: ClientConfig = {
  branding: { app_name: "Acme", logo_url: "https://cdn.example.test/acme.png" },
};

async function mount(ui: React.ReactElement, clientConfig?: ClientConfig) {
  const { fetchFn } = routedFetch({});
  return await act(async () => render(withProvider(ui, { fetch: fetchFn, session: null, clientConfig })));
}

/**
 * `config.branding` is the backend telling the UI whose app this is. The card
 * and the sign-in/sign-up headings pick it up on their own, and explicit props
 * still win so a host app can always override it.
 */
describe("branding from client config", () => {
  it("falls back to the configured logo when no logo prop is passed", async () => {
    await mount(
      <AuthCard title="Hello">
        <p>body</p>
      </AuthCard>,
      branded,
    );

    const img = screen.getByRole("img") as HTMLImageElement;
    expect(img.src).toBe("https://cdn.example.test/acme.png");
    expect(img.alt).toBe("Acme");
  });

  it("renders no logo when logo={null}, even with branding configured", async () => {
    await mount(
      <AuthCard title="Hello" logo={null}>
        <p>body</p>
      </AuthCard>,
      branded,
    );

    expect(screen.queryByRole("img")).toBeNull();
  });

  it("renders outside an AuthProvider without a logo", () => {
    render(
      <AuthCard title="Standalone">
        <p>body</p>
      </AuthCard>,
    );

    expect(screen.getByText("Standalone")).toBeTruthy();
    expect(screen.queryByRole("img")).toBeNull();
  });

  it("titles the sign-in form with the app name", async () => {
    await mount(<SignInForm />, branded);
    expect(screen.getByText("Sign in to Acme")).toBeTruthy();
  });

  it("keeps an explicit sign-in title over the app name", async () => {
    await mount(<SignInForm title="Welcome back" />, branded);
    expect(screen.getByText("Welcome back")).toBeTruthy();
    expect(screen.queryByText("Sign in to Acme")).toBeNull();
  });

  it("uses the plain sign-in title without branding", async () => {
    await mount(<SignInForm />);
    expect(screen.getByText("Sign in")).toBeTruthy();
  });

  it("titles the sign-up form with the app name", async () => {
    await mount(<SignUpForm />, branded);
    expect(screen.getByText("Create your Acme account")).toBeTruthy();
  });
});
