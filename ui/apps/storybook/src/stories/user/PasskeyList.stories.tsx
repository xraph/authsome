import React from "react";
import type { Meta, StoryObj } from "@storybook/react";
import { fn } from "@storybook/test";
import { PasskeyList } from "@authsome/ui-components";
import { MockAuthProvider } from "../../mocks/auth-provider";

const meta: Meta<typeof PasskeyList> = {
  title: "User/Passkey List",
  component: PasskeyList,
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
type Story = StoryObj<typeof PasskeyList>;

export const Default: Story = {
  args: {
    onDelete: fn(),
    onRegister: fn(),
  },
};

export const WithoutRegister: Story = {
  args: {
    onDelete: fn(),
  },
};
