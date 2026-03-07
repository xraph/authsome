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

export interface MagicLinkFormProps {
  /** Callback invoked after the magic link has been sent successfully. */
  onSuccess?: () => void;
  /** URL to the sign-in page. Renders a "Sign in with password?" footer link. */
  signInUrl?: string;
  /** Optional logo element rendered above the title. */
  logo?: React.ReactNode;
  /** Additional CSS class names. */
  className?: string;
}

/**
 * A styled magic-link form with Clerk-style polish.
 *
 * Sends a passwordless sign-in link to the provided email address.
 * On success, displays a confirmation message with an icon.
 */
export function MagicLinkForm({
  onSuccess,
  signInUrl,
  logo,
  className,
}: MagicLinkFormProps) {
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
      // TODO: Replace with a dedicated magic-link endpoint (e.g. client.requestMagicLink({ email }))
      // once the AuthClient supports it. Using forgotPassword as a placeholder.
      await client.forgotPassword({ email });
      setIsSuccess(true);
      onSuccess?.();
    } catch (err) {
      setError(
        err instanceof Error
          ? err.message
          : "Failed to send magic link. Please try again.",
      );
    } finally {
      setIsSubmitting(false);
    }
  };

  const footer = signInUrl ? (
    <p className="text-[13px] text-muted-foreground">
      Sign in with password?{" "}
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
      title="Magic link"
      description="Enter your email to receive a sign-in link."
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
            Check your email for a sign-in link. If you don&apos;t see it, check
            your spam folder.
          </p>
        </div>
      ) : (
        <form onSubmit={handleSubmit} className="grid gap-3">
          <ErrorDisplay error={error} />

          <div className="grid gap-1.5">
            <Label htmlFor="magic-email" className="text-[13px]">
              Email address
            </Label>
            <Input
              id="magic-email"
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
            Send magic link
          </Button>
        </form>
      )}
    </AuthCard>
  );
}
