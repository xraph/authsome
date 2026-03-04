import React from "react";
import type { Meta, StoryObj } from "@storybook/react";
import { ForgotPasswordForm } from "@authsome/ui-components";

const meta: Meta = {
  title: "Pages/Full Page Forgot Password",
  tags: ["autodocs"],
  parameters: { layout: "fullscreen" },
};
export default meta;
type Story = StoryObj;

export const Default: Story = {
  render: () => (
    <div className="flex min-h-screen items-center justify-center bg-gradient-to-br from-background to-muted p-4">
      <ForgotPasswordForm signInUrl="/sign-in" />
    </div>
  ),
};
