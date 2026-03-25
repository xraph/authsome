import React from "react";
import type { Meta, StoryObj } from "@storybook/react";
import { fn } from "@storybook/test";
import { SignUpForm } from "@authsome/ui-components";
import { MockAuthProvider } from "../../mocks/auth-provider";

const meta: Meta<typeof SignUpForm> = {
  title: "Auth Forms/Sign Up",
  component: SignUpForm,
  tags: ["autodocs"],
  parameters: { layout: "centered" },
};
export default meta;
type Story = StoryObj<typeof SignUpForm>;

export const Default: Story = {
  args: {
    signInUrl: "/sign-in",
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
        errorMessage="An account with this email already exists. Please sign in instead."
      >
        <Story />
      </MockAuthProvider>
    ),
  ],
  args: {
    signInUrl: "/sign-in",
  },
};

export const Minimal: Story = {
  args: {},
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

export const LeftAligned: Story = {
  args: {
    ...Default.args,
    align: "left",
  },
};

export const FlatCard: Story = {
  args: {
    ...Default.args,
    variant: "flat",
  },
};

export const BorderedCard: Story = {
  args: {
    ...Default.args,
    variant: "bordered",
  },
};

export const WithForgotPassword: Story = {
  args: {
    ...Default.args,
    forgotPasswordUrl: "/forgot-password",
  },
};

export const LeftAlignedFlat: Story = {
  args: {
    ...Default.args,
    align: "left",
    variant: "flat",
    forgotPasswordUrl: "/forgot-password",
  },
};
