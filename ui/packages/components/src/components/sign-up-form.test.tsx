import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import type { ClientConfig } from "@authsome/ui-core";
import { describe, expect, it } from "vitest";

import { SignUpForm } from "./sign-up-form";
import { routedFetch, withProvider } from "../test-support";

const withDefaults = {
  signup_fields: [
    {
      key: "company",
      label: "Company",
      type: "text",
      order: 1,
      default: "Acme Inc",
    },
    { key: "role", label: "Role", type: "text", order: 2 },
  ],
} as unknown as ClientConfig;

function field(key: string): HTMLInputElement {
  return document.getElementById(`signup-field-${key}`) as HTMLInputElement;
}

/** The dynamic fields live on the second step, behind the email form. */
function continuePastEmail(): void {
  fireEvent.change(document.getElementById("signup-email")!, {
    target: { value: "ada@test" },
  });
  fireEvent.click(screen.getByRole("button", { name: "Continue" }));
}

describe("SignUpForm dynamic field defaults", () => {
  const mount = (clientConfig: ClientConfig) => {
    const { fetchFn } = routedFetch({});
    return render(
      withProvider(<SignUpForm />, {
        fetch: fetchFn,
        session: null,
        clientConfig,
      }),
    );
  };

  it("pre-fills a configured field default", async () => {
    mount(withDefaults);
    continuePastEmail();

    await waitFor(() => expect(field("company")).toBeTruthy());
    expect(field("company").value).toBe("Acme Inc");
    expect(field("role").value).toBe("");
  });

  it("keeps what the user typed over a default across re-renders", async () => {
    // Family C regression guard. Seeding defaults on every render, rather than
    // only when the field config changes, throws away the user's edit as soon
    // as anything else re-renders the form.
    const { rerender } = mount(withDefaults);
    continuePastEmail();

    await waitFor(() => expect(field("company")).toBeTruthy());
    fireEvent.change(field("company"), { target: { value: "Other Co" } });
    expect(field("company").value).toBe("Other Co");

    const { fetchFn } = routedFetch({});
    rerender(
      withProvider(<SignUpForm />, {
        fetch: fetchFn,
        session: null,
        clientConfig: withDefaults,
      }),
    );

    expect(field("company").value).toBe("Other Co");
  });
});
