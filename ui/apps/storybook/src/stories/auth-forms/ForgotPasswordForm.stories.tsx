import React from "react";
import type { Meta, StoryObj } from "@storybook/react";
import { ForgotPasswordForm } from "@authsome/ui-components";
import { MockAuthProvider } from "../../mocks/auth-provider";

const meta: Meta<typeof ForgotPasswordForm> = {
  title: "Auth Forms/Forgot Password",
  component: ForgotPasswordForm,
  tags: ["autodocs"],
  parameters: { layout: "centered" },
};
export default meta;
type Story = StoryObj<typeof ForgotPasswordForm>;

export const Default: Story = {
  args: {
    signInUrl: "/sign-in",
  },
};

export const WithError: Story = {
  decorators: [
    (Story) => (
      <MockAuthProvider
        simulateError
        errorMessage="No account found with this email address."
      >
        <Story />
      </MockAuthProvider>
    ),
  ],
  args: {
    signInUrl: "/sign-in",
  },
};

export const Submitted: Story = {
  args: {
    signInUrl: "/sign-in",
  },
};
