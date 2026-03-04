import React from "react";
import type { Meta, StoryObj } from "@storybook/react";
import { MFAChallengeFormStyled as MFAChallengeForm } from "@authsome/ui-components";
import { MockAuthProvider } from "../../mocks/auth-provider";

const meta: Meta<typeof MFAChallengeForm> = {
  title: "Auth Forms/MFA Challenge",
  component: MFAChallengeForm,
  tags: ["autodocs"],
  parameters: { layout: "centered" },
};
export default meta;
type Story = StoryObj<typeof MFAChallengeForm>;

export const Default: Story = {
  args: {
    enrollmentId: "enroll_123",
  },
};

export const WithError: Story = {
  decorators: [
    (Story) => (
      <MockAuthProvider
        simulateError
        errorMessage="Invalid verification code. Please try again."
      >
        <Story />
      </MockAuthProvider>
    ),
  ],
  args: {
    enrollmentId: "enroll_123",
  },
};
