import React from "react";
import type { Meta, StoryObj } from "@storybook/react";
import { fn } from "@storybook/test";
import { DeviceList } from "@authsome/ui-components";
import { MockAuthProvider } from "../../mocks/auth-provider";

const meta: Meta<typeof DeviceList> = {
  title: "User/Device List",
  component: DeviceList,
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
type Story = StoryObj<typeof DeviceList>;

export const Default: Story = {
  args: {
    onTrust: fn(),
    onDelete: fn(),
  },
};

export const ReadOnly: Story = {
  args: {},
};
