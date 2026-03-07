"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import {
  SignUpForm,
  type SocialButtonLayout,
  type SocialProvider,
} from "@authsome/ui-components";

export interface SignUpPageProps {
  signInUrl?: string;
  afterSignUpUrl?: string;
  socialProviders?: SocialProvider[];
  socialLayout?: SocialButtonLayout;
  onSocialLogin?: (providerId: string) => void;
  logo?: React.ReactNode;
}

export function SignUpPage({
  signInUrl = "/sign-in",
  afterSignUpUrl = "/",
  socialProviders,
  socialLayout,
  onSocialLogin,
  logo,
}: SignUpPageProps) {
  const router = useRouter();

  return (
    <div className="flex min-h-screen items-center justify-center bg-muted/40 p-4">
      <SignUpForm
        signInUrl={signInUrl}
        socialProviders={socialProviders}
        socialLayout={socialLayout}
        onSocialLogin={onSocialLogin}
        onSuccess={() => router.push(afterSignUpUrl)}
        logo={logo}
      />
    </div>
  );
}
