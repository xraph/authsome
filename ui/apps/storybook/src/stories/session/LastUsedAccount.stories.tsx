import type { Meta, StoryObj } from "@storybook/react";
import { LastUsedAccount } from "@authsome/ui-components";

const meta: Meta<typeof LastUsedAccount> = {
  title: "Session/Last Used Account",
  component: LastUsedAccount,
  parameters: { layout: "centered" },
  decorators: [
    (Story) => (
      <div className="w-[400px]">
        <Story />
      </div>
    ),
  ],
};
export default meta;
type Story = StoryObj<typeof LastUsedAccount>;

const rex = {
  userId: "usr_1",
  displayName: "Rex Raphael",
  identifier: "rex@acme.com",
  strategy: "oidc:kineta",
  strategyLabel: "Kineta",
  at: new Date().toISOString(),
};

export const Default: Story = {
  args: { account: rex, onSelect: () => {} },
};

export const NoDisplayName: Story = {
  args: {
    account: { ...rex, displayName: undefined },
    onSelect: () => {},
  },
};

export const PasswordStrategy: Story = {
  args: {
    account: { ...rex, strategy: "password", strategyLabel: undefined },
    onSelect: () => {},
  },
};

export const NotDismissible: Story = {
  args: { account: rex, onSelect: () => {}, dismissible: false },
};

export const LongIdentifier: Story = {
  args: {
    account: {
      ...rex,
      displayName: "Alexandra Constantina Rivera-Montgomery",
      identifier: "alexandra.constantina.rivera@a-very-long-company-name.example",
    },
    onSelect: () => {},
  },
};

export const NothingRemembered: Story = {
  name: "Nothing remembered (renders nothing)",
  args: { account: null },
};
