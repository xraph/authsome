import React from "react";
import type { Meta, StoryObj } from "@storybook/react";
import { fn } from "@storybook/test";
import { SocialButtons } from "@authsome/ui-components";

const meta: Meta<typeof SocialButtons> = {
  title: "Shared/Social Buttons",
  component: SocialButtons,
  tags: ["autodocs"],
  parameters: { layout: "centered" },
};
export default meta;
type Story = StoryObj<typeof SocialButtons>;

export const Default: Story = {
  args: {
    providers: [
      { id: "google", name: "Google" },
      { id: "github", name: "GitHub" },
      { id: "apple", name: "Apple" },
    ],
    onSocialLogin: (id: string) => console.log("Social login:", id),
  },
};

export const IconRow: Story = {
  args: {
    ...Default.args,
    layout: "icon-row",
  },
};

export const Vertical: Story = {
  args: {
    ...Default.args,
    layout: "vertical",
  },
};

export const IconRowMany: Story = {
  args: {
    providers: [
      { id: "google", name: "Google" },
      { id: "github", name: "GitHub" },
      { id: "apple", name: "Apple" },
      { id: "microsoft", name: "Microsoft" },
      { id: "twitter", name: "Twitter" },
    ],
    onProviderClick: fn(),
    layout: "icon-row",
  },
};

export const SingleProviderVertical: Story = {
  args: {
    providers: [{ id: "google", name: "Google" }],
    onProviderClick: fn(),
    layout: "vertical",
  },
};
