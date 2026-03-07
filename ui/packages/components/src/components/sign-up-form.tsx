"use client";

import * as React from "react";
import { useState } from "react";
import { useAuth, useClientConfig } from "@authsome/ui-react";
import { cn } from "../lib/utils";
import { Button } from "../primitives/button";
import { Input } from "../primitives/input";
import { Label } from "../primitives/label";
import { AuthCard } from "./auth-card";
import { ErrorDisplay } from "./error-display";
import { LoadingSpinner } from "./loading-spinner";
import { PasswordInput } from "./password-input";
import {
  SocialButtons,
  OrDivider,
  type SocialProvider,
  type SocialButtonLayout,
} from "./social-buttons";
import { ArrowLeft } from "lucide-react";

export interface SignUpFormComponentProps {
  /** Callback invoked after a successful sign-up. */
  onSuccess?: () => void;
  /** URL to the sign-in page. Renders an "Already have an account?" footer link. */
  signInUrl?: string;
  /** Social/OAuth providers to display below the form. */
  socialProviders?: SocialProvider[];
  /** Callback when a social provider button is clicked. */
  onSocialLogin?: (providerId: string) => void;
  /** Layout mode for social login buttons. */
  socialLayout?: SocialButtonLayout;
  /** Optional logo element rendered above the title. */
  logo?: React.ReactNode;
  /** Additional CSS class names. */
  className?: string;
}

/**
 * Default social login handler: redirects to the backend OAuth initiation endpoint.
 */
function defaultSocialLogin(providerId: string, baseURL: string) {
  const url = `${baseURL}/v1/auth/social/${providerId}`;
  window.location.href = url;
}

/**
 * A fully styled sign-up form with Clerk-style UX:
 *
 * - **Social-first**: When social providers are available, they appear at the top.
 * - **Multi-step**: User enters email first, clicks "Continue", then enters name & password.
 * - **Auto-configuration**: When `publishableKey` is set on `AuthProvider`, the form
 *   auto-derives social providers from the backend client config.
 * - Explicit props always take precedence over auto-discovered values.
 */
export function SignUpForm({
  onSuccess,
  signInUrl,
  socialProviders: socialProvidersProp,
  onSocialLogin: onSocialLoginProp,
  socialLayout,
  logo,
  className,
}: SignUpFormComponentProps) {
  const { signUp, client } = useAuth();
  const { config } = useClientConfig();

  // Auto-derive social providers from client config when not explicitly provided.
  const socialProviders =
    socialProvidersProp ??
    (config?.social?.enabled && config.social.providers.length > 0
      ? config.social.providers.map((p) => ({ id: p.id, name: p.name }))
      : undefined);

  // Default social login handler: full-page redirect to OAuth endpoint.
  const baseURL = (client as any).config?.baseURL ?? "";
  const onSocialLogin =
    onSocialLoginProp ??
    (socialProviders && socialProviders.length > 0
      ? (providerId: string) => defaultSocialLogin(providerId, baseURL)
      : undefined);

  const hasSocial =
    socialProviders && socialProviders.length > 0 && onSocialLogin;

  const [step, setStep] = useState<"email" | "details">("email");
  const [email, setEmail] = useState("");
  const [name, setName] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  const handleEmailContinue = (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setError(null);

    if (!email.trim()) {
      setError("Please enter your email address.");
      return;
    }

    setStep("details");
  };

  const handleSignUp = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setError(null);
    setIsSubmitting(true);

    try {
      await signUp(email, password, name || undefined);
      onSuccess?.();
    } catch (err) {
      setError(
        err instanceof Error
          ? err.message
          : "Sign up failed. Please try again.",
      );
    } finally {
      setIsSubmitting(false);
    }
  };

  const goBack = () => {
    setStep("email");
    setPassword("");
    setError(null);
  };

  const footer = signInUrl ? (
    <p className="text-[13px] text-muted-foreground">
      Already have an account?{" "}
      <a
        href={signInUrl}
        className="font-medium text-foreground underline-offset-4 hover:underline"
      >
        Sign in
      </a>
    </p>
  ) : undefined;

  /* -- Step 1: Email ---------------------------------------- */

  if (step === "email") {
    return (
      <AuthCard
        title="Create an account"
        description="Enter your email to get started."
        logo={logo}
        footer={footer}
        className={cn(className)}
      >
        <div className="grid gap-4">
          {/* Social buttons first (Clerk-style) */}
          {hasSocial && (
            <SocialButtons
              providers={socialProviders!}
              onProviderClick={onSocialLogin!}
              isLoading={isSubmitting}
              layout={socialLayout}
              showDivider={false}
            />
          )}

          {hasSocial && <OrDivider />}

          <form onSubmit={handleEmailContinue} className="grid gap-3">
            <ErrorDisplay error={error} />

            <div className="grid gap-1.5">
              <Label htmlFor="signup-email" className="text-[13px]">
                Email address
              </Label>
              <Input
                id="signup-email"
                type="email"
                placeholder="name@example.com"
                autoComplete="email"
                required
                disabled={isSubmitting}
                value={email}
                onChange={(e) => setEmail(e.target.value)}
              />
            </div>

            <Button
              type="submit"
              className="w-full"
              disabled={isSubmitting}
            >
              Continue
            </Button>
          </form>
        </div>
      </AuthCard>
    );
  }

  /* -- Step 2: Name & Password ------------------------------ */

  return (
    <AuthCard
      title="Complete your account"
      description={email}
      logo={logo}
      footer={footer}
      className={cn(className)}
    >
      <div className="grid gap-4">
        <button
          type="button"
          onClick={goBack}
          className="inline-flex items-center gap-1.5 text-[13px] text-muted-foreground transition-colors hover:text-foreground"
        >
          <ArrowLeft className="h-3.5 w-3.5" />
          Use a different email
        </button>

        <form onSubmit={handleSignUp} className="grid gap-3">
          <ErrorDisplay error={error} />

          <div className="grid gap-1.5">
            <Label htmlFor="signup-name" className="text-[13px]">
              Name
              <span className="ml-1 text-muted-foreground">(optional)</span>
            </Label>
            <Input
              id="signup-name"
              type="text"
              placeholder="John Doe"
              autoComplete="name"
              disabled={isSubmitting}
              value={name}
              onChange={(e) => setName(e.target.value)}
            />
          </div>

          <div className="grid gap-1.5">
            <Label htmlFor="signup-password" className="text-[13px]">
              Password
            </Label>
            <PasswordInput
              id="signup-password"
              placeholder="Create a password"
              autoComplete="new-password"
              required
              autoFocus
              disabled={isSubmitting}
              value={password}
              onChange={(e) => setPassword(e.target.value)}
            />
          </div>

          <Button
            type="submit"
            className="w-full"
            disabled={isSubmitting}
          >
            {isSubmitting && <LoadingSpinner size="sm" className="mr-2" />}
            Create account
          </Button>
        </form>
      </div>
    </AuthCard>
  );
}
