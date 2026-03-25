"use client";

import * as React from "react";
import { SignInForm } from "./sign-in-form";
import { ForgotPasswordForm } from "./forgot-password-form";
import { ResetPasswordForm } from "./reset-password-form";
import { EmailVerificationForm } from "./email-verification-form";
import type { AuthCardAlign, AuthCardVariant } from "./auth-card";
import type { SocialButtonLayout, SocialProvider } from "./social-buttons";

export interface SignInProps {
  /** The base path for the sign-in page (default: "/sign-in"). */
  path?: string;
  /** URL to the sign-up page. */
  signUpUrl?: string;
  /** Callback invoked after a successful sign-in (any method). */
  onSuccess?: () => void;
  /** Social/OAuth providers to display. Auto-derived from config when omitted. */
  socialProviders?: SocialProvider[];
  /** Override social login click handler. */
  onSocialLogin?: (providerId: string) => void;
  /** Layout mode for social login buttons. */
  socialLayout?: SocialButtonLayout;
  /** Show passkey sign-in button. */
  showPasskey?: boolean;
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
 * Self-routing sign-in component (Clerk-style).
 *
 * Renders the appropriate form based on the current URL path:
 * - `/sign-in` → Sign-in form (email/password + social)
 * - `/sign-in/forgot-password` → Forgot password form
 * - `/sign-in/reset-password?token=...` → Reset password form
 * - `/sign-in/verify-email` → Email verification form
 *
 * Usage with a Next.js catch-all route `[[...sign-in]]/page.tsx`:
 * ```tsx
 * import { SignIn } from "@authsome/ui-components";
 * export default function Page() {
 *   return <SignIn />;
 * }
 * ```
 */
export function SignIn({
  path = "/sign-in",
  signUpUrl = "/sign-up",
  onSuccess,
  socialProviders,
  onSocialLogin,
  socialLayout,
  showPasskey,
  logo,
  align,
  variant,
  className,
}: SignInProps) {
  const subPath = useSubPath(path);

  const handleSuccess = React.useCallback(() => {
    if (onSuccess) {
      onSuccess();
      return;
    }
    const params = new URLSearchParams(window.location.search);
    const redirectTo = params.get("redirect");
    window.location.href = redirectTo || "/";
  }, [onSuccess]);

  if (subPath === "forgot-password") {
    return (
      <ForgotPasswordForm
        signInUrl={path}
        logo={logo}
        align={align}
        variant={variant}
        className={className}
      />
    );
  }

  if (subPath === "reset-password") {
    const token =
      typeof window !== "undefined"
        ? new URLSearchParams(window.location.search).get("token") ?? ""
        : "";

    return (
      <ResetPasswordForm
        token={token}
        onSuccess={() => {
          window.location.href = path;
        }}
        logo={logo}
        className={className}
      />
    );
  }

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

  // Default: sign-in form
  return (
    <SignInForm
      onSuccess={handleSuccess}
      signUpUrl={signUpUrl}
      forgotPasswordUrl={`${path}/forgot-password`}
      socialProviders={socialProviders}
      onSocialLogin={onSocialLogin}
      socialLayout={socialLayout}
      showPasskey={showPasskey}
      logo={logo}
      align={align}
      variant={variant}
      className={className}
    />
  );
}

/**
 * Extracts the sub-path segment after the base path from the current URL.
 * E.g. for base="/sign-in" and URL="/sign-in/forgot-password", returns "forgot-password".
 */
function useSubPath(basePath: string): string | undefined {
  const [subPath, setSubPath] = React.useState<string | undefined>(() => {
    if (typeof window === "undefined") return undefined;
    return extractSubPath(window.location.pathname, basePath);
  });

  React.useEffect(() => {
    setSubPath(extractSubPath(window.location.pathname, basePath));

    const handler = () => {
      setSubPath(extractSubPath(window.location.pathname, basePath));
    };
    window.addEventListener("popstate", handler);
    return () => window.removeEventListener("popstate", handler);
  }, [basePath]);

  return subPath;
}

function extractSubPath(pathname: string, basePath: string): string | undefined {
  const normalized = basePath.replace(/\/+$/, "");
  if (!pathname.startsWith(normalized)) return undefined;
  const rest = pathname.slice(normalized.length).replace(/^\/+/, "");
  return rest || undefined;
}
