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
import { CheckCircle2 } from "lucide-react";

export interface WaitlistFormProps {
  /** Callback invoked after a successful waitlist submission. */
  onSuccess?: () => void;
  /** URL to the sign-in page. Renders an "Already have an account?" footer link. */
  signInUrl?: string;
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
 * A waitlist sign-up form that collects email and optional name.
 *
 * - **Auto-configuration**: When `publishableKey` is set on `AuthProvider`, the form
 *   checks `config.waitlist.enabled` from the backend client config.
 * - On submit, sends a POST to `/v1/waitlist/join` with `{ email, name }`.
 * - Shows a success state with a green checkmark once the user is on the list.
 */
export function WaitlistForm({
  onSuccess,
  signInUrl,
  logo,
  align,
  variant,
  className,
}: WaitlistFormProps) {
  const { client } = useAuth();
  const { config } = useClientConfig();

  const [email, setEmail] = useState("");
  const [name, setName] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [isSuccess, setIsSuccess] = useState(false);

  const handleSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setError(null);
    setIsSubmitting(true);

    try {
      // Access baseURL from the client instance.
      const baseURL = (client as any).baseURL ?? "";
      const res = await fetch(baseURL + "/v1/waitlist/join", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email, name: name || undefined }),
      });

      if (!res.ok) {
        const data = await res.json().catch(() => null);
        throw new Error(
          data?.error ?? "Something went wrong. Please try again.",
        );
      }

      setIsSuccess(true);
      onSuccess?.();
    } catch (err) {
      setError(
        err instanceof Error
          ? err.message
          : "Failed to join the waitlist. Please try again.",
      );
    } finally {
      setIsSubmitting(false);
    }
  };

  // If waitlist is explicitly disabled, don't render.
  if (config?.waitlist && !config.waitlist.enabled) {
    return null;
  }

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

  /* -- Success state ---------------------------------------- */

  if (isSuccess) {
    return (
      <AuthCard
        title="Join the Waitlist"
        description="Sign up to be notified when access is available."
        logo={logo}
        footer={footer}
        align={align}
        variant={variant}
        className={cn(className)}
      >
        <div className="grid gap-4">
          <div className="flex flex-col items-center gap-3 py-4 text-center">
            <div className="flex h-10 w-10 items-center justify-center rounded-full bg-green-100 dark:bg-green-900/30">
              <CheckCircle2 className="h-5 w-5 text-green-600 dark:text-green-400" />
            </div>
            <div className="grid gap-1">
              <p className="text-sm font-medium text-foreground">
                You're on the list!
              </p>
              <p className="text-[13px] text-muted-foreground">
                We'll notify you when your spot is ready.
              </p>
            </div>
          </div>
        </div>
      </AuthCard>
    );
  }

  /* -- Form state ------------------------------------------- */

  return (
    <AuthCard
      title="Join the Waitlist"
      description="Sign up to be notified when access is available."
      logo={logo}
      footer={footer}
      align={align}
      variant={variant}
      className={cn(className)}
    >
      <div className="grid gap-4">
        <form onSubmit={handleSubmit} className="grid gap-3">
          <ErrorDisplay error={error} />

          <div className="grid gap-1.5">
            <Label htmlFor="waitlist-email" className="text-[13px]">
              Email address
            </Label>
            <Input
              id="waitlist-email"
              type="email"
              placeholder="name@example.com"
              autoComplete="email"
              required
              disabled={isSubmitting}
              value={email}
              onChange={(e) => setEmail(e.target.value)}
            />
          </div>

          <div className="grid gap-1.5">
            <Label htmlFor="waitlist-name" className="text-[13px]">
              Name
              <span className="ml-1 text-muted-foreground">(optional)</span>
            </Label>
            <Input
              id="waitlist-name"
              type="text"
              placeholder="John Doe"
              autoComplete="name"
              disabled={isSubmitting}
              value={name}
              onChange={(e) => setName(e.target.value)}
            />
          </div>

          <Button
            type="submit"
            className="w-full"
            disabled={isSubmitting}
          >
            {isSubmitting && <LoadingSpinner size="sm" className="mr-2" />}
            Join Waitlist
          </Button>
        </form>
      </div>
    </AuthCard>
  );
}
