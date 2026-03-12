"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import {
  SignUpForm,
  type SocialButtonLayout,
  type SocialProvider,
} from "@authsome/ui-components";
import { useAuth } from "@authsome/ui-react";

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
  const { isAuthenticated, isLoading } = useAuth();

  React.useEffect(() => {
    if (isAuthenticated) {
      router.push(afterSignUpUrl);
    }
  }, [isAuthenticated, router, afterSignUpUrl]);

  if (isLoading || isAuthenticated) {
    return null;
  }

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
