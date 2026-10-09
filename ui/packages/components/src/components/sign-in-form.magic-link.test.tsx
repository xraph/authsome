import { fireEvent, render, screen } from "@testing-library/react";
import type { ClientConfig } from "@authsome/ui-core";
import { describe, expect, it } from "vitest";

import { SignInForm, type SignInFormComponentProps } from "./sign-in-form";
import { json, routedFetch, withProvider } from "../test-support";

const magicOn: ClientConfig = { magiclink: { enabled: true } };

/**
 * Wraps the routed fetch so the test can read the JSON body the form sent,
 * which routedFetch itself does not expose.
 */
function recordingFetch(sendStatus = 200) {
  const bodies: unknown[] = [];
  const routed = routedFetch({
    "POST /v1/magic-link/send": () =>
      sendStatus === 200
        ? { status: "sent" }
        : json({ error: "too many requests" }, sendStatus),
  });
  const fetchFn = (async (input: string | URL | Request, init?: RequestInit) => {
    if (String(input).endsWith("/v1/magic-link/send") && init?.body) {
      bodies.push(JSON.parse(String(init.body)));
    }
    return routed.fetchFn(input, init);
  }) as typeof globalThis.fetch;
  return { fetchFn, calls: routed.calls, bodies };
}

function mount(
  props: SignInFormComponentProps,
  clientConfig: ClientConfig,
  sendStatus?: number,
) {
  const rec = recordingFetch(sendStatus);
  render(
    withProvider(<SignInForm {...props} />, {
      fetch: rec.fetchFn,
      session: null,
      clientConfig,
    }),
  );
  return rec;
}

function continueWithEmail(value: string) {
  fireEvent.change(screen.getByLabelText("Email address"), {
    target: { value },
  });
  fireEvent.click(screen.getByRole("button", { name: "Continue" }));
}

describe("SignInForm magic link", () => {
  it("offers a sign-in link on the password step when config enables it", async () => {
    mount({}, magicOn);
    continueWithEmail("ada@test");

    await screen.findByLabelText("Password");
    expect(
      screen.getByRole("button", { name: "Email me a sign-in link instead" }),
    ).toBeTruthy();
  });

  it("hides the option when config leaves magic link off", async () => {
    mount({}, { magiclink: { enabled: false } });
    continueWithEmail("ada@test");

    await screen.findByLabelText("Password");
    expect(screen.queryByText(/sign-in link/i)).toBeNull();
  });

  it("lets showMagicLink={false} override an enabled config", async () => {
    mount({ showMagicLink: false }, magicOn);
    continueWithEmail("ada@test");

    await screen.findByLabelText("Password");
    expect(screen.queryByText(/sign-in link/i)).toBeNull();
  });

  it("sends the link to the entered email and shows the sent panel", async () => {
    const { bodies } = mount({}, magicOn);
    continueWithEmail("ada@test");

    fireEvent.click(
      await screen.findByRole("button", {
        name: "Email me a sign-in link instead",
      }),
    );

    await screen.findByText("Check your inbox");
    expect(screen.getByText("We sent a sign-in link to ada@test.")).toBeTruthy();
    expect(bodies).toEqual([{ email: "ada@test" }]);

    // The back link returns to the email step.
    fireEvent.click(screen.getByRole("button", { name: /use a different email/i }));
    expect(screen.getByRole("button", { name: "Continue" })).toBeTruthy();
  });

  it("shows a send failure inline", async () => {
    mount({}, magicOn, 429);
    continueWithEmail("ada@test");

    fireEvent.click(
      await screen.findByRole("button", {
        name: "Email me a sign-in link instead",
      }),
    );

    expect((await screen.findByRole("alert")).textContent).toMatch(
      /too many requests/i,
    );
    expect(screen.queryByText("Check your inbox")).toBeNull();
  });

  it("gives password-less apps an email field that sends a link", async () => {
    const { bodies } = mount({}, { password: { enabled: false }, ...magicOn });

    expect(screen.queryByText(/no sign-in methods/i)).toBeNull();
    fireEvent.change(screen.getByLabelText("Email address"), {
      target: { value: "grace@test" },
    });
    fireEvent.click(
      screen.getByRole("button", { name: "Email me a sign-in link" }),
    );

    await screen.findByText("Check your inbox");
    expect(bodies).toEqual([{ email: "grace@test" }]);
  });
});
