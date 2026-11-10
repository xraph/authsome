import type { Meta, StoryObj } from "@storybook/react";
import { ConsentList } from "@authsome/plugin-consent/react";
import { MockAuthProvider } from "../../mocks/auth-provider";

const consents = [
  {
    id: "c1",
    user_id: "u1",
    app_id: "a1",
    purpose: "marketing_email",
    version: "2026-01",
    granted: true,
    granted_at: "2026-09-02T10:00:00Z",
    ip_address: "81.193.4.22",
    created_at: "2026-09-02T10:00:00Z",
    updated_at: "2026-09-02T10:00:00Z",
  },
  {
    id: "c2",
    user_id: "u1",
    app_id: "a1",
    purpose: "product_analytics",
    version: "2025-11",
    granted: true,
    granted_at: "2025-11-14T09:30:00Z",
    revoked_at: "2026-04-01T08:00:00Z",
    ip_address: "81.193.4.22",
    created_at: "2025-11-14T09:30:00Z",
    updated_at: "2026-04-01T08:00:00Z",
  },
];

const stub = {
  list: async () => ({ consents }),
  grant: async () => consents[0],
  revoke: async () => ({ status: "revoked" }),
};

const meta: Meta<typeof ConsentList> = {
  title: "Plugins/Consent List",
  component: ConsentList,
  parameters: { layout: "centered" },
  decorators: [
    (Story) => (
      <MockAuthProvider initialState="authenticated" plugins={{ consent: stub }}>
        <div className="w-[460px]">
          <Story />
        </div>
      </MockAuthProvider>
    ),
  ],
};
export default meta;
type Story = StoryObj<typeof ConsentList>;

export const Default: Story = {
  args: {
    labels: {
      marketing_email: {
        title: "Product news by email",
        description: "Occasional updates about what we have shipped.",
      },
      product_analytics: {
        title: "Usage analytics",
        description: "Helps us see which features are worth keeping.",
      },
    },
  },
};

export const Empty: Story = {
  decorators: [
    (Story) => (
      <MockAuthProvider
        initialState="authenticated"
        plugins={{ consent: { ...stub, list: async () => ({ consents: [] }) } }}
      >
        <div className="w-[460px]">
          <Story />
        </div>
      </MockAuthProvider>
    ),
  ],
};
