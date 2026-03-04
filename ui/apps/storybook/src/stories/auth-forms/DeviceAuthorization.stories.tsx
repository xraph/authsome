import React from "react";
import type { Meta, StoryObj } from "@storybook/react";
import { fn } from "@storybook/test";
import { DeviceAuthorizationForm } from "@authsome/ui-components";

const meta: Meta<typeof DeviceAuthorizationForm> = {
  title: "Auth Forms/Device Authorization",
  component: DeviceAuthorizationForm,
  tags: ["autodocs"],
  parameters: { layout: "centered" },
};
export default meta;
type Story = StoryObj<typeof DeviceAuthorizationForm>;

export const Default: Story = {
  args: {
    onSuccess: fn(),
    onError: fn(),
  },
};

export const SixDigitCode: Story = {
  args: {
    codeLength: 6,
    onSuccess: fn(),
  },
};

export const WithLogo: Story = {
  args: {
    onSuccess: fn(),
  },
  render: (args) => (
    <DeviceAuthorizationForm
      {...args}
      logo={
        <div className="flex h-10 w-10 items-center justify-center rounded-full bg-primary text-primary-foreground font-bold">
          A
        </div>
      }
    />
  ),
};
