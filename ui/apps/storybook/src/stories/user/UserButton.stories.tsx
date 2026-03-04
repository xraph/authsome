import React from "react";
import type { Meta, StoryObj } from "@storybook/react";
import { UserButton } from "@authsome/ui-components";
import { MockAuthProvider } from "../../mocks/auth-provider";

const meta: Meta<typeof UserButton> = {
  title: "User/User Button",
  component: UserButton,
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
type Story = StoryObj<typeof UserButton>;

export const Default: Story = {
  args: {},
};

export const WithProfileLink: Story = {
  args: {
    profileUrl: "/profile",
  },
};

export const WithMenuItems: Story = {
  args: {
    profileUrl: "/profile",
    menuItems: [
      { label: "Settings", href: "/settings" },
      { label: "Billing", href: "/billing" },
      { label: "Team", href: "/team" },
    ],
  },
};
