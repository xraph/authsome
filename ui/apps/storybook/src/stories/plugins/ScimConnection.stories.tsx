import type { Meta, StoryObj } from "@storybook/react";
import { ScimConnectionCard, ScimTokenList } from "@authsome/plugin-scim/react";
import { MockAuthProvider } from "../../mocks/auth-provider";

const config = {
  id: "cfg_1",
  app_id: "a1",
  org_id: "org_1",
  name: "Okta production",
  enabled: true,
  auto_create: true,
  auto_suspend: true,
  group_sync: false,
  default_role: "member",
  created_at: "2026-05-01T00:00:00Z",
  updated_at: "2026-10-01T00:00:00Z",
};

const tokens = [
  {
    id: "tok_1",
    config_id: "cfg_1",
    name: "Okta production",
    last_used_at: "2026-11-09T22:14:00Z",
    created_at: "2026-05-01T00:00:00Z",
  },
  {
    id: "tok_2",
    config_id: "cfg_1",
    name: "Okta sandbox",
    expires_at: "2026-06-01T00:00:00Z",
    created_at: "2026-01-01T00:00:00Z",
  },
];

const stub = {
  listConfigs: async () => ({ configs: [config] }),
  createConfig: async () => config,
  getConfig: async () => config,
  updateConfig: async () => config,
  deleteConfig: async () => ({ status: "deleted" }),
  listTokens: async () => ({ tokens }),
  createToken: async () => ({
    token: tokens[0],
    plaintext: "scim_live_8f2c4d9a1b6e3047a5c8d2f1e9b7a403",
    base_url: "/scim/v2",
  }),
  rotateToken: async () => ({
    token: tokens[0],
    plaintext: "scim_live_1a2b3c4d5e6f70819a2b3c4d5e6f7081",
    base_url: "/scim/v2",
  }),
  revokeToken: async () => ({ status: "revoked" }),
};

const meta: Meta<typeof ScimConnectionCard> = {
  title: "Plugins/SCIM Connection",
  component: ScimConnectionCard,
  parameters: { layout: "centered" },
  decorators: [
    (Story) => (
      <MockAuthProvider initialState="authenticated" plugins={{ scim: stub }}>
        <div className="w-[520px]">
          <Story />
        </div>
      </MockAuthProvider>
    ),
  ],
};
export default meta;
type Story = StoryObj<typeof ScimConnectionCard>;

export const Connected: Story = {
  args: { orgId: "org_1" },
};

export const WithTokens: Story = {
  args: {
    orgId: "org_1",
    children: <ScimTokenList configId="cfg_1" />,
  },
};

export const NothingConnected: Story = {
  args: { orgId: "org_1" },
  decorators: [
    (Story) => (
      <MockAuthProvider
        initialState="authenticated"
        plugins={{ scim: { ...stub, listConfigs: async () => ({ configs: [] }) } }}
      >
        <div className="w-[520px]">
          <Story />
        </div>
      </MockAuthProvider>
    ),
  ],
};
