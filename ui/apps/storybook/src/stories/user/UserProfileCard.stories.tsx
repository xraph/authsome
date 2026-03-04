import React from "react";
import type { Meta, StoryObj } from "@storybook/react";
import { UserProfileCard } from "@authsome/ui-components";
import { MockAuthProvider } from "../../mocks/auth-provider";

const meta: Meta<typeof UserProfileCard> = {
  title: "User/Profile Card",
  component: UserProfileCard,
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
type Story = StoryObj<typeof UserProfileCard>;

export const Default: Story = {
  args: {},
};
