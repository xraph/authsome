import type { Meta, StoryObj } from "@storybook/react";
import { useState } from "react";
import { PhoneInput } from "@authsome/ui-components";

const meta: Meta<typeof PhoneInput> = {
  title: "Primitives/Phone Input",
  component: PhoneInput,
  parameters: { layout: "centered" },
  decorators: [
    (Story) => (
      <div className="w-[360px]">
        <Story />
      </div>
    ),
  ],
};
export default meta;
type Story = StoryObj<typeof PhoneInput>;

export const Default: Story = {
  render: (args) => {
    const [value, setValue] = useState("");
    const [valid, setValid] = useState(false);
    return (
      <div className="flex flex-col gap-3">
        <PhoneInput
          {...args}
          value={value}
          onChange={(v, isValid) => {
            setValue(v);
            setValid(isValid);
          }}
        />
        <p className="text-[13px] text-muted-foreground">
          E.164: <code>{value || "(empty)"}</code>
        </p>
        <p className="text-[13px] text-muted-foreground">
          valid: <code>{String(valid)}</code>
        </p>
      </div>
    );
  },
};

export const Portugal: Story = {
  ...Default,
  args: { defaultCountry: "PT" },
};

export const ShortList: Story = {
  ...Default,
  args: { defaultCountry: "PT", countries: ["PT", "ES", "GB", "US", "NG"] },
};

export const Disabled: Story = {
  args: { defaultCountry: "GB", disabled: true },
};
