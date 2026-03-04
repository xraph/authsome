import React from "react";
import type { Meta, StoryObj } from "@storybook/react";
import { ResetPasswordForm } from "@authsome/ui-components";
import { MockAuthProvider } from "../../mocks/auth-provider";

const meta: Meta<typeof ResetPasswordForm> = {
  title: "Auth Forms/Reset Password",
  component: ResetPasswordForm,
  tags: ["autodocs"],
  parameters: { layout: "centered" },
};
export default meta;
type Story = StoryObj<typeof ResetPasswordForm>;

export const Default: Story = {
  args: {
    token: "mock-token",
  },
};

export const WithError: Story = {
  decorators: [
    (Story) => (
      <MockAuthProvider
        simulateError
        errorMessage="This reset link has expired. Please request a new one."
      >
        <Story />
      </MockAuthProvider>
    ),
  ],
  args: {
    token: "mock-token",
  },
};
