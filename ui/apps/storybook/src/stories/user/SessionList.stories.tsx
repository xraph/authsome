import React from "react";
import type { Meta, StoryObj } from "@storybook/react";
import { fn } from "@storybook/test";
import { SessionList } from "@authsome/ui-components";
import { MockAuthProvider } from "../../mocks/auth-provider";

const meta: Meta<typeof SessionList> = {
  title: "User/Session List",
  component: SessionList,
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
type Story = StoryObj<typeof SessionList>;

export const Default: Story = {
  args: {
    onRevoke: fn(),
  },
};

export const WithCurrentSession: Story = {
  args: {
    currentSessionToken: "mock_session_token_abc123",
    onRevoke: fn(),
  },
};
