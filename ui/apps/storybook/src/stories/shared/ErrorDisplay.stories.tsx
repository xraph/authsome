import React from "react";
import type { Meta, StoryObj } from "@storybook/react";
import { ErrorDisplay } from "@authsome/ui-components";

const meta: Meta<typeof ErrorDisplay> = {
  title: "Shared/Error Display",
  component: ErrorDisplay,
  tags: ["autodocs"],
  parameters: { layout: "centered" },
};
export default meta;
type Story = StoryObj<typeof ErrorDisplay>;

export const Default: Story = {
  args: {
    error: "Something went wrong. Please try again later.",
  },
};

export const NoError: Story = {
  args: {
    error: null,
  },
};
