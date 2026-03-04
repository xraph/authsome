import React from "react";
import type { Meta, StoryObj } from "@storybook/react";
import { MagicLinkForm } from "@authsome/ui-components";
import { MockAuthProvider } from "../../mocks/auth-provider";

const meta: Meta<typeof MagicLinkForm> = {
  title: "Auth Forms/Magic Link",
  component: MagicLinkForm,
  tags: ["autodocs"],
  parameters: { layout: "centered" },
};
export default meta;
type Story = StoryObj<typeof MagicLinkForm>;

export const Default: Story = {
  args: {},
};

export const WithError: Story = {
  decorators: [
    (Story) => (
      <MockAuthProvider
        simulateError
        errorMessage="Unable to send magic link. Please try again later."
      >
        <Story />
      </MockAuthProvider>
    ),
  ],
  args: {},
};
