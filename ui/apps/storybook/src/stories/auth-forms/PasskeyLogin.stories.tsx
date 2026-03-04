import type { Meta, StoryObj } from "@storybook/react";
import { fn } from "@storybook/test";
import { PasskeyLoginButton } from "@authsome/ui-components";

const meta: Meta<typeof PasskeyLoginButton> = {
  title: "Auth Forms/Passkey Login Button",
  component: PasskeyLoginButton,
  tags: ["autodocs"],
  parameters: { layout: "centered" },
};
export default meta;
type Story = StoryObj<typeof PasskeyLoginButton>;

export const Default: Story = {
  args: {
    onSuccess: fn(),
    onError: fn(),
  },
};

export const OutlineVariant: Story = {
  args: {
    variant: "outline",
    onSuccess: fn(),
  },
};

export const LargeSize: Story = {
  args: {
    size: "lg",
    onSuccess: fn(),
  },
};

export const CustomLabel: Story = {
  args: {
    label: "Continue with passkey",
    onSuccess: fn(),
  },
};

export const GhostVariant: Story = {
  args: {
    variant: "ghost",
    size: "sm",
    onSuccess: fn(),
  },
};
