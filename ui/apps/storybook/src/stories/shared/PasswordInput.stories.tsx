import React from "react";
import type { Meta, StoryObj } from "@storybook/react";
import { PasswordInput } from "@authsome/ui-components";

const meta: Meta<typeof PasswordInput> = {
  title: "Shared/Password Input",
  component: PasswordInput,
  tags: ["autodocs"],
  parameters: { layout: "centered" },
};
export default meta;
type Story = StoryObj<typeof PasswordInput>;

export const Default: Story = {
  args: {},
};
