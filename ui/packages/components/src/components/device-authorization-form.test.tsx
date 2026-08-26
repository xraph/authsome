import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { DeviceAuthorizationForm } from "./device-authorization-form";
import { routedFetch, withProvider } from "../test-support";

function at(search: string): void {
  window.history.pushState({}, "", `/device${search}`);
}

function navigate(search: string): void {
  act(() => {
    window.history.pushState({}, "", `/device${search}`);
    window.dispatchEvent(new PopStateEvent("popstate"));
  });
}

function codeInput(container: HTMLElement): HTMLInputElement {
  const input = container.querySelector("input");
  if (!input) throw new Error("no code input rendered");
  return input as HTMLInputElement;
}

describe("DeviceAuthorizationForm", () => {
  it("takes the code from ?user_code, stripping the dashes", () => {
    at("?user_code=ABCD-EFGH");
    const { fetchFn } = routedFetch({});
    const { container } = render(
      withProvider(<DeviceAuthorizationForm autoSubmit={false} />, {
        fetch: fetchFn,
      }),
    );
    expect(codeInput(container).value).toBe("ABCDEFGH");
  });

  it("also accepts ?code", () => {
    at("?code=WXYZ1234");
    const { fetchFn } = routedFetch({});
    const { container } = render(
      withProvider(<DeviceAuthorizationForm autoSubmit={false} />, {
        fetch: fetchFn,
      }),
    );
    expect(codeInput(container).value).toBe("WXYZ1234");
  });

  it("follows popstate to a new user_code", () => {
    at("?user_code=AAAA1111");
    const { fetchFn } = routedFetch({});
    const { container } = render(
      withProvider(<DeviceAuthorizationForm autoSubmit={false} />, {
        fetch: fetchFn,
      }),
    );
    expect(codeInput(container).value).toBe("AAAA1111");

    navigate("?user_code=BBBB2222");
    expect(codeInput(container).value).toBe("BBBB2222");
  });

  it("adopts a new initialCode prop", () => {
    at("");
    const { fetchFn } = routedFetch({});
    const { container, rerender } = render(
      withProvider(
        <DeviceAuthorizationForm autoSubmit={false} initialCode="aaaa1111" />,
        { fetch: fetchFn },
      ),
    );
    expect(codeInput(container).value).toBe("AAAA1111");

    rerender(
      withProvider(
        <DeviceAuthorizationForm autoSubmit={false} initialCode="cccc3333" />,
        { fetch: fetchFn },
      ),
    );
    expect(codeInput(container).value).toBe("CCCC3333");
  });

  it("keeps what the user typed when the props have not changed", () => {
    // Family C regression guard. Deriving the code from the prop on every
    // render, rather than only when the prop changes, silently throws away
    // typing the moment anything else re-renders the form.
    at("?user_code=AAAA1111");
    const { fetchFn } = routedFetch({});
    const { container, rerender } = render(
      withProvider(<DeviceAuthorizationForm autoSubmit={false} />, {
        fetch: fetchFn,
      }),
    );
    expect(codeInput(container).value).toBe("AAAA1111");

    fireEvent.change(codeInput(container), { target: { value: "ZZZZ9999" } });
    expect(codeInput(container).value).toBe("ZZZZ9999");

    rerender(
      withProvider(<DeviceAuthorizationForm autoSubmit={false} />, {
        fetch: fetchFn,
      }),
    );
    expect(codeInput(container).value).toBe("ZZZZ9999");
  });

  it("auto-submits a complete code from the URL exactly once", async () => {
    at("?user_code=ABCD-EFGH");
    const bodies: string[] = [];
    const { fetchFn } = routedFetch({
      "POST /v1/oauth/device/complete": () => {
        bodies.push("called");
        return { status: "approved" };
      },
    });

    render(
      withProvider(<DeviceAuthorizationForm />, { fetch: fetchFn }),
    );

    await waitFor(() =>
      expect(screen.getByText("Device authorized successfully")).toBeTruthy(),
    );
    expect(bodies).toHaveLength(1);
  });
});
