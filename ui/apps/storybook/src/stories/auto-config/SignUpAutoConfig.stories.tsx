import React from "react";
import type { Meta, StoryObj } from "@storybook/react";
import { fn } from "@storybook/test";
import { SignUpForm } from "@authsome/ui-components";
import { MockAuthProvider } from "../../mocks/auth-provider";
import {
  CONFIG_ALL_ENABLED,
  CONFIG_SOCIAL_ONLY,
  CONFIG_PASSWORD_ONLY,
} from "../../mocks/client-config-presets";

/**
 * These stories test the auto-configuration behavior of `SignUpForm`.
 *
 * When a `publishableKey` is set on `AuthProvider`, the backend returns
 * a `ClientConfig` that tells the form which auth methods are enabled.
 * The form auto-derives social providers from this config.
 *
 * Here we simulate this by passing `clientConfig` to `MockAuthProvider`.
 */
const meta: Meta<typeof SignUpForm> = {
  title: "Auto Config/Sign Up",
  component: SignUpForm,
  tags: ["autodocs"],
  parameters: { layout: "centered" },
};
export default meta;
type Story = StoryObj<typeof SignUpForm>;

const defaultArgs = {
  signInUrl: "/sign-in",
};

/** All auth methods enabled. Social buttons appear automatically. */
export const AllMethodsEnabled: Story = {
  decorators: [
    (Story) => (
      <MockAuthProvider clientConfig={CONFIG_ALL_ENABLED}>
        <Story />
      </MockAuthProvider>
    ),
  ],
  args: defaultArgs,
};

/** Only social providers enabled. Social buttons appear, no passkey. */
export const SocialOnly: Story = {
  decorators: [
    (Story) => (
      <MockAuthProvider clientConfig={CONFIG_SOCIAL_ONLY}>
        <Story />
      </MockAuthProvider>
    ),
  ],
  args: defaultArgs,
};

/** Only password enabled. Plain registration form. */
export const PasswordOnly: Story = {
  decorators: [
    (Story) => (
      <MockAuthProvider clientConfig={CONFIG_PASSWORD_ONLY}>
        <Story />
      </MockAuthProvider>
    ),
  ],
  args: defaultArgs,
};

/** No client config at all (simulates no publishableKey). Falls back to plain form. */
export const NoConfig: Story = {
  args: defaultArgs,
};

/** You see the waitlist message after entering an email awaiting approval. */
export const PendingWaitlist: Story = {
  decorators: [
    (Story) => (
      <MockAuthProvider
        clientConfig={{ waitlist: { enabled: true } }}
        waitlistStatus="pending"
      >
        <Story />
      </MockAuthProvider>
    ),
  ],
  args: defaultArgs,
};

/** An approved email continues to the signup details. */
export const ApprovedWaitlist: Story = {
  decorators: [
    (Story) => (
      <MockAuthProvider
        clientConfig={{ waitlist: { enabled: true } }}
        waitlistStatus="approved"
      >
        <Story />
      </MockAuthProvider>
    ),
  ],
  args: defaultArgs,
};

/** A rejected entry keeps the signup details hidden. */
export const RejectedWaitlist: Story = {
  decorators: [
    (Story) => (
      <MockAuthProvider
        clientConfig={{ waitlist: { enabled: true } }}
        waitlistStatus="rejected"
      >
        <Story />
      </MockAuthProvider>
    ),
  ],
  args: defaultArgs,
};

/**
 * Explicit props override auto-derived config values.
 *
 * Even though CONFIG_ALL_ENABLED includes Google/GitHub/Apple,
 * the explicit `socialProviders` prop with only Microsoft takes precedence.
 */
export const ExplicitPropsOverride: Story = {
  decorators: [
    (Story) => (
      <MockAuthProvider clientConfig={CONFIG_ALL_ENABLED}>
        <Story />
      </MockAuthProvider>
    ),
  ],
  args: {
    ...defaultArgs,
    socialProviders: [{ id: "microsoft", name: "Microsoft" }],
    onSocialLogin: fn(),
  },
};

/** Closed signup keeps the sign-in link available. */
export const SignupClosed: Story = {
  decorators: [
    (Story) => (
      <MockAuthProvider clientConfig={{ signup_enabled: false }}>
        <Story />
      </MockAuthProvider>
    ),
  ],
  args: defaultArgs,
};

/** Configured consent must be accepted before signup. */
export const RequiredConsent: Story = {
  decorators: [
    (Story) => (
      <MockAuthProvider
        clientConfig={{
          signup_fields: [
            {
              key: "plan",
              label: "Plan",
              type: "radio",
              order: 1,
              options: [
                { label: "Personal", value: "personal" },
                { label: "Team", value: "team" },
              ],
              validation: { required: true },
            },
            {
              key: "terms",
              label: "Accept the terms",
              type: "checkbox",
              order: 2,
              validation: { required: true },
            },
          ],
        }}
      >
        <Story />
      </MockAuthProvider>
    ),
  ],
  args: defaultArgs,
};
