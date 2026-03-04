import React from "react";
import type { Meta, StoryObj } from "@storybook/react";
import { AuthCard } from "@authsome/ui-components";

const meta: Meta<typeof AuthCard> = {
  title: "Shared/Auth Card",
  component: AuthCard,
  tags: ["autodocs"],
  parameters: { layout: "centered" },
};
export default meta;
type Story = StoryObj<typeof AuthCard>;

export const Default: Story = {
  args: {
    title: "Welcome Back",
    description: "Sign in to your account to continue",
    children: (
      <div className="space-y-4">
        <div className="space-y-2">
          <label className="text-sm font-medium" htmlFor="email">
            Email
          </label>
          <input
            id="email"
            type="email"
            placeholder="name@example.com"
            className="w-full rounded-md border px-3 py-2 text-sm"
          />
        </div>
        <div className="space-y-2">
          <label className="text-sm font-medium" htmlFor="password">
            Password
          </label>
          <input
            id="password"
            type="password"
            placeholder="Enter your password"
            className="w-full rounded-md border px-3 py-2 text-sm"
          />
        </div>
        <button className="w-full rounded-md bg-primary px-4 py-2 text-sm text-primary-foreground">
          Sign In
        </button>
      </div>
    ),
  },
};

export const WithLogo: Story = {
  args: {
    ...Default.args,
    logo: (
      <div className="flex items-center justify-center">
        <div className="flex h-10 w-10 items-center justify-center rounded-full bg-primary text-primary-foreground font-bold">
          A
        </div>
      </div>
    ),
  },
};

export const WithFooter: Story = {
  args: {
    ...Default.args,
    footer: (
      <p className="text-center text-sm text-muted-foreground">
        By continuing, you agree to our{" "}
        <a href="/terms" className="underline">
          Terms of Service
        </a>{" "}
        and{" "}
        <a href="/privacy" className="underline">
          Privacy Policy
        </a>
        .
      </p>
    ),
  },
};
