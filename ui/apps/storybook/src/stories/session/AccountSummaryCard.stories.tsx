import type { Meta, StoryObj } from "@storybook/react";
import { AccountSummaryCard } from "@authsome/ui-components";

const meta: Meta<typeof AccountSummaryCard> = {
  title: "Session/Account Summary Card",
  component: AccountSummaryCard,
  parameters: { layout: "centered" },
  decorators: [
    (Story) => (
      <div className="w-[420px]">
        <Story />
      </div>
    ),
  ],
};
export default meta;
type Story = StoryObj<typeof AccountSummaryCard>;

const identity = {
  name: "Rex Raphael",
  identifier: "rex@acme.com",
  role: "Owner of Acme",
};

const today = () => {
  const d = new Date();
  d.setHours(8, 41, 0, 0);
  return d.toISOString();
};

export const Default: Story = {
  args: {
    identity,
    summary: {
      lastSignInAt: today(),
      ipAddress: "81.193.4.22",
      device: "Chrome on macOS",
      trustedDeviceCount: 2,
      twoStepEnabled: true,
    },
    footer: "Something look wrong? Sign out everywhere from Settings, under Security.",
  },
};

export const TwoStepOff: Story = {
  args: {
    ...Default.args,
    summary: { ...Default.args!.summary!, twoStepEnabled: false },
  },
};

export const TwoStepUnknown: Story = {
  name: "Two-step unknown (row hidden)",
  args: {
    ...Default.args,
    summary: { ...Default.args!.summary!, twoStepEnabled: null },
  },
};

export const Yesterday: Story = {
  args: {
    ...Default.args,
    summary: {
      ...Default.args!.summary!,
      lastSignInAt: new Date(Date.now() - 26 * 3600_000).toISOString(),
    },
  },
};

export const LongAgo: Story = {
  args: {
    ...Default.args,
    summary: {
      ...Default.args!.summary!,
      lastSignInAt: new Date("2026-03-04T09:15:00Z").toISOString(),
    },
  },
};

export const NoRole: Story = {
  args: {
    ...Default.args,
    identity: { name: "Rex Raphael", identifier: "rex@acme.com" },
  },
};
