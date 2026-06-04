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
import { ArrowLeft, MailCheck } from "lucide-react";
import { TurnstileWidget } from "./turnstile-widget";
import { AuthClientError } from "@authsome/ui-core";

export interface SignInFormComponentProps {
  /** Callback invoked after a successful sign-in. */
  onSuccess?: () => void;
  /** URL to the sign-up page. Renders a "Don't have an account?" footer link. */
  signUpUrl?: string;
  /** URL to the forgot-password page. Renders a "Forgot password?" link. */
  forgotPasswordUrl?: string;
  /**
   * URL to the dedicated email verification view. When set, sign-in attempts
   * that fail with `email_not_verified` navigate here (with `?email=`
   * appended) instead of swapping to the inline resend panel. Pair with the
   * `<SignIn>` composite, which exposes this view at
   * `${path}/verify-email`.
   */
  verifyEmailUrl?: string;
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
  verifyEmailUrl,
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
  const { signIn, client, resendVerification } = useAuth();
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

  const [step, setStep] = useState<"email" | "password" | "verify">("email");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [captchaToken, setCaptchaToken] = useState<string | null>(null);
  const [resendStatus, setResendStatus] = useState<"idle" | "sent" | "error">(
    "idle",
  );

  const captchaCfg = config?.captcha;
  const captchaEnabled =
    !!captchaCfg?.required &&
    captchaCfg.provider === "turnstile" &&
    !!captchaCfg.site_key;

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
      await signIn(
        email,
        password,
        captchaToken ? { captchaToken } : undefined,
      );
      onSuccess?.();
    } catch (err) {
      // Surface "email_not_verified" via a dedicated panel, not a generic
      // error message. When a verifyEmailUrl is provided, navigate the
      // browser to the dedicated OTP-based verification view (and kick off
      // a resend in the background so the user has a fresh code on
      // arrival). Otherwise fall back to the inline resend panel.
      if (
        err instanceof AuthClientError &&
        err.type === "email_not_verified"
      ) {
        if (verifyEmailUrl && typeof window !== "undefined") {
          void resendVerification(email).catch(() => {
            // Best-effort: the verify view also exposes a Resend button.
          });
          const sep = verifyEmailUrl.includes("?") ? "&" : "?";
          window.location.href =
            verifyEmailUrl + sep + "email=" + encodeURIComponent(email);
          return;
        }
        setStep("verify");
        return;
      }
      setError(
        err instanceof Error
          ? err.message
          : "Sign in failed. Please try again.",
      );
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleResend = async () => {
    setResendStatus("idle");
    try {
      await resendVerification(email);
      setResendStatus("sent");
    } catch {
      setResendStatus("error");
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

  /* ── Verify: account needs email verification ──────── */

  if (step === "verify") {
    return (
      <AuthCard
        title="Verify your email"
        description={`Your account at ${email} hasn't been verified yet.`}
        logo={logo}
        footer={footer}
        align={align}
        variant={variant}
        className={cn(className)}
      >
        <div className="grid gap-4">
          <div className="flex flex-col items-center gap-3 py-2 text-center">
            <div className="rounded-full bg-muted p-3">
              <MailCheck className="h-6 w-6 text-foreground" />
            </div>
            <p className="text-sm text-muted-foreground">
              Click the verification link we emailed you, then sign in again.
            </p>
          </div>
          <Button
            type="button"
            variant="outline"
            className="w-full"
            onClick={handleResend}
            disabled={resendStatus === "sent"}
          >
            {resendStatus === "sent"
              ? "Verification email sent"
              : "Resend verification email"}
          </Button>
          {resendStatus === "error" && (
            <ErrorDisplay error="Could not resend the email. Please try again." />
          )}
          <button
            type="button"
            onClick={() => {
              setStep("email");
              setPassword("");
              setError(null);
              setResendStatus("idle");
            }}
            className="inline-flex items-center justify-center gap-1.5 text-[13px] text-muted-foreground transition-colors hover:text-foreground"
          >
            <ArrowLeft className="h-3.5 w-3.5" />
            Use a different email
          </button>
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
                name="username"
                type="email"
                placeholder="name@example.com"
                autoComplete="username"
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
          {/* Keeps the email in the password form's DOM so password managers can
              save / prefill the email+password pair. Visually hidden (not
              type="hidden") because many managers skip type="hidden" inputs. */}
          <input
            type="text"
            name="username"
            autoComplete="username"
            value={email}
            readOnly
            tabIndex={-1}
            aria-hidden="true"
            className="sr-only"
          />
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
              name="password"
              placeholder="Enter your password"
              autoComplete="current-password"
              required
              autoFocus
              disabled={isSubmitting}
              value={password}
              onChange={(e) => setPassword(e.target.value)}
            />
          </div>

          {captchaEnabled && captchaCfg?.site_key && (
            <TurnstileWidget
              siteKey={captchaCfg.site_key}
              onToken={setCaptchaToken}
              onExpire={() => setCaptchaToken(null)}
              onError={() => setCaptchaToken(null)}
            />
          )}

          <Button
            type="submit"
            className="w-full"
            disabled={isSubmitting || (captchaEnabled && !captchaToken)}
          >
            {isSubmitting && <LoadingSpinner size="sm" className="mr-2" />}
            Continue
          </Button>
        </form>
      </div>
    </AuthCard>
  );
}
