import React from "react";
import type { Meta, StoryObj } from "@storybook/react";
import { fn } from "@storybook/test";
import { SignInForm } from "@authsome/ui-components";
import { MockAuthProvider } from "../../mocks/auth-provider";

const meta: Meta<typeof SignInForm> = {
  title: "Auth Forms/Sign In",
  component: SignInForm,
  tags: ["autodocs"],
  parameters: { layout: "centered" },
};
export default meta;
type Story = StoryObj<typeof SignInForm>;

export const Default: Story = {
  args: {
    signUpUrl: "/sign-up",
    forgotPasswordUrl: "/forgot-password",
  },
};

export const WithSocialProviders: Story = {
  args: {
    ...Default.args,
    socialProviders: [
      { id: "google", name: "Google" },
      { id: "github", name: "GitHub" },
      { id: "apple", name: "Apple" },
    ],
    onSocialLogin: (id: string) => console.log("Social:", id),
  },
};

export const WithError: Story = {
  decorators: [
    (Story) => (
      <MockAuthProvider
        simulateError
        errorMessage="Invalid email or password. Please check your credentials and try again."
      >
        <Story />
      </MockAuthProvider>
    ),
  ],
  args: {
    signUpUrl: "/sign-up",
    forgotPasswordUrl: "/forgot-password",
  },
};

export const Minimal: Story = {
  args: {},
};

export const WithPasskey: Story = {
  args: {
    ...Default.args,
    showPasskey: true,
    onPasskeySuccess: fn(),
  },
};

export const IconRowSocial: Story = {
  args: {
    ...WithSocialProviders.args,
    socialLayout: "icon-row",
  },
};

export const VerticalSocial: Story = {
  args: {
    ...WithSocialProviders.args,
    socialLayout: "vertical",
  },
};

export const FullFeatured: Story = {
  args: {
    signUpUrl: "/sign-up",
    forgotPasswordUrl: "/forgot-password",
    showPasskey: true,
    onPasskeySuccess: fn(),
    socialProviders: [
      { id: "google", name: "Google" },
      { id: "github", name: "GitHub" },
      { id: "apple", name: "Apple" },
    ],
    onSocialLogin: fn(),
    socialLayout: "icon-row",
  },
};
