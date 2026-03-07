"use client";

import * as React from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { ResetPasswordForm } from "@authsome/ui-components";

export interface ResetPasswordPageProps {
  signInUrl?: string;
  logo?: React.ReactNode;
}

export function ResetPasswordPage({
  signInUrl = "/sign-in",
  logo,
}: ResetPasswordPageProps) {
  const router = useRouter();
  const searchParams = useSearchParams();
  const token = searchParams.get("token");

  if (!token) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-muted/40 p-4">
        <div className="text-center">
          <h2 className="text-lg font-semibold">Invalid Reset Link</h2>
          <p className="mt-2 text-sm text-muted-foreground">
            This password reset link is invalid or has expired.
          </p>
          <a
            href={signInUrl}
            className="mt-4 inline-block text-sm text-primary underline"
          >
            Back to sign in
          </a>
        </div>
      </div>
    );
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-muted/40 p-4">
      <ResetPasswordForm
        token={token}
        onSuccess={() => router.push(signInUrl)}
        logo={logo}
      />
    </div>
  );
}
