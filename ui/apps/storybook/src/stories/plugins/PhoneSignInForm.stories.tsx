import type { Meta, StoryObj } from "@storybook/react";
import { PhoneSignInForm } from "@authsome/plugin-phone/react";
import { MockAuthProvider } from "../../mocks/auth-provider";

/**
 * The stub stands in for the server. `start` always succeeds; `verify`
 * accepts 482917 and rejects everything else, so the error state is
 * reachable without a backend.
 */
const phoneStub = {
  start: async () => {
    await new Promise((r) => setTimeout(r, 400));
    return { status: "sent", expires_in: 300 };
  },
  verify: async ({ code }: { code: string }) => {
    await new Promise((r) => setTimeout(r, 400));
    if (code !== "482917") throw new Error("That code is wrong or has expired.");
    return {
      session_token: "sess_mock",
      refresh_token: "ref_mock",
      expires_at: new Date(Date.now() + 3600_000).toISOString(),
      new_user: false,
      user: { id: "usr_1", email: "rex@acme.com" },
    };
  },
};

const meta: Meta<typeof PhoneSignInForm> = {
  title: "Plugins/Phone Sign In",
  component: PhoneSignInForm,
  parameters: { layout: "centered" },
  decorators: [
    (Story) => (
      <MockAuthProvider plugins={{ phone: phoneStub }}>
        <Story />
      </MockAuthProvider>
    ),
  ],
};
export default meta;
type Story = StoryObj<typeof PhoneSignInForm>;

export const Default: Story = {};

export const Borderless: Story = {
  args: { variant: "borderless" },
};

export const FourDigitCode: Story = {
  args: { codeLength: 4 },
};

export const WithFooter: Story = {
  args: {
    footer: <span className="text-[13px] text-muted-foreground">Use email instead</span>,
  },
};
