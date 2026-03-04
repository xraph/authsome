import React from "react";
import type { Meta, StoryObj } from "@storybook/react";
import { UserAvatar } from "@authsome/ui-components";
import { MockAuthProvider } from "../../mocks/auth-provider";

const meta: Meta<typeof UserAvatar> = {
  title: "User/User Avatar",
  component: UserAvatar,
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
type Story = StoryObj<typeof UserAvatar>;

export const Default: Story = {
  args: {},
};
