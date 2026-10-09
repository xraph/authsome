import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { AuthProvider } from "@authsome/ui-react";
import {
  routedFetch,
  withProvider,
  json,
  memoryStorage,
} from "../test-support";
import { SignUpForm } from "./sign-up-form";
import { WaitlistForm } from "./waitlist-form";

function continueWithEmail(email = "ada@example.com") {
  fireEvent.change(screen.getByLabelText("Email address"), {
    target: { value: email },
  });
  fireEvent.click(screen.getByRole("button", { name: "Continue" }));
}

describe("SignUpForm waitlist approval", () => {
  it("waits for auto-config before checking approval", async () => {
    let resolve!: (value: unknown) => void;
    const { fetchFn, calls } = routedFetch({
      "GET /v1/client-config": () =>
        new Promise((done) => {
          resolve = done;
        }),
      "POST /v1/waitlist/join": () => ({
        email: "ada@example.com",
        status: "pending",
      }),
    });
    render(
      <AuthProvider
        baseURL="https://api.example.test"
        publishableKey="pk_test"
        fetch={fetchFn}
        storage={memoryStorage()}
      >
        <SignUpForm />
      </AuthProvider>,
    );
    await waitFor(() => expect(calls).toEqual(["GET /v1/client-config"]));
    continueWithEmail();
    expect(screen.queryByLabelText("Password")).toBeNull();
    resolve({ waitlist: { enabled: true } });
    await screen.findByText("You're on the waitlist");
    expect(calls).toEqual(["GET /v1/client-config", "POST /v1/waitlist/join"]);
  });
  it.each(["pending", "rejected"])(
    "keeps %s entries out of signup details",
    async (status) => {
      const { fetchFn, calls } = routedFetch({
        "POST /v1/waitlist/join": () => ({ email: "ada@example.com", status }),
      });
      render(
        withProvider(<SignUpForm />, {
          fetch: fetchFn,
          session: null,
          clientConfig: { waitlist: { enabled: true } },
        }),
      );
      continueWithEmail();

      await screen.findByText(
        status === "pending"
          ? "You're on the waitlist"
          : "Signup isn't available",
      );
      expect(screen.queryByLabelText("Password")).toBeNull();
      expect(calls).toEqual(["POST /v1/waitlist/join"]);
      fireEvent.click(
        screen.getByRole("button", { name: "Use another email" }),
      );
      expect(screen.getByLabelText("Email address")).toBeTruthy();
    },
  );

  it("continues approved entries to signup with the checked email", async () => {
    const { fetchFn, calls } = routedFetch({
      "POST /v1/waitlist/join": () => ({
        email: "ada@example.com",
        status: "approved",
      }),
      "POST /v1/signup": () => json({ error: "approval revoked" }, 403),
    });
    render(
      withProvider(<SignUpForm />, {
        fetch: fetchFn,
        session: null,
        clientConfig: { waitlist: { enabled: true } },
      }),
    );
    continueWithEmail(" Ada@Example.com ");

    const password = await screen.findByLabelText("Password");
    expect(screen.getByText("ada@example.com")).toBeTruthy();
    fireEvent.change(password, { target: { value: "password123" } });
    fireEvent.click(screen.getByRole("button", { name: "Create account" }));
    await screen.findByText("approval revoked");
    expect(calls).toEqual(["POST /v1/waitlist/join", "POST /v1/signup"]);
  });

  it("checks again after a pending entry is approved", async () => {
    let status = "pending";
    const { fetchFn } = routedFetch({
      "POST /v1/waitlist/join": () => ({ email: "ada@example.com", status }),
    });
    render(
      withProvider(<SignUpForm />, {
        fetch: fetchFn,
        session: null,
        clientConfig: { waitlist: { enabled: true } },
      }),
    );
    continueWithEmail();
    await screen.findByText("You're on the waitlist");
    status = "approved";
    fireEvent.click(screen.getByRole("button", { name: "Check status" }));
    await screen.findByLabelText("Password");
  });

  it("stays on email after a failed check and supports retry", async () => {
    let fail = true;
    const { fetchFn } = routedFetch({
      "POST /v1/waitlist/join": () =>
        fail
          ? json({ error: "Temporarily unavailable" }, 503)
          : { email: "ada@example.com", status: "pending" },
    });
    render(
      withProvider(<SignUpForm />, {
        fetch: fetchFn,
        session: null,
        clientConfig: { waitlist: { enabled: true } },
      }),
    );
    continueWithEmail();
    await screen.findByText("Temporarily unavailable");
    expect(screen.queryByLabelText("Password")).toBeNull();
    fail = false;
    fireEvent.click(screen.getByRole("button", { name: "Continue" }));
    await screen.findByText("You're on the waitlist");
  });

  it("disables Continue while the approval check is pending", async () => {
    let resolve!: (value: unknown) => void;
    const { fetchFn, calls } = routedFetch({
      "POST /v1/waitlist/join": () =>
        new Promise((done) => {
          resolve = done;
        }),
    });
    render(
      withProvider(<SignUpForm />, {
        fetch: fetchFn,
        session: null,
        clientConfig: { waitlist: { enabled: true } },
      }),
    );
    continueWithEmail();
    expect(
      (screen.getByRole("button", { name: "Continue" }) as HTMLButtonElement)
        .disabled,
    ).toBe(true);
    expect(calls).toEqual(["POST /v1/waitlist/join"]);
    resolve({ email: "ada@example.com", status: "pending" });
    await screen.findByText("You're on the waitlist");
  });

  it.each([undefined, { enabled: false }])(
    "skips waitlist requests when disabled or absent",
    async (waitlist) => {
      const { fetchFn, calls } = routedFetch({});
      render(
        withProvider(<SignUpForm />, {
          fetch: fetchFn,
          session: null,
          clientConfig: { waitlist },
        }),
      );
      continueWithEmail();
      await screen.findByLabelText("Password");
      expect(calls).toEqual([]);
    },
  );

  it("uses the configured client for the standalone waitlist form", async () => {
    const { fetchFn, calls } = routedFetch({
      "POST /v1/waitlist/join": () => ({
        email: "ada@example.com",
        status: "pending",
      }),
    });
    render(
      withProvider(<WaitlistForm />, {
        fetch: fetchFn,
        session: null,
        clientConfig: { waitlist: { enabled: true } },
      }),
    );
    fireEvent.change(screen.getByLabelText("Email address"), {
      target: { value: "ada@example.com" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Join Waitlist" }));
    await waitFor(() =>
      expect(screen.getByText("You're on the list!")).toBeTruthy(),
    );
    expect(calls).toEqual(["POST /v1/waitlist/join"]);
  });
});
