"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import {
  UserProfileCard,
  ChangePasswordForm,
  Tabs,
  TabsList,
  TabsTrigger,
  TabsContent,
} from "@authsome/ui-components";
import { useAuth } from "@authsome/ui-react";

export interface UserProfilePageProps {
  signInUrl?: string;
  title?: string;
}

export function UserProfilePage({
  signInUrl = "/sign-in",
  title = "My Account",
}: UserProfilePageProps) {
  const router = useRouter();
  const { isAuthenticated } = useAuth();

  React.useEffect(() => {
    if (!isAuthenticated) {
      router.push(signInUrl);
    }
  }, [isAuthenticated, router, signInUrl]);

  if (!isAuthenticated) {
    return null;
  }

  return (
    <div className="mx-auto min-h-screen max-w-2xl p-4 pt-8">
      <h1 className="mb-6 text-2xl font-bold">{title}</h1>
      <Tabs defaultValue="profile">
        <TabsList>
          <TabsTrigger value="profile">Profile</TabsTrigger>
          <TabsTrigger value="security">Security</TabsTrigger>
        </TabsList>
        <TabsContent value="profile">
          <UserProfileCard />
        </TabsContent>
        <TabsContent value="security">
          <ChangePasswordForm />
        </TabsContent>
      </Tabs>
    </div>
  );
}
