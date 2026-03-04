import React from "react";
import type { Meta, StoryObj } from "@storybook/react";
import {
  Dialog,
  DialogTrigger,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@authsome/ui-components";

const meta: Meta<typeof Dialog> = {
  title: "Primitives/Dialog",
  component: Dialog,
  tags: ["autodocs"],
  parameters: { layout: "centered" },
};
export default meta;
type Story = StoryObj<typeof Dialog>;

export const Default: Story = {
  render: () => (
    <Dialog>
      <DialogTrigger asChild>
        <button className="rounded-md bg-primary px-4 py-2 text-sm text-primary-foreground">
          Open Dialog
        </button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Are you sure?</DialogTitle>
          <DialogDescription>
            This action cannot be undone. This will permanently delete your
            account and remove your data from our servers.
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <button className="rounded-md border px-4 py-2 text-sm">
            Cancel
          </button>
          <button className="rounded-md bg-primary px-4 py-2 text-sm text-primary-foreground">
            Confirm
          </button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  ),
};
