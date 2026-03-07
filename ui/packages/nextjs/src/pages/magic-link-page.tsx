"use client";

import * as React from "react";
import { MagicLinkForm } from "@authsome/ui-components";

export interface MagicLinkPageProps {
  signInUrl?: string;
  logo?: React.ReactNode;
}

export function MagicLinkPage({
  signInUrl = "/sign-in",
  logo,
}: MagicLinkPageProps) {
  const [submitted, setSubmitted] = React.useState(false);

  if (submitted) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-muted/40 p-4">
        <div className="text-center">
          <h2 className="text-lg font-semibold">Check Your Email</h2>
          <p className="mt-2 text-sm text-muted-foreground">
            We&apos;ve sent a magic link to your email address. Click the link
            to sign in.
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
      <MagicLinkForm
        signInUrl={signInUrl}
        onSuccess={() => setSubmitted(true)}
        logo={logo}
      />
    </div>
  );
}
