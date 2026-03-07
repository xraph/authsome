import React, { useState } from "react";
import type { Meta, StoryObj } from "@storybook/react";
import {
  SignInForm,
  DeviceAuthorizationForm,
  DeviceList,
  UserAvatar,
  AuthCard,
  Button,
} from "@authsome/ui-components";
import { MockAuthProvider, MOCK_USER } from "../../mocks/auth-provider";
import { CONFIG_ALL_ENABLED } from "../../mocks/client-config-presets";
import { Check, Monitor } from "lucide-react";

/**
 * Interactive flow: Sign In → Device Authorization → Authenticated with Devices.
 *
 * Demonstrates the device login flow where a user signs in on one device
 * and then authorizes another device (e.g. a TV or CLI) using an 8-character
 * device code. After authorization, the user can manage their trusted devices.
 */
const meta: Meta = {
  title: "Flows/Device Login",
  tags: ["autodocs"],
  parameters: { layout: "centered" },
};
export default meta;
type Story = StoryObj;

type FlowStep = "sign-in" | "device-code" | "authenticated";

function DeviceLoginRenderer() {
  const [step, setStep] = useState<FlowStep>("sign-in");

  return (
    <MockAuthProvider
      clientConfig={CONFIG_ALL_ENABLED}
      initialState={step === "sign-in" ? "unauthenticated" : "authenticated"}
      delay={800}
    >
      <div className="w-[380px]">
        {step === "sign-in" && (
          <SignInForm
            signUpUrl="/sign-up"
            forgotPasswordUrl="/forgot-password"
            onSuccess={() => setStep("device-code")}
          />
        )}

        {step === "device-code" && (
          <DeviceAuthorizationForm
            onSuccess={() => setStep("authenticated")}
            codeLength={8}
          />
        )}

        {step === "authenticated" && (
          <div className="grid gap-4">
            <div className="rounded-lg border bg-card p-6 text-center shadow-sm">
              <div className="mx-auto mb-3 flex h-12 w-12 items-center justify-center rounded-full bg-green-100 dark:bg-green-900/30">
                <Check className="h-6 w-6 text-green-600 dark:text-green-400" />
              </div>
              <div className="mb-1 flex items-center justify-center gap-2">
                <UserAvatar user={MOCK_USER} size="sm" />
                <span className="text-sm font-medium">{MOCK_USER.name}</span>
              </div>
              <p className="text-[13px] text-muted-foreground">
                Device authorized successfully.
              </p>
            </div>

            <DeviceList />
          </div>
        )}
      </div>
    </MockAuthProvider>
  );
}

/** Happy path: Sign In → Enter Device Code → Success with Device List */
export const HappyPath: Story = {
  render: () => <DeviceLoginRenderer />,
};

/** Manage Devices: starts at authenticated state with device list. */
export const ManageDevices: Story = {
  render: () => (
    <MockAuthProvider
      clientConfig={CONFIG_ALL_ENABLED}
      initialState="authenticated"
      delay={800}
    >
      <div className="w-[380px]">
        <DeviceList />
      </div>
    </MockAuthProvider>
  ),
};

/** Device authorization step only (e.g. user is already authenticated). */
export const DeviceAuthorizationOnly: Story = {
  render: () => (
    <MockAuthProvider
      clientConfig={CONFIG_ALL_ENABLED}
      initialState="authenticated"
      delay={800}
    >
      <div className="w-[380px]">
        <DeviceAuthorizationForm
          onSuccess={() => {}}
          codeLength={8}
        />
      </div>
    </MockAuthProvider>
  ),
};
