import type { Meta, StoryObj } from "@storybook/react";
import { OAuthConsentScreen } from "@authsome/plugin-oauth2/react";
import { MockAuthProvider } from "../../mocks/auth-provider";

const stub = {
  getConsentRequest: async () => ({
    cid: "cid_1",
    client_id: "cli_1",
    client_name: "Kineta",
    scopes: ["openid", "profile", "email", "offline_access"],
    csrf: "csrf_1",
    action: "/v1/oauth/consent",
  }),
  decide: async () => ({ redirect_url: "#decided" }),
  listGrants: async () => ({ grants: [] }),
  revokeGrant: async () => undefined,
};

const meta: Meta<typeof OAuthConsentScreen> = {
  title: "Plugins/OAuth Consent Screen",
  component: OAuthConsentScreen,
  parameters: { layout: "centered" },
  decorators: [
    (Story) => (
      <MockAuthProvider initialState="authenticated" plugins={{ oauth2provider: stub }}>
        <Story />
      </MockAuthProvider>
    ),
  ],
};
export default meta;
type Story = StoryObj<typeof OAuthConsentScreen>;

export const Default: Story = {
  args: { cid: "cid_1", onDecided: () => {} },
};

export const CustomScope: Story = {
  args: {
    cid: "cid_1",
    onDecided: () => {},
    scopeLabels: { "custom:widgets": "Manage your widgets" },
  },
  decorators: [
    (Story) => (
      <MockAuthProvider
        initialState="authenticated"
        plugins={{
          oauth2provider: {
            ...stub,
            getConsentRequest: async () => ({
              cid: "cid_1",
              client_id: "cli_1",
              client_name: "Widget Co",
              scopes: ["openid", "custom:widgets", "unmapped:thing"],
              csrf: "csrf_1",
              action: "/v1/oauth/consent",
            }),
          },
        }}
      >
        <Story />
      </MockAuthProvider>
    ),
  ],
};

export const NotValid: Story = {
  args: { cid: "cid_bad", onDecided: () => {} },
  decorators: [
    (Story) => (
      <MockAuthProvider
        initialState="authenticated"
        plugins={{
          oauth2provider: {
            ...stub,
            getConsentRequest: async () => {
              throw new Error("expired");
            },
          },
        }}
      >
        <Story />
      </MockAuthProvider>
    ),
  ],
};
