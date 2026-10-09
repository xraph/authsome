import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { useAuth } from "@authsome/ui-react";
import { describe, expect, it, vi } from "vitest";
import { MFAChallengeForm } from "./mfa-challenge-form";
import { json, makeSession, makeUser, routedFetch, withProvider } from "../test-support";

function SignIn() {
  const { signIn, state } = useAuth();
  return <>
    <button onClick={() => void signIn("ada@test", "password")}>Start sign in</button>
    <output data-testid="auth-status">{state.status}</output>
  </>;
}

function inputCode(container: HTMLElement, code: string) {
  const input = container.querySelector("input");
  if (!input) throw new Error("No code input");
  fireEvent.change(input, { target: { value: code } });
}

describe("MFAChallengeForm ceremony boundaries", () => {
  it("cannot select SMS login through methods or defaultMethod", async () => {
    const onSuccess = vi.fn();
    const { fetchFn, calls } = routedFetch({
      "POST /v1/signin": () => json({ error: "MFA required", type: "mfa_required", mfa_ticket: "ticket", available_methods: ["totp", "sms"] }, 403),
    });
    render(withProvider(<><SignIn /><MFAChallengeForm methods={["sms"]} defaultMethod="sms" onSuccess={onSuccess} /></>, { fetch: fetchFn, session: null }));
    await waitFor(() => expect(screen.getByTestId("auth-status").textContent).toBe("unauthenticated"));
    fireEvent.click(screen.getByText("Start sign in"));
    await waitFor(() => expect(screen.getByTestId("auth-status").textContent).toBe("mfa_required"));
    expect(screen.queryByText("Send code")).toBeNull();
    expect(screen.queryByText("Use SMS instead")).toBeNull();
    expect(screen.getByText("Recovery code")).toBeTruthy();
    expect(calls).toEqual(["POST /v1/signin"]);
    expect(onSuccess).not.toHaveBeenCalled();
  });

  it("shows a rejected recovery code and invokes login success only after retry succeeds", async () => {
    const onSuccess = vi.fn();
    let accepted = false;
    const { fetchFn, calls } = routedFetch({
      "POST /v1/signin": () => json({ error: "MFA required", type: "mfa_required", mfa_ticket: "ticket", available_methods: ["totp"] }, 403),
      "POST /v1/mfa/challenge": () => accepted ? { ...makeSession(), user: makeUser() } : json({ error: "Recovery code rejected" }, 401),
    });
    const { container } = render(withProvider(<><SignIn /><MFAChallengeForm defaultMethod="recovery" onSuccess={onSuccess} /></>, { fetch: fetchFn, session: null }));
    await waitFor(() => expect(screen.getByTestId("auth-status").textContent).toBe("unauthenticated"));
    fireEvent.click(screen.getByText("Start sign in"));
    await waitFor(() => expect(screen.getByTestId("auth-status").textContent).toBe("mfa_required"));
    inputCode(container, "RECOVERY");
    fireEvent.submit(container.querySelector("form")!);
    await screen.findByText("Recovery code rejected");
    expect(screen.getByTestId("auth-status").textContent).toBe("mfa_required");
    expect(onSuccess).not.toHaveBeenCalled();
    accepted = true;
    inputCode(container, "GOODCODE");
    fireEvent.submit(container.querySelector("form")!);
    await waitFor(() => expect(onSuccess).toHaveBeenCalledTimes(1));
    expect(screen.getByTestId("auth-status").textContent).toBe("authenticated");
    expect(calls.filter((call) => call.includes("recovery/verify"))).toEqual([]);
  });

  it.each([true, false])("reports SMS verified=%s without a login success callback", async (verified) => {
    const onSuccess = vi.fn();
    const { fetchFn } = routedFetch({
      "POST /v1/mfa/sms/send": () => ({ sent: true, phone_masked: "+1***1234", expires_in_seconds: 300 }),
      "POST /v1/mfa/sms/verify": ({ token }) => {
        expect(token).toBe("tok");
        return { verified, method: "sms" };
      },
    });
    const { container } = render(withProvider(<><SignIn /><MFAChallengeForm methods={["sms"]} defaultMethod="sms" onSuccess={onSuccess} /></>, { fetch: fetchFn }));
    fireEvent.click(await screen.findByText("Send code"));
    await screen.findByText("+1***1234");
    inputCode(container, "123456");
    await screen.findByText(verified ? "Phone verification complete." : "SMS verification failed");
    expect(screen.getByTestId("auth-status").textContent).toBe("authenticated");
    expect(onSuccess).not.toHaveBeenCalled();
  });
});
