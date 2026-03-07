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
import { Check } from "lucide-react";

export interface ChangePasswordFormProps {
  /** Callback invoked after a successful password change. */
  onSuccess?: () => void;
  /** Additional CSS class names. */
  className?: string;
}

/**
 * A styled change-password form for authenticated users.
 *
 * Requires the current password and a new password (with confirmation).
 * Uses `client.changePassword()` from the AuthSome API client.
 * Includes a password strength indicator.
 */
export function ChangePasswordForm({
  onSuccess,
  className,
}: ChangePasswordFormProps) {
  const { client, session } = useAuth();

  const [currentPassword, setCurrentPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [isSuccess, setIsSuccess] = useState(false);

  const passwordStrength = React.useMemo(() => {
    if (!newPassword) return { score: 0, label: "", color: "" };
    let score = 0;
    if (newPassword.length >= 8) score++;
    if (newPassword.length >= 12) score++;
    if (/[A-Z]/.test(newPassword)) score++;
    if (/[0-9]/.test(newPassword)) score++;
    if (/[^A-Za-z0-9]/.test(newPassword)) score++;

    if (score <= 1) return { score, label: "Weak", color: "bg-destructive" };
    if (score <= 2) return { score, label: "Fair", color: "bg-orange-500" };
    if (score <= 3) return { score, label: "Good", color: "bg-yellow-500" };
    return { score, label: "Strong", color: "bg-green-500" };
  }, [newPassword]);

  const handleSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setError(null);

    if (newPassword !== confirmPassword) {
      setError("New passwords do not match.");
      return;
    }

    if (newPassword.length < 8) {
      setError("Password must be at least 8 characters.");
      return;
    }

    setIsSubmitting(true);

    try {
      if (session) {
        await client.changePassword(
          { current_password: currentPassword, new_password: newPassword },
          session.session_token,
        );
      }
      setIsSuccess(true);
      setCurrentPassword("");
      setNewPassword("");
      setConfirmPassword("");
      onSuccess?.();
    } catch (err) {
      setError(
        err instanceof Error
          ? err.message
          : "Failed to change password. Please try again.",
      );
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <AuthCard
      title="Change password"
      description="Update your account password."
      className={cn(className)}
    >
      {isSuccess ? (
        <div className="grid gap-3 text-center">
          <div className="mx-auto flex h-10 w-10 items-center justify-center rounded-full bg-green-100 dark:bg-green-900/30">
            <Check className="h-5 w-5 text-green-600 dark:text-green-400" />
          </div>
          <p className="text-[13px] text-muted-foreground">
            Your password has been updated successfully.
          </p>
          <Button
            variant="outline"
            size="sm"
            className="mx-auto"
            onClick={() => setIsSuccess(false)}
          >
            Change again
          </Button>
        </div>
      ) : (
        <form onSubmit={handleSubmit} className="grid gap-3">
          <ErrorDisplay error={error} />

          <div className="grid gap-1.5">
            <Label htmlFor="current-password" className="text-[13px]">
              Current password
            </Label>
            <PasswordInput
              id="current-password"
              placeholder="Enter current password"
              autoComplete="current-password"
              required
              disabled={isSubmitting}
              value={currentPassword}
              onChange={(e) => setCurrentPassword(e.target.value)}
            />
          </div>

          <div className="grid gap-1.5">
            <Label htmlFor="new-password" className="text-[13px]">
              New password
            </Label>
            <PasswordInput
              id="new-password"
              placeholder="Enter new password"
              autoComplete="new-password"
              required
              disabled={isSubmitting}
              value={newPassword}
              onChange={(e) => setNewPassword(e.target.value)}
            />
            {newPassword && (
              <div className="grid gap-1">
                <div className="flex gap-1">
                  {Array.from({ length: 4 }).map((_, i) => (
                    <div
                      key={i}
                      className={cn(
                        "h-1 flex-1 rounded-full transition-colors",
                        i < passwordStrength.score
                          ? passwordStrength.color
                          : "bg-muted",
                      )}
                    />
                  ))}
                </div>
                <p className="text-[11px] text-muted-foreground">
                  {passwordStrength.label}
                </p>
              </div>
            )}
          </div>

          <div className="grid gap-1.5">
            <Label htmlFor="confirm-password" className="text-[13px]">
              Confirm new password
            </Label>
            <PasswordInput
              id="confirm-password"
              placeholder="Re-enter new password"
              autoComplete="new-password"
              required
              disabled={isSubmitting}
              value={confirmPassword}
              onChange={(e) => setConfirmPassword(e.target.value)}
            />
          </div>

          <Button type="submit" className="w-full" disabled={isSubmitting}>
            {isSubmitting && <LoadingSpinner size="sm" className="mr-2" />}
            Update password
          </Button>
        </form>
      )}
    </AuthCard>
  );
}
