import React from "react";
import type { Meta, StoryObj } from "@storybook/react";
import { SignInForm, SignUpForm } from "@authsome/ui-components";
import { MockAuthProvider } from "../../mocks/auth-provider";
import {
  CONFIG_ALL_ENABLED,
  CONFIG_SOCIAL_ONLY,
} from "../../mocks/client-config-presets";

/**
 * Full-page stories that demonstrate auto-configured sign-in and sign-up forms
 * as they would appear in a real application. These render with `layout: "fullscreen"`.
 */
const meta: Meta = {
  title: "Auto Config/Full Page",
  tags: ["autodocs"],
  parameters: { layout: "fullscreen" },
};
export default meta;
type Story = StoryObj;

/** Full page sign-in with all auth methods auto-configured from the backend. */
export const SignInAllMethods: Story = {
  render: () => (
    <MockAuthProvider clientConfig={CONFIG_ALL_ENABLED}>
      <div className="flex min-h-screen items-center justify-center bg-gradient-to-br from-background to-muted p-4">
        <SignInForm
          signUpUrl="/sign-up"
          forgotPasswordUrl="/forgot-password"
        />
      </div>
    </MockAuthProvider>
  ),
};

/** Full page sign-in with only social providers auto-configured. */
export const SignInSocialOnly: Story = {
  render: () => (
    <MockAuthProvider clientConfig={CONFIG_SOCIAL_ONLY}>
      <div className="flex min-h-screen items-center justify-center bg-gradient-to-br from-background to-muted p-4">
        <SignInForm
          signUpUrl="/sign-up"
          forgotPasswordUrl="/forgot-password"
        />
      </div>
    </MockAuthProvider>
  ),
};

/** Full page sign-up with all auth methods auto-configured from the backend. */
export const SignUpAllMethods: Story = {
  render: () => (
    <MockAuthProvider clientConfig={CONFIG_ALL_ENABLED}>
      <div className="flex min-h-screen items-center justify-center bg-gradient-to-br from-background to-muted p-4">
        <SignUpForm signInUrl="/sign-in" />
      </div>
    </MockAuthProvider>
  ),
};
