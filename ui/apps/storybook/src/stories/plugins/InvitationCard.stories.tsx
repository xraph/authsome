import type { Meta, StoryObj } from "@storybook/react";
import { InvitationCard } from "@authsome/plugin-organization/react";
import { Button } from "@authsome/ui-components";
import { MockAuthProvider } from "../../mocks/auth-provider";

const preview = {
  org_id: "org_1",
  org_name: "Revenue Ops",
  org_slug: "revenue-ops",
  inviter_name: "Rex Raphael",
  role: "builder",
  email_masked: "so***@acme.com",
  expires_at: new Date(Date.now() + 6 * 86400000).toISOString(),
};

const stub = {
  getInvitation: async () => preview,
  acceptInvitation: async () => ({
    id: "m1",
    org_id: "org_1",
    user_id: "u1",
    role: "builder",
    created_at: "",
    updated_at: "",
  }),
  declineInvitation: async () => ({ status: "declined" }),
  listInvitations: async () => ({ invitations: [] }),
  createInvitation: async () => ({}) as never,
};

const wrap = (plugins: Record<string, unknown>) =>
  function Decorator(Story: () => JSX.Element) {
    return (
      <MockAuthProvider initialState="authenticated" plugins={plugins}>
        <Story />
      </MockAuthProvider>
    );
  };

const meta: Meta<typeof InvitationCard> = {
  title: "Plugins/Invitation Card",
  component: InvitationCard,
  parameters: { layout: "centered" },
  decorators: [wrap({ organization: stub })],
};
export default meta;
type Story = StoryObj<typeof InvitationCard>;

export const SignedIn: Story = {
  args: {
    token: "inv_1",
    isSignedIn: true,
    roleLabels: { builder: "Builder" },
  },
};

export const SignedOut: Story = {
  name: "Signed out (sign-in slot)",
  args: {
    token: "inv_1",
    isSignedIn: false,
    roleLabels: { builder: "Builder" },
    signInSlot: (
      <div className="grid gap-2">
        <Button className="w-full">Continue with Kineta</Button>
        <Button variant="outline" className="w-full">
          Sign in with email
        </Button>
      </div>
    ),
  },
};

export const Expired: Story = {
  args: { token: "inv_1", isSignedIn: true },
  decorators: [
    wrap({
      organization: {
        ...stub,
        getInvitation: async () => ({
          ...preview,
          expires_at: new Date(Date.now() - 86400000).toISOString(),
        }),
      },
    }),
  ],
};

export const NotValid: Story = {
  args: { token: "bad", isSignedIn: true },
  decorators: [
    wrap({
      organization: {
        ...stub,
        getInvitation: async () => {
          throw new Error("not found");
        },
      },
    }),
  ],
};
