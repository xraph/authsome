"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import {
  SignInForm,
  type SocialButtonLayout,
  type SocialProvider,
} from "@authsome/ui-components";
import { useAuth } from "@authsome/ui-react";

export interface SignInPageProps {
  signUpUrl?: string;
  forgotPasswordUrl?: string;
  afterSignInUrl?: string;
  socialProviders?: SocialProvider[];
  socialLayout?: SocialButtonLayout;
  onSocialLogin?: (providerId: string) => void;
  showPasskey?: boolean;
  logo?: React.ReactNode;
}

export function SignInPage({
  signUpUrl = "/sign-up",
  forgotPasswordUrl = "/forgot-password",
  afterSignInUrl = "/",
  socialProviders,
  socialLayout,
  onSocialLogin,
  showPasskey,
  logo,
}: SignInPageProps) {
  const router = useRouter();
  const { isAuthenticated, isLoading } = useAuth();

  React.useEffect(() => {
    if (isAuthenticated) {
      router.push(afterSignInUrl);
    }
  }, [isAuthenticated, router, afterSignInUrl]);

  if (isLoading || isAuthenticated) {
    return null;
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-muted/40 p-4">
      <SignInForm
        signUpUrl={signUpUrl}
        forgotPasswordUrl={forgotPasswordUrl}
        socialProviders={socialProviders}
        socialLayout={socialLayout}
        onSocialLogin={onSocialLogin}
        showPasskey={showPasskey}
        onSuccess={() => router.push(afterSignInUrl)}
        logo={logo}
      />
    </div>
  );
}
