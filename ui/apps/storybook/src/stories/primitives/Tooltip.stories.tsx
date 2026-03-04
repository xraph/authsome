import React from "react";
import type { Meta, StoryObj } from "@storybook/react";
import {
  TooltipProvider,
  Tooltip,
  TooltipTrigger,
  TooltipContent,
} from "@authsome/ui-components";

const meta: Meta<typeof Tooltip> = {
  title: "Primitives/Tooltip",
  component: Tooltip,
  tags: ["autodocs"],
  parameters: { layout: "centered" },
};
export default meta;
type Story = StoryObj<typeof Tooltip>;

export const Default: Story = {
  render: () => (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger asChild>
          <button className="rounded-md border px-4 py-2 text-sm">
            Hover me
          </button>
        </TooltipTrigger>
        <TooltipContent>
          <p>This is a helpful tooltip</p>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  ),
};
