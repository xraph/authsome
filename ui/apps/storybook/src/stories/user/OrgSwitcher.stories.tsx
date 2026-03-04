import React from "react";
import type { Meta, StoryObj } from "@storybook/react";
import { OrgSwitcher } from "@authsome/ui-components";
import { MockAuthProvider } from "../../mocks/auth-provider";

const meta: Meta<typeof OrgSwitcher> = {
  title: "User/Org Switcher",
  component: OrgSwitcher,
  tags: ["autodocs"],
  parameters: { layout: "centered" },
  decorators: [
    (Story) => (
      <MockAuthProvider initialState="authenticated">
        <Story />
      </MockAuthProvider>
    ),
  ],
};
export default meta;
type Story = StoryObj<typeof OrgSwitcher>;

export const Default: Story = {
  args: {
    onOrgChange: (orgId: string) => console.log("Org changed:", orgId),
  },
};

export const WithActiveOrg: Story = {
  args: {
    activeOrgId: "org_1",
    onOrgChange: (orgId: string) => console.log("Org changed:", orgId),
  },
};
