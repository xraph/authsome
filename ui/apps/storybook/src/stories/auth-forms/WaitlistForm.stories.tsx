import React from "react";
import type { Meta, StoryObj } from "@storybook/react";
import { WaitlistForm } from "@authsome/ui-components";
import { MockAuthProvider } from "../../mocks/auth-provider";

const meta: Meta<typeof WaitlistForm> = {
  title: "Auth Forms/Waitlist",
  component: WaitlistForm,
  tags: ["autodocs"],
  parameters: { layout: "centered" },
};
export default meta;
type Story = StoryObj<typeof WaitlistForm>;

export const Default: Story = {
  args: {
    signInUrl: "/sign-in",
  },
};

export const WithLogo: Story = {
  args: {
    ...Default.args,
    logo: (
      <img
        src="https://placehold.co/120x40?text=LOGO"
        alt="Logo"
        className="h-8"
      />
    ),
  },
};

export const LeftAligned: Story = {
  args: {
    ...Default.args,
    align: "left",
  },
};

export const FlatCard: Story = {
  args: {
    ...Default.args,
    variant: "flat",
  },
};

export const WithError: Story = {
  decorators: [
    (Story) => (
      <MockAuthProvider simulateError errorMessage="This email is already on the waitlist.">
        <Story />
      </MockAuthProvider>
    ),
  ],
  args: {
    signInUrl: "/sign-in",
  },
};

export const Minimal: Story = {
  args: {},
};
