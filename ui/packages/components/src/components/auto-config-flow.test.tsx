import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeAll, describe, expect, it, vi } from "vitest";
import { AuthProvider } from "@authsome/ui-react";
import { SignInForm } from "./sign-in-form";
import { SignUpForm } from "./sign-up-form";
import { EmailVerificationForm } from "./email-verification-form";
import {
  json,
  makeSession,
  makeUser,
  memoryStorage,
  routedFetch,
  withProvider,
} from "../test-support";

vi.mock("./turnstile-widget", () => ({
  TurnstileWidget: ({ onToken }: { onToken: (token: string) => void }) => (
    <button type="button" onClick={() => onToken("challenge-token")}>
      Complete captcha
    </button>
  ),
}));
beforeAll(() =>
  vi.stubGlobal(
    "ResizeObserver",
    class {
      observe() {}
      unobserve() {}
      disconnect() {}
    },
  ),
);
function email() {
  fireEvent.change(screen.getByLabelText("Email address"), {
    target: { value: "Ada@Example.com" },
  });
  fireEvent.click(screen.getByRole("button", { name: "Continue" }));
}
async function password(label = "Continue") {
  fireEvent.change(await screen.findByLabelText("Password"), {
    target: { value: "password123" },
  });
  fireEvent.click(screen.getByRole("button", { name: label }));
}

describe("AutoConfig Continue", () => {
  it("waits for delayed sign-in options and never opens a disabled password step", async () => {
    let resolve!: (value: unknown) => void;
    const { fetchFn, calls } = routedFetch({
      "GET /v1/client-config": () =>
        new Promise((done) => {
          resolve = done;
        }),
    });
    render(
      <AuthProvider
        baseURL="https://api.example.test"
        publishableKey="pk_test"
        fetch={fetchFn}
        storage={memoryStorage()}
      >
        <SignInForm />
      </AuthProvider>,
    );
    await waitFor(() => expect(calls).toEqual(["GET /v1/client-config"]));
    email();
    expect(screen.queryByLabelText("Password")).toBeNull();
    resolve({ password: { enabled: false } });
    await screen.findByText(/No sign-in methods/);
    expect(screen.queryByLabelText("Password")).toBeNull();
  });
  it("blocks closed signup and hides the signup link on sign-in", async () => {
    const { fetchFn, calls } = routedFetch({});
    const props = {
      fetch: fetchFn,
      session: null,
      clientConfig: { signup_enabled: false },
    };
    const view = render(withProvider(<SignUpForm />, props));
    expect(screen.getByText("Signup isn't available")).toBeTruthy();
    expect(screen.queryByLabelText("Email address")).toBeNull();
    view.unmount();
    render(withProvider(<SignInForm signUpUrl="/sign-up" />, props));
    expect(screen.queryByRole("link", { name: "Sign up" })).toBeNull();
    expect(calls).toEqual([]);
  });
  it("shows a failed enforced SSO handoff and permits retry", async () => {
    const start = vi
      .fn()
      .mockRejectedValueOnce(new Error("IdP unavailable"))
      .mockResolvedValue(undefined);
    const { fetchFn } = routedFetch({});
    render(
      withProvider(
        <SignInForm
          resolveSSO={async () => ({ enforced: true, continue: start })}
        />,
        { fetch: fetchFn, session: null },
      ),
    );
    email();
    await screen.findByText("IdP unavailable");
    fireEvent.click(screen.getByRole("button", { name: "Continue with SSO" }));
    await waitFor(() => expect(start).toHaveBeenCalledTimes(2));
  });
  it("hands MFA to the challenge without firing login success", async () => {
    const success = vi.fn();
    const { fetchFn } = routedFetch({
      "POST /v1/signin": () =>
        json(
          {
            error: "MFA required",
            type: "mfa_required",
            mfa_ticket: "ticket",
            available_methods: ["totp"],
          },
          403,
        ),
    });
    render(
      withProvider(<SignInForm onSuccess={success} />, {
        fetch: fetchFn,
        session: null,
      }),
    );
    email();
    await password();
    await screen.findByText(/authenticator/i);
    expect(success).not.toHaveBeenCalled();
  });
  it("uses the code form for unverified sign-in and returns without login success", async () => {
    const success = vi.fn();
    const { fetchFn, calls } = routedFetch({
      "POST /v1/signin": () =>
        json({ error: "Verify your email", type: "email_not_verified" }, 403),
      "POST /v1/verify-email": () => ({ status: "verified" }),
    });
    render(
      withProvider(<SignInForm onSuccess={success} />, {
        fetch: fetchFn,
        session: null,
      }),
    );
    email();
    await password();
    await screen.findByText("Verify your email");
    fireEvent.change(document.querySelector("input")!, {
      target: { value: "123456" },
    });
    await screen.findByLabelText("Email address");
    expect(calls).toEqual(["POST /v1/signin", "POST /v1/verify-email"]);
    expect(success).not.toHaveBeenCalled();
  });

  it.each([true, false])(
    "uses the correct signup completion callback when verification required is %s",
    async (required) => {
      const success = vi.fn();
      const verify = vi.fn();
      const { fetchFn } = routedFetch({
        "POST /v1/signup": () => ({ ...makeSession(), user: makeUser() }),
      });
      render(
        withProvider(
          <SignUpForm onSuccess={success} onVerificationRequired={verify} />,
          {
            fetch: fetchFn,
            session: null,
            clientConfig: { email_verification: { enabled: true, required } },
          },
        ),
      );
      email();
      await password("Create account");
      await waitFor(() =>
        expect(required ? verify : success).toHaveBeenCalledTimes(1),
      );
      expect(required ? success : verify).not.toHaveBeenCalled();
      if (required) expect(verify).toHaveBeenCalledWith("ada@example.com");
    },
  );
  it("requires configured consent before creating an account", async () => {
    const { fetchFn, calls } = routedFetch({
      "POST /v1/signup": () => ({ ...makeSession(), user: makeUser() }),
    });
    render(
      withProvider(<SignUpForm />, {
        fetch: fetchFn,
        session: null,
        clientConfig: {
          signup_fields: [
            {
              key: "consent",
              type: "checkbox",
              label: "Accept terms",
              order: 1,
              validation: { required: true },
            },
          ],
        },
      }),
    );
    email();
    await password("Create account");
    await screen.findByText("Accept terms is required.");
    expect(calls).toEqual([]);
    fireEvent.click(screen.getByRole("checkbox"));
    fireEvent.click(screen.getByRole("button", { name: "Create account" }));
    await screen.findByText("Verify your email");
    expect(calls).toEqual(["POST /v1/signup"]);
  });
});

describe("captcha retries and verification", () => {
  it("blocks a required captcha with missing configuration, including form submission", async () => {
    const { fetchFn, calls } = routedFetch({});
    render(
      withProvider(<SignInForm />, {
        fetch: fetchFn,
        session: null,
        clientConfig: { captcha: { required: true, provider: "turnstile" } },
      }),
    );
    email();
    const input = await screen.findByLabelText("Password");
    fireEvent.change(input, { target: { value: "pw" } });
    expect(
      screen.getByText("Verification is unavailable. Please try again later."),
    ).toBeTruthy();
    fireEvent.submit(input.closest("form")!);
    expect(calls).toEqual([]);
    expect(screen.getByText("Please complete the captcha.")).toBeTruthy();
  });
  it("requires a fresh captcha after a rejected sign-in", async () => {
    const { fetchFn, calls } = routedFetch({
      "POST /v1/signin": () => json({ error: "Wrong password" }, 401),
    });
    render(
      withProvider(<SignInForm />, {
        fetch: fetchFn,
        session: null,
        clientConfig: {
          captcha: { required: true, provider: "turnstile", site_key: "site" },
        },
      }),
    );
    email();
    fireEvent.change(await screen.findByLabelText("Password"), {
      target: { value: "pw" },
    });
    fireEvent.click(screen.getByText("Complete captcha"));
    fireEvent.click(screen.getByRole("button", { name: "Continue" }));
    await screen.findByText("Wrong password");
    fireEvent.submit(screen.getByLabelText("Password").closest("form")!);
    expect(calls).toEqual(["POST /v1/signin"]);
  });
  it("resends a code by default and allows retry after transport failure", async () => {
    let count = 0;
    const { fetchFn, calls } = routedFetch({
      "POST /v1/verify-email/resend": () =>
        ++count === 1 ? json({ error: "offline" }, 503) : {},
    });
    render(
      withProvider(<EmailVerificationForm email="ada@example.com" />, {
        fetch: fetchFn,
        session: null,
      }),
    );
    fireEvent.click(screen.getByRole("button", { name: "Resend" }));
    await screen.findByText("offline");
    fireEvent.click(screen.getByRole("button", { name: "Resend" }));
    await screen.findByText("Resend in 60s");
    expect(calls).toEqual([
      "POST /v1/verify-email/resend",
      "POST /v1/verify-email/resend",
    ]);
  });
  it("sends the email with the OTP code for unauthenticated verification", async () => {
    let body: unknown;
    const fetchFn = vi.fn(async (_url: unknown, init?: RequestInit) => {
      body = JSON.parse(init?.body as string);
      return json({ status: "verified" });
    }) as typeof fetch;
    render(
      withProvider(<EmailVerificationForm email="ada@example.com" />, {
        fetch: fetchFn,
        session: null,
      }),
    );
    fireEvent.change(document.querySelector("input")!, {
      target: { value: "123456" },
    });
    await screen.findByText("Email verified");
    expect(body).toEqual({ email: "ada@example.com", code: "123456" });
  });
});
