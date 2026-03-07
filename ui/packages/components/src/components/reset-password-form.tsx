"use client";

import * as React from "react";
import { useState } from "react";
import { useAuth } from "@authsome/ui-react";
import { cn } from "../lib/utils";
import { Button } from "../primitives/button";
import { Label } from "../primitives/label";
import { AuthCard } from "./auth-card";
import { ErrorDisplay } from "./error-display";
import { LoadingSpinner } from "./loading-spinner";
import { PasswordInput } from "./password-input";

export interface ResetPasswordFormProps {
  /** The password-reset token (typically from a URL query parameter). */
  token: string;
  /** Callback invoked after the password has been reset successfully. */
  onSuccess?: () => void;
  /** Optional logo element rendered above the title. */
  logo?: React.ReactNode;
  /** Additional CSS class names. */
  className?: string;
}

/**
 * A styled reset-password form with Clerk-style polish.
 *
 * Accepts a reset token and lets the user set a new password.
 * Validates that both password fields match before submitting.
 */
export function ResetPasswordForm({
  token,
  onSuccess,
  logo,
  className,
}: ResetPasswordFormProps) {
  const { client } = useAuth();

  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  const handleSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setError(null);

    if (password !== confirmPassword) {
      setError("Passwords do not match.");
      return;
    }

    setIsSubmitting(true);

    try {
      await client.resetPassword({ token, new_password: password });
      onSuccess?.();
    } catch (err) {
      setError(
        err instanceof Error
          ? err.message
          : "Failed to reset password. Please try again.",
      );
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <AuthCard
      title="Reset password"
      description="Enter your new password."
      logo={logo}
      className={cn(className)}
    >
      <form onSubmit={handleSubmit} className="grid gap-3">
        <ErrorDisplay error={error} />

        <div className="grid gap-1.5">
          <Label htmlFor="reset-password" className="text-[13px]">
            New password
          </Label>
          <PasswordInput
            id="reset-password"
            placeholder="Enter new password"
            autoComplete="new-password"
            required
            disabled={isSubmitting}
            value={password}
            onChange={(e) => setPassword(e.target.value)}
          />
        </div>

        <div className="grid gap-1.5">
          <Label htmlFor="reset-confirm-password" className="text-[13px]">
            Confirm password
          </Label>
          <PasswordInput
            id="reset-confirm-password"
            placeholder="Confirm new password"
            autoComplete="new-password"
            required
            disabled={isSubmitting}
            value={confirmPassword}
            onChange={(e) => setConfirmPassword(e.target.value)}
          />
        </div>

        <Button type="submit" className="w-full" disabled={isSubmitting}>
          {isSubmitting && <LoadingSpinner size="sm" className="mr-2" />}
          Reset password
        </Button>
      </form>
    </AuthCard>
  );
}
