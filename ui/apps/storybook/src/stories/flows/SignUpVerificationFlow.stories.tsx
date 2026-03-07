import React, { useState } from "react";
import type { Meta, StoryObj } from "@storybook/react";
import {
  SignUpForm,
  EmailVerificationForm,
  UserAvatar,
} from "@authsome/ui-components";
import { MockAuthProvider, MOCK_USER } from "../../mocks/auth-provider";
import { CONFIG_ALL_ENABLED } from "../../mocks/client-config-presets";
import { Check } from "lucide-react";

/**
 * Interactive multi-step flow: Sign Up → Email Verification → Authenticated.
 *
 * Demonstrates the full sign-up experience when email verification is required.
 * After creating an account, the user is asked to verify their email with a
 * 6-digit OTP code before gaining access.
 */
const meta: Meta = {
  title: "Flows/Sign Up → Email Verification",
  tags: ["autodocs"],
  parameters: { layout: "centered" },
};
export default meta;
type Story = StoryObj;

type FlowStep = "sign-up" | "verify-email" | "success";

function SignUpVerificationRenderer() {
  const [step, setStep] = useState<FlowStep>("sign-up");

  return (
    <MockAuthProvider
      clientConfig={CONFIG_ALL_ENABLED}
      signUpBehavior="email_verification_required"
      delay={800}
    >
      <div className="w-[380px]">
        {step === "sign-up" && (
          <SignUpForm
            signInUrl="/sign-in"
            onSuccess={() => {
              // signUpBehavior triggers verification required
              setStep("verify-email");
            }}
          />
        )}

        {step === "verify-email" && (
          <EmailVerificationForm
            email="jane@example.com"
            onSuccess={() => setStep("success")}
            onResend={() => {}}
          />
        )}

        {step === "success" && (
          <div className="rounded-lg border bg-card p-6 text-center shadow-sm">
            <div className="mx-auto mb-3 flex h-12 w-12 items-center justify-center rounded-full bg-green-100 dark:bg-green-900/30">
              <Check className="h-6 w-6 text-green-600 dark:text-green-400" />
            </div>
            <div className="mb-1 flex items-center justify-center gap-2">
              <UserAvatar user={MOCK_USER} size="sm" />
              <span className="text-sm font-medium">Welcome!</span>
            </div>
            <p className="text-[13px] text-muted-foreground">
              Your account has been created and verified.
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

/** Happy path: Sign Up → Verify Email → Success */
export const HappyPath: Story = {
  render: () => <SignUpVerificationRenderer />,
};

/** Starting at the verification step (e.g. user navigated directly). */
export const VerificationStepOnly: Story = {
  render: () => (
    <MockAuthProvider clientConfig={CONFIG_ALL_ENABLED} delay={800}>
      <div className="w-[380px]">
        <EmailVerificationForm
          email="jane@example.com"
          onSuccess={() => {}}
          onResend={() => {}}
        />
      </div>
    </MockAuthProvider>
  ),
};
