import React from "react";
import type { Meta, StoryObj } from "@storybook/react";
import { LoadingSpinner } from "@authsome/ui-components";

const meta: Meta<typeof LoadingSpinner> = {
  title: "Shared/Loading Spinner",
  component: LoadingSpinner,
  tags: ["autodocs"],
  parameters: { layout: "centered" },
};
export default meta;
type Story = StoryObj<typeof LoadingSpinner>;

export const Small: Story = {
  args: {
    size: "sm",
  },
};

export const Default: Story = {
  args: {},
};

export const Large: Story = {
  args: {
    size: "lg",
  },
};
