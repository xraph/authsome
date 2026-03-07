"use client";

import * as React from "react";
import { useState } from "react";
import { useAuth } from "@authsome/ui-react";
import { cn } from "../lib/utils";
import { Button } from "../primitives/button";
import { Input } from "../primitives/input";
import { Label } from "../primitives/label";
import { AuthCard } from "./auth-card";
import { ErrorDisplay } from "./error-display";
import { LoadingSpinner } from "./loading-spinner";
import { MailCheck } from "lucide-react";

export interface ForgotPasswordFormProps {
  /** Callback invoked after the reset link has been sent successfully. */
  onSuccess?: () => void;
  /** URL to the sign-in page. Renders a "Remember your password?" footer link. */
  signInUrl?: string;
  /** Optional logo element rendered above the title. */
  logo?: React.ReactNode;
  /** Additional CSS class names. */
  className?: string;
}

/**
 * A styled forgot-password form with Clerk-style polish.
 *
 * Sends a password-reset link to the provided email address.
 * On success, displays a confirmation message with an icon.
 */
export function ForgotPasswordForm({
  onSuccess,
  signInUrl,
  logo,
  className,
}: ForgotPasswordFormProps) {
  const { client } = useAuth();

  const [email, setEmail] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [isSuccess, setIsSuccess] = useState(false);

  const handleSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setError(null);
    setIsSubmitting(true);

    try {
      await client.forgotPassword({ email });
      setIsSuccess(true);
      onSuccess?.();
    } catch (err) {
      setError(
        err instanceof Error
          ? err.message
          : "Failed to send reset link. Please try again.",
      );
    } finally {
      setIsSubmitting(false);
    }
  };

  const footer = signInUrl ? (
    <p className="text-[13px] text-muted-foreground">
      Remember your password?{" "}
      <a
        href={signInUrl}
        className="font-medium text-foreground underline-offset-4 hover:underline"
      >
        Sign in
      </a>
    </p>
  ) : undefined;

  return (
    <AuthCard
      title="Forgot password"
      description="Enter your email to receive a reset link."
      logo={logo}
      footer={footer}
      className={cn(className)}
    >
      {isSuccess ? (
        <div className="grid gap-3 text-center">
          <div className="mx-auto flex h-10 w-10 items-center justify-center rounded-full bg-primary/10">
            <MailCheck className="h-5 w-5 text-primary" />
          </div>
          <p className="text-[13px] text-muted-foreground">
            Check your email for a reset link. If you don&apos;t see it, check
            your spam folder.
          </p>
        </div>
      ) : (
        <form onSubmit={handleSubmit} className="grid gap-3">
          <ErrorDisplay error={error} />

          <div className="grid gap-1.5">
            <Label htmlFor="forgot-email" className="text-[13px]">
              Email address
            </Label>
            <Input
              id="forgot-email"
              type="email"
              placeholder="name@example.com"
              autoComplete="email"
              required
              disabled={isSubmitting}
              value={email}
              onChange={(e) => setEmail(e.target.value)}
            />
          </div>

          <Button type="submit" className="w-full" disabled={isSubmitting}>
            {isSubmitting && <LoadingSpinner size="sm" className="mr-2" />}
            Send reset link
          </Button>
        </form>
      )}
    </AuthCard>
  );
}
