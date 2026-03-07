import React, { useState } from "react";
import type { Meta, StoryObj } from "@storybook/react";
import {
  SignInForm,
  EmailVerificationForm,
  UserAvatar,
} from "@authsome/ui-components";
import { MockAuthProvider, MOCK_USER } from "../../mocks/auth-provider";
import { CONFIG_ALL_ENABLED } from "../../mocks/client-config-presets";
import { Check } from "lucide-react";

/**
 * Interactive flow: Sign In → Email Verification Required → Authenticated.
 *
 * Demonstrates what happens when a user tries to sign in but their email
 * has not yet been verified. The sign-in succeeds but redirects to the
 * email verification form before granting full access.
 */
const meta: Meta = {
  title: "Flows/Sign In → Email Verification",
  tags: ["autodocs"],
  parameters: { layout: "centered" },
};
export default meta;
type Story = StoryObj;

type FlowStep = "sign-in" | "verify-email" | "success";

function SignInVerificationRenderer() {
  const [step, setStep] = useState<FlowStep>("sign-in");

  return (
    <MockAuthProvider
      clientConfig={CONFIG_ALL_ENABLED}
      signInBehavior="email_verification_required"
      delay={800}
    >
      <div className="w-[380px]">
        {step === "sign-in" && (
          <SignInForm
            signUpUrl="/sign-up"
            forgotPasswordUrl="/forgot-password"
            onSuccess={() => setStep("verify-email")}
          />
        )}

        {step === "verify-email" && (
          <EmailVerificationForm
            email={MOCK_USER.email ?? "jane@example.com"}
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
              <span className="text-sm font-medium">{MOCK_USER.name}</span>
            </div>
            <p className="text-[13px] text-muted-foreground">
              Email verified! You&apos;re now signed in.
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

/** Happy path: Sign In → Email Verification → Success */
export const HappyPath: Story = {
  render: () => <SignInVerificationRenderer />,
};
