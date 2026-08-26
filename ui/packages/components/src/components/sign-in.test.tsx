import { act, render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { SignIn } from "./sign-in";
import { routedFetch, withProvider } from "../test-support";

function at(pathname: string): void {
  window.history.pushState({}, "", pathname);
}

function navigate(pathname: string): void {
  act(() => {
    window.history.pushState({}, "", pathname);
    window.dispatchEvent(new PopStateEvent("popstate"));
  });
}

/**
 * SignIn picks its screen off the sub-path. These pin that mapping through the
 * component, so the shared useSubPath hook cannot be swapped underneath it
 * without the routing being re-checked.
 */
describe("SignIn routing", () => {
  const mount = () => {
    const { fetchFn } = routedFetch({});
    return render(withProvider(<SignIn />, { fetch: fetchFn, session: null }));
  };

  it("shows the forgot-password screen at /sign-in/forgot-password", () => {
    at("/sign-in/forgot-password");
    mount();
    expect(screen.getByText("Forgot password")).toBeTruthy();
  });

  it("shows the reset-password screen at /sign-in/reset-password", () => {
    at("/sign-in/reset-password?token=abc");
    mount();
    // The title and the submit button share this label.
    expect(screen.getAllByText("Reset password").length).toBeGreaterThan(0);
  });

  it("leaves the sub-screens when the user navigates back", () => {
    at("/sign-in/forgot-password");
    mount();
    expect(screen.getByText("Forgot password")).toBeTruthy();

    navigate("/sign-in");
    expect(screen.queryByText("Forgot password")).toBeNull();
  });
});
