"use client";

import * as React from "react";
import { useState } from "react";
import { useAuth, useClientConfig } from "@authsome/ui-react";
import { cn } from "../lib/utils";
import { Button } from "../primitives/button";
import { Input } from "../primitives/input";
import { Label } from "../primitives/label";
import { AuthCard, type AuthCardAlign, type AuthCardVariant } from "./auth-card";
import { ErrorDisplay } from "./error-display";
import { LoadingSpinner } from "./loading-spinner";
import { PasswordInput } from "./password-input";
import { PasskeyLoginButton } from "./passkey-login-button";
import {
  SocialButtons,
  OrDivider,
  type SocialProvider,
  type SocialButtonLayout,
} from "./social-buttons";
import { handleSocialLogin } from "../lib/social-login";
import { ArrowLeft } from "lucide-react";

export interface SignInFormComponentProps {
  /** Callback invoked after a successful sign-in. */
  onSuccess?: () => void;
  /** URL to the sign-up page. Renders a "Don't have an account?" footer link. */
  signUpUrl?: string;
  /** URL to the forgot-password page. Renders a "Forgot password?" link. */
  forgotPasswordUrl?: string;
  /** Social/OAuth providers to display below the form. */
  socialProviders?: SocialProvider[];
  /** Callback when a social provider button is clicked. Overrides the built-in popup flow. */
  onSocialLogin?: (providerId: string) => void;
  /** Layout mode for social login buttons. */
  socialLayout?: SocialButtonLayout;
  /** Show passkey sign-in button. */
  showPasskey?: boolean;
  /** Callback after successful passkey sign-in. */
  onPasskeySuccess?: () => void;
  /** Optional logo element rendered above the title. */
  logo?: React.ReactNode;
  /** Title and description alignment. */
  align?: AuthCardAlign;
  /** Card visual style. */
  variant?: AuthCardVariant;
  /** Additional CSS class names. */
  className?: string;
}

/**
 * A fully styled sign-in form with Clerk-style UX:
 *
 * - **Social-first**: When social providers are available, they appear at the top.
 * - **Multi-step**: User enters email, clicks "Continue", then enters password.
 * - **Auto-configuration**: When `publishableKey` is set on `AuthProvider`, the form
 *   auto-derives social providers and passkey support from the backend client config.
 * - Explicit props always take precedence over auto-discovered values.
 */
export function SignInForm({
  onSuccess,
  signUpUrl,
  forgotPasswordUrl,
  socialProviders: socialProvidersProp,
  onSocialLogin: onSocialLoginProp,
  socialLayout,
  showPasskey: showPasskeyProp,
  onPasskeySuccess,
  logo,
  align,
  variant,
  className,
}: SignInFormComponentProps) {
  const { signIn, client } = useAuth();
  const { config } = useClientConfig();

  // Auto-derive social providers from client config when not explicitly provided.
  const socialProviders =
    socialProvidersProp ??
    (config?.social?.enabled && config.social.providers.length > 0
      ? config.social.providers.map((p) => ({ id: p.id, name: p.name }))
      : undefined);

  // Auto-derive passkey support from client config when not explicitly provided.
  const showPasskey = showPasskeyProp ?? config?.passkey?.enabled ?? false;

  // Auto-derive password support from client config (default: true).
  const showPassword = config?.password?.enabled ?? true;

  // Default social login: popup-based OAuth flow via startOAuth API.
  // Falls back to full-page redirect when popups are blocked.
  const onSocialLogin =
    onSocialLoginProp ??
    (socialProviders && socialProviders.length > 0
      ? (providerId: string) =>
          handleSocialLogin(client, providerId, () => {
            onSuccess?.();
            // Fallback: reload the page so the server-side middleware
            // picks up the httpOnly session cookie set during the callback.
            window.location.reload();
          })
      : undefined);

  const hasSocial =
    socialProviders && socialProviders.length > 0 && onSocialLogin;

  const [step, setStep] = useState<"email" | "password">("email");
  const [email, setEmail] = useState("");
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

    setStep("password");
  };

  const handleSignIn = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setError(null);
    setIsSubmitting(true);

    try {
      await signIn(email, password);
      onSuccess?.();
    } catch (err) {
      setError(
        err instanceof Error
          ? err.message
          : "Sign in failed. Please try again.",
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

  const footer = signUpUrl ? (
    <p className="text-[13px] text-muted-foreground">
      Don&apos;t have an account?{" "}
      <a
        href={signUpUrl}
        className="font-medium text-foreground underline-offset-4 hover:underline"
      >
        Sign up
      </a>
    </p>
  ) : undefined;

  /* ── Password disabled: show only alternative methods ── */

  if (!showPassword) {
    const hasAnyMethod = hasSocial || showPasskey;
    return (
      <AuthCard
        title="Sign in"
        description="Welcome back. Sign in to your account."
        logo={logo}
        footer={footer}
        align={align}
        variant={variant}
        className={cn(className)}
      >
        <div className="grid gap-4">
          {hasSocial && (
            <SocialButtons
              providers={socialProviders!}
              onProviderClick={onSocialLogin!}
              isLoading={isSubmitting}
              layout={socialLayout}
              showDivider={false}
            />
          )}

          {hasSocial && showPasskey && <OrDivider />}

          {showPasskey && (
            <PasskeyLoginButton
              onSuccess={onPasskeySuccess ?? onSuccess}
              variant="outline"
              className="w-full"
            />
          )}

          {!hasAnyMethod && (
            <p className="text-sm text-center text-muted-foreground py-4">
              No sign-in methods are currently available. Please contact your
              administrator.
            </p>
          )}
        </div>
      </AuthCard>
    );
  }

  /* ── Step 1: Email ──────────────────────────────────── */

  if (step === "email") {
    return (
      <AuthCard
        title="Sign in"
        description="Welcome back. Sign in to your account."
        logo={logo}
        footer={footer}
        align={align}
        variant={variant}
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
              <Label htmlFor="signin-email" className="text-[13px]">
                Email address
              </Label>
              <Input
                id="signin-email"
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

          {showPasskey && (
            <>
              <OrDivider className="my-2" />
              <PasskeyLoginButton
                onSuccess={onPasskeySuccess ?? onSuccess}
                variant="outline"
                className="w-full"
              />
            </>
          )}
        </div>
      </AuthCard>
    );
  }

  /* ── Step 2: Password ───────────────────────────────── */

  return (
    <AuthCard
      title="Enter your password"
      description={email}
      logo={logo}
      footer={footer}
      align={align}
      variant={variant}
      className={cn(className)}
    >
      <div className="grid gap-4">
        <button
          type="button"
          onClick={goBack}
          className="inline-flex items-center gap-1.5 text-[13px] text-muted-foreground transition-colors hover:text-foreground"
        >
          <ArrowLeft className="h-3.5 w-3.5" />
          Use a different method
        </button>

        <form onSubmit={handleSignIn} className="grid gap-3">
          <ErrorDisplay error={error} />

          <div className="grid gap-1.5">
            <div className="flex items-center justify-between">
              <Label htmlFor="signin-password" className="text-[13px]">
                Password
              </Label>
              {forgotPasswordUrl && (
                <a
                  href={forgotPasswordUrl}
                  className="text-[13px] text-muted-foreground transition-colors hover:text-foreground"
                >
                  Forgot password?
                </a>
              )}
            </div>
            <PasswordInput
              id="signin-password"
              placeholder="Enter your password"
              autoComplete="current-password"
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
            Continue
          </Button>
        </form>
      </div>
    </AuthCard>
  );
}
