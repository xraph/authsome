import React from "react";
import type { Meta, StoryObj } from "@storybook/react";
import { SignUpForm, type SocialProvider } from "@authsome/ui-components";

const socialProviders: SocialProvider[] = [
  { id: "google", name: "Google" },
  { id: "github", name: "GitHub" },
];

const meta: Meta = {
  title: "Pages/Full Page Sign Up",
  tags: ["autodocs"],
  parameters: { layout: "fullscreen" },
};
export default meta;
type Story = StoryObj;

export const Default: Story = {
  render: () => (
    <div className="flex min-h-screen items-center justify-center bg-gradient-to-br from-background to-muted p-4">
      <SignUpForm
        signInUrl="/sign-in"
        socialProviders={socialProviders}
        onSocialLogin={(id) => console.log("Social login:", id)}
      />
    </div>
  ),
};
