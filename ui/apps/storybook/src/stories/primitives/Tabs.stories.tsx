import React from "react";
import type { Meta, StoryObj } from "@storybook/react";
import {
  Tabs,
  TabsList,
  TabsTrigger,
  TabsContent,
} from "@authsome/ui-components";

const meta: Meta<typeof Tabs> = {
  title: "Primitives/Tabs",
  component: Tabs,
  tags: ["autodocs"],
  parameters: { layout: "centered" },
};
export default meta;
type Story = StoryObj<typeof Tabs>;

export const Default: Story = {
  render: () => (
    <Tabs defaultValue="account" className="w-[400px]">
      <TabsList>
        <TabsTrigger value="account">Account</TabsTrigger>
        <TabsTrigger value="security">Security</TabsTrigger>
        <TabsTrigger value="notifications">Notifications</TabsTrigger>
      </TabsList>
      <TabsContent value="account" className="p-4">
        <p className="text-sm text-muted-foreground">
          Manage your account settings and preferences.
        </p>
      </TabsContent>
      <TabsContent value="security" className="p-4">
        <p className="text-sm text-muted-foreground">
          Configure your security settings, including password and two-factor
          authentication.
        </p>
      </TabsContent>
      <TabsContent value="notifications" className="p-4">
        <p className="text-sm text-muted-foreground">
          Choose which notifications you want to receive.
        </p>
      </TabsContent>
    </Tabs>
  ),
};
