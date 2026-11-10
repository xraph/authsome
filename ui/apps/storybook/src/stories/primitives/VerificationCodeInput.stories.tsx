import type { Meta, StoryObj } from "@storybook/react";
import { useState } from "react";
import { VerificationCodeInput } from "@authsome/ui-components";

const meta: Meta<typeof VerificationCodeInput> = {
  title: "Primitives/Verification Code",
  component: VerificationCodeInput,
  parameters: { layout: "centered" },
};
export default meta;
type Story = StoryObj<typeof VerificationCodeInput>;

export const Numeric: Story = {
  args: { length: 6, mode: "numeric" },
};

export const Alphanumeric: Story = {
  args: { length: 6, mode: "alphanumeric" },
};

export const Grouped: Story = {
  name: "Grouped (device code)",
  args: { length: 8, mode: "alphanumeric", groupSize: 4 },
};

export const FourDigit: Story = {
  args: { length: 4, mode: "numeric" },
};

export const Invalid: Story = {
  args: { length: 6, mode: "numeric", value: "483920", invalid: true },
};

export const Disabled: Story = {
  args: { length: 6, mode: "numeric", value: "4829", disabled: true },
};

export const Controlled: Story = {
  render: () => {
    const [code, setCode] = useState("");
    const [done, setDone] = useState<string | null>(null);
    return (
      <div className="flex flex-col items-center gap-4">
        <VerificationCodeInput
          length={6}
          mode="numeric"
          value={code}
          onChange={setCode}
          onComplete={setDone}
        />
        <p className="text-sm text-muted-foreground">
          value: <code>{code || "(empty)"}</code>
        </p>
        <p className="text-sm text-muted-foreground">
          onComplete fired with: <code>{done ?? "(not yet)"}</code>
        </p>
      </div>
    );
  },
};
