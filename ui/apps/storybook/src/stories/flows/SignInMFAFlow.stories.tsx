import React, { useState } from "react";
import type { Meta, StoryObj } from "@storybook/react";
import {
  SignInForm,
  MFAChallengeFormStyled,
  UserAvatar,
} from "@authsome/ui-components";
import { MockAuthProvider, MOCK_USER } from "../../mocks/auth-provider";
import {
  CONFIG_ALL_ENABLED,
  CONFIG_MFA_TOTP_ONLY,
  CONFIG_MFA_SMS_ONLY,
  CONFIG_MFA_ENABLED,
} from "../../mocks/client-config-presets";
import type { ClientConfig } from "@authsome/ui-core";
import { Check } from "lucide-react";

/**
 * Interactive multi-step flow: Sign In → MFA Challenge → Authenticated.
 *
 * Demonstrates the full sign-in experience when MFA is required.
 * The MFA challenge form auto-detects available methods (TOTP, SMS)
 * from the client config and shows a method switcher when multiple
 * methods are available. Recovery codes are always accessible.
 */
const meta: Meta = {
  title: "Flows/Sign In → MFA",
  tags: ["autodocs"],
  parameters: { layout: "centered" },
};
export default meta;
type Story = StoryObj;

type FlowStep = "sign-in" | "mfa" | "success";

function SignInMFAFlowRenderer({
  mfaError,
  config = CONFIG_ALL_ENABLED,
}: {
  mfaError?: boolean;
  config?: ClientConfig;
}) {
  const [step, setStep] = useState<FlowStep>("sign-in");

  return (
    <MockAuthProvider
      clientConfig={config}
      signInBehavior="mfa_required"
      simulateError={step === "mfa" && mfaError}
      errorMessage="Invalid MFA code. Please try again."
      delay={800}
    >
      <div className="w-[380px]">
        {step === "sign-in" && (
          <SignInForm
            signUpUrl="/sign-up"
            forgotPasswordUrl="/forgot-password"
            onSuccess={() => {
              // signInBehavior="mfa_required" sets state to mfa_required
              // on the provider level, so we just transition step here
              setStep("mfa");
            }}
          />
        )}

        {step === "mfa" && (
          <MFAChallengeFormStyled
            enrollmentId="enroll_mock_totp"
            onSuccess={() => setStep("success")}
          />
        )}

        {step === "success" && (
          <div className="rounded-lg border bg-card p-6 text-center shadow-sm">
            <div className="mx-auto mb-3 flex h-12 w-12 items-center justify-center rounded-full bg-green-100 dark:bg-green-900/30">
              <Check className="h-6 w-6 text-green-600 dark:text-green-400" />
            </div>
            <div className="mb-1 flex items-center justify-center gap-2">
              <UserAvatar user={MOCK_USER} size="sm" />
              <span className="text-sm font-medium">{MOCK_USER.name}</span>
            </div>
            <p className="text-[13px] text-muted-foreground">
              Welcome back! You&apos;re now signed in.
            </p>
            <p className="mt-1 text-xs text-muted-foreground/70">
              {MOCK_USER.email}
            </p>
          </div>
        )}
      </div>
    </MockAuthProvider>
  );
}

/** Happy path: Sign In → MFA (TOTP + SMS methods) → Success */
export const HappyPath: Story = {
  render: () => <SignInMFAFlowRenderer />,
};

/** With MFA error: entering wrong code shows error, then retry succeeds. */
export const WithMFAError: Story = {
  render: () => <SignInMFAFlowRenderer mfaError />,
};

/** TOTP only: no method switcher — just authenticator app code. */
export const TOTPOnly: Story = {
  render: () => <SignInMFAFlowRenderer config={CONFIG_MFA_TOTP_ONLY} />,
};

/** SMS only: no TOTP, sends SMS code directly. */
export const SMSOnly: Story = {
  render: () => <SignInMFAFlowRenderer config={CONFIG_MFA_SMS_ONLY} />,
};

/** TOTP + SMS: both methods available with method switcher. */
export const TOTPAndSMS: Story = {
  render: () => <SignInMFAFlowRenderer config={CONFIG_MFA_ENABLED} />,
};
