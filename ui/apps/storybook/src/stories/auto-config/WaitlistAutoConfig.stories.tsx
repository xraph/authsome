import React from "react";
import type { Meta, StoryObj } from "@storybook/react";
import { WaitlistForm } from "@authsome/ui-components";
import { MockAuthProvider } from "../../mocks/auth-provider";
import { CONFIG_WAITLIST_ENABLED } from "../../mocks/client-config-presets";

/**
 * These stories test the auto-configuration behavior of `WaitlistForm`.
 *
 * When a `publishableKey` is set on `AuthProvider`, the backend returns
 * a `ClientConfig` that tells the form whether waitlist mode is enabled.
 *
 * Here we simulate this by passing `clientConfig` to `MockAuthProvider`.
 */
const meta: Meta<typeof WaitlistForm> = {
  title: "Auto Config/Waitlist",
  component: WaitlistForm,
  tags: ["autodocs"],
  parameters: { layout: "centered" },
};
export default meta;
type Story = StoryObj<typeof WaitlistForm>;

const defaultArgs = {
  signInUrl: "/sign-in",
};

/** Waitlist enabled. The form is rendered normally. */
export const WaitlistEnabled: Story = {
  decorators: [
    (Story) => (
      <MockAuthProvider clientConfig={CONFIG_WAITLIST_ENABLED}>
        <Story />
      </MockAuthProvider>
    ),
  ],
  args: defaultArgs,
};

/** Waitlist disabled. The form renders nothing (returns null). */
export const WaitlistDisabled: Story = {
  decorators: [
    (Story) => (
      <MockAuthProvider
        clientConfig={{
          ...CONFIG_WAITLIST_ENABLED,
          waitlist: { enabled: false },
        }}
      >
        <Story />
      </MockAuthProvider>
    ),
  ],
  args: defaultArgs,
};
