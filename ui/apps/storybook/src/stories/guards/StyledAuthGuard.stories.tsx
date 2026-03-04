import React from "react";
import type { Meta, StoryObj } from "@storybook/react";
import { StyledAuthGuard } from "@authsome/ui-components";
import { MockAuthProvider } from "../../mocks/auth-provider";

const meta: Meta<typeof StyledAuthGuard> = {
  title: "Guards/Auth Guard",
  component: StyledAuthGuard,
  tags: ["autodocs"],
  parameters: { layout: "centered" },
};
export default meta;
type Story = StoryObj<typeof StyledAuthGuard>;

export const Authenticated: Story = {
  decorators: [
    (Story) => (
      <MockAuthProvider initialState="authenticated">
        <Story />
      </MockAuthProvider>
    ),
  ],
  args: {
    children: (
      <div className="p-8 text-center">
        <h2 className="text-lg font-semibold">Protected Content</h2>
        <p className="text-muted-foreground">
          This content is only visible to authenticated users.
        </p>
      </div>
    ),
  },
};

export const Unauthenticated: Story = {
  decorators: [
    (Story) => (
      <MockAuthProvider initialState="unauthenticated">
        <Story />
      </MockAuthProvider>
    ),
  ],
  args: {
    children: (
      <div className="p-8 text-center">
        <h2 className="text-lg font-semibold">Protected Content</h2>
        <p className="text-muted-foreground">
          You should not see this if unauthenticated.
        </p>
      </div>
    ),
  },
};

export const Loading: Story = {
  decorators: [
    (Story) => (
      <MockAuthProvider initialState="loading">
        <Story />
      </MockAuthProvider>
    ),
  ],
  args: {
    children: (
      <div className="p-8 text-center">
        <h2 className="text-lg font-semibold">Protected Content</h2>
        <p className="text-muted-foreground">Loading state is displayed.</p>
      </div>
    ),
  },
};
