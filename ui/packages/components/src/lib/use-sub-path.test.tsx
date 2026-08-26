import { act, render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { extractSubPath, useSubPath } from "./use-sub-path";

function go(pathname: string): void {
  window.history.pushState({}, "", pathname);
}

function pop(pathname: string): void {
  act(() => {
    window.history.pushState({}, "", pathname);
    window.dispatchEvent(new PopStateEvent("popstate"));
  });
}

function Probe({ basePath }: { basePath: string }) {
  const sub = useSubPath(basePath);
  return <span data-testid="sub">{sub ?? "(none)"}</span>;
}

function current(): string {
  return screen.getByTestId("sub").textContent ?? "";
}

describe("extractSubPath", () => {
  it("returns the segment following the base path", () => {
    expect(extractSubPath("/sign-in/forgot-password", "/sign-in")).toBe(
      "forgot-password",
    );
  });

  it("returns undefined when the path is exactly the base path", () => {
    expect(extractSubPath("/sign-in", "/sign-in")).toBeUndefined();
  });

  it("ignores trailing slashes on the base path", () => {
    expect(extractSubPath("/sign-in/verify-email", "/sign-in//")).toBe(
      "verify-email",
    );
  });

  it("returns undefined when the path is outside the base path", () => {
    expect(extractSubPath("/settings/profile", "/sign-in")).toBeUndefined();
  });
});

describe("useSubPath", () => {
  it("reports the sub-path of the current location on first render", () => {
    go("/sign-in/forgot-password");
    render(<Probe basePath="/sign-in" />);
    expect(current()).toBe("forgot-password");
  });

  it("follows popstate navigation", () => {
    go("/sign-in");
    render(<Probe basePath="/sign-in" />);
    expect(current()).toBe("(none)");

    pop("/sign-in/reset-password");
    expect(current()).toBe("reset-password");

    pop("/sign-in");
    expect(current()).toBe("(none)");
  });

  it("resyncs when the base path changes", () => {
    // The family B regression guard. The old effect re-derived the value on
    // every run, which looks redundant next to the useState initializer and
    // is the obvious thing to drop. Dropping it leaves mount behaviour intact
    // and breaks exactly this: the same URL read against a new base path.
    go("/sign-up/verify-email");
    const { rerender } = render(<Probe basePath="/sign-in" />);
    expect(current()).toBe("(none)");

    rerender(<Probe basePath="/sign-up" />);
    expect(current()).toBe("verify-email");
  });

  it("stops listening once unmounted", () => {
    go("/sign-in");
    const { unmount } = render(<Probe basePath="/sign-in" />);
    unmount();
    // Would warn about setting state on an unmounted component if the
    // subscription outlived the render.
    pop("/sign-in/forgot-password");
  });
});
