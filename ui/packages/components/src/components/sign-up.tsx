"use client";

import * as React from "react";
import { useClientConfig } from "@authsome/ui-react";
import { safeRedirectTarget } from "@authsome/ui-core";
import { useSubPath } from "../lib/use-sub-path";
import { SignUpForm } from "./sign-up-form";
import { EmailVerificationForm } from "./email-verification-form";
import type { AuthCardAlign, AuthCardVariant } from "./auth-card";
import type { SocialButtonLayout, SocialProvider } from "./social-buttons";

export interface SignUpProps {
  /** The base path for the sign-up page (default: "/sign-up"). */
  path?: string;
  /** URL to the sign-in page. */
  signInUrl?: string;
  /** Callback invoked after a successful sign-up (any method). */
  onSuccess?: () => void;
  /** Social/OAuth providers to display. Auto-derived from config when omitted. */
  socialProviders?: SocialProvider[];
  /** Override social login click handler. */
  onSocialLogin?: (providerId: string) => void;
  /** Layout mode for social login buttons. */
  socialLayout?: SocialButtonLayout;
  /** Optional logo element. */
  logo?: React.ReactNode;
  /** Title and description alignment. */
  align?: AuthCardAlign;
  /** Card visual style. */
  variant?: AuthCardVariant;
  /** Additional CSS class names. */
  className?: string;
}

/**
 * Self-routing sign-up component (Clerk-style).
 *
 * Renders the appropriate form based on the current URL path:
 * - `/sign-up` → Sign-up form
 * - `/sign-up/verify-email` → Email verification form (auto-shown after signup
 *   when email verification is required by the backend config)
 *
 * Usage with a Next.js catch-all route `[[...sign-up]]/page.tsx`:
 * ```tsx
 * import { SignUp } from "@authsome/ui-components";
 * export default function Page() {
 *   return <SignUp />;
 * }
 * ```
 */
export function SignUp({
  path = "/sign-up",
  signInUrl = "/sign-in",
  onSuccess,
  socialProviders,
  onSocialLogin,
  socialLayout,
  logo,
  align,
  variant,
  className,
}: SignUpProps) {
  const subPath = useSubPath(path);
  const { config } = useClientConfig();
  const emailVerificationRequired =
    config?.email_verification?.enabled && config?.email_verification?.required;

  const handleSuccess = React.useCallback(() => {
    if (onSuccess) {
      onSuccess();
      return;
    }
    const params = new URLSearchParams(window.location.search);
    // See sign-in.tsx — `redirect` is attacker-controllable and must resolve
    // to this origin before we navigate a just-authenticated user to it.
    window.location.href = safeRedirectTarget(params.get("redirect"), "/", {
      currentOrigin: window.location.origin,
    });
  }, [onSuccess]);

  const handleSignUpSuccess = React.useCallback(() => {
    if (emailVerificationRequired) {
      // Redirect to verify-email sub-route after signup.
      window.location.href = `${path}/verify-email`;
      return;
    }
    handleSuccess();
  }, [emailVerificationRequired, path, handleSuccess]);

  if (subPath === "verify-email") {
    const email =
      typeof window !== "undefined"
        ? new URLSearchParams(window.location.search).get("email") ?? ""
        : "";

    return (
      <EmailVerificationForm
        email={email}
        onSuccess={handleSuccess}
        logo={logo}
        className={className}
      />
    );
  }

  return (
    <SignUpForm
      onSuccess={handleSignUpSuccess}
      signInUrl={signInUrl}
      forgotPasswordUrl={`${signInUrl}/forgot-password`}
      socialProviders={socialProviders}
      onSocialLogin={onSocialLogin}
      socialLayout={socialLayout}
      logo={logo}
      align={align}
      variant={variant}
      className={className}
    />
  );
}

