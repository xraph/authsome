import React from "react";
import type { Meta, StoryObj } from "@storybook/react";
import { fn } from "@storybook/test";
import { SignInForm } from "@authsome/ui-components";
import { MockAuthProvider } from "../../mocks/auth-provider";
import {
  CONFIG_ALL_ENABLED,
  CONFIG_SOCIAL_ONLY,
  CONFIG_PASSKEY_ONLY,
  CONFIG_PASSWORD_ONLY,
  CONFIG_SOCIAL_AND_PASSKEY,
  CONFIG_EMPTY,
  CONFIG_WITH_BRANDING,
} from "../../mocks/client-config-presets";

/**
 * These stories test the auto-configuration behavior of `SignInForm`.
 *
 * When a `publishableKey` is set on `AuthProvider`, the backend returns
 * a `ClientConfig` that tells the form which auth methods are enabled.
 * The form auto-derives social providers and passkey support from this config.
 *
 * Here we simulate this by passing `clientConfig` to `MockAuthProvider`,
 * which injects the config into the auth context just as the real provider would.
 */
const meta: Meta<typeof SignInForm> = {
  title: "Auto Config/Sign In",
  component: SignInForm,
  tags: ["autodocs"],
  parameters: { layout: "centered" },
};
export default meta;
type Story = StoryObj<typeof SignInForm>;

const defaultArgs = {
  signUpUrl: "/sign-up",
  forgotPasswordUrl: "/forgot-password",
};

/** All auth methods enabled. Social buttons and passkey button appear automatically. */
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

/** Only passkey enabled. Passkey button appears, no social buttons. */
export const PasskeyOnly: Story = {
  decorators: [
    (Story) => (
      <MockAuthProvider clientConfig={CONFIG_PASSKEY_ONLY}>
        <Story />
      </MockAuthProvider>
    ),
  ],
  args: defaultArgs,
};

/** Only password enabled. Plain email/password form, no social or passkey. */
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

/** Social + passkey enabled. Both social buttons and passkey button appear. */
export const SocialAndPasskey: Story = {
  decorators: [
    (Story) => (
      <MockAuthProvider clientConfig={CONFIG_SOCIAL_AND_PASSKEY}>
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

/**
 * Explicit props override auto-derived config values.
 *
 * Even though CONFIG_ALL_ENABLED includes Google/GitHub/Apple,
 * the explicit `socialProviders` prop with only Twitter takes precedence.
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
    socialProviders: [{ id: "twitter", name: "Twitter" }],
    onSocialLogin: fn(),
  },
};

/**
 * Explicit `showPasskey={false}` overrides config's `passkey.enabled=true`.
 *
 * The config says passkey is enabled but the explicit prop disables it.
 */
export const ExplicitPasskeyOff: Story = {
  decorators: [
    (Story) => (
      <MockAuthProvider clientConfig={CONFIG_ALL_ENABLED}>
        <Story />
      </MockAuthProvider>
    ),
  ],
  args: {
    ...defaultArgs,
    showPasskey: false,
  },
};

/** Config with branding (app name and logo URL). */
export const WithBranding: Story = {
  decorators: [
    (Story) => (
      <MockAuthProvider clientConfig={CONFIG_WITH_BRANDING}>
        <Story />
      </MockAuthProvider>
    ),
  ],
  args: defaultArgs,
};

/** Empty config object. Nothing is explicitly enabled, renders plain form. */
export const EmptyConfig: Story = {
  decorators: [
    (Story) => (
      <MockAuthProvider clientConfig={CONFIG_EMPTY}>
        <Story />
      </MockAuthProvider>
    ),
  ],
  args: defaultArgs,
};
