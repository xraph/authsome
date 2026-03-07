"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import { ForgotPasswordForm } from "@authsome/ui-components";

export interface ForgotPasswordPageProps {
  signInUrl?: string;
  afterSubmitUrl?: string;
  logo?: React.ReactNode;
}

export function ForgotPasswordPage({
  signInUrl = "/sign-in",
  afterSubmitUrl,
  logo,
}: ForgotPasswordPageProps) {
  const router = useRouter();

  return (
    <div className="flex min-h-screen items-center justify-center bg-muted/40 p-4">
      <ForgotPasswordForm
        signInUrl={signInUrl}
        onSuccess={() => {
          if (afterSubmitUrl) {
            router.push(afterSubmitUrl);
          }
        }}
        logo={logo}
      />
    </div>
  );
}
