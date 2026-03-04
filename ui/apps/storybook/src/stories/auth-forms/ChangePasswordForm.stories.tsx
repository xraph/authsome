import React from "react";
import type { Meta, StoryObj } from "@storybook/react";
import { ChangePasswordForm } from "@authsome/ui-components";
import { MockAuthProvider } from "../../mocks/auth-provider";

const meta: Meta<typeof ChangePasswordForm> = {
  title: "Auth Forms/Change Password",
  component: ChangePasswordForm,
  tags: ["autodocs"],
  parameters: { layout: "centered" },
  decorators: [
    (Story) => (
      <MockAuthProvider initialState="authenticated">
        <Story />
      </MockAuthProvider>
    ),
  ],
};
export default meta;
type Story = StoryObj<typeof ChangePasswordForm>;

export const Default: Story = {
  args: {},
};

export const WithError: Story = {
  decorators: [
    (Story) => (
      <MockAuthProvider
        initialState="authenticated"
        simulateError
        errorMessage="Current password is incorrect."
      >
        <Story />
      </MockAuthProvider>
    ),
  ],
  args: {},
};
