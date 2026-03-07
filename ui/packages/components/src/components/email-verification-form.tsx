"use client";

import * as React from "react";
import { REGEXP_ONLY_DIGITS } from "input-otp";
import { useAuth } from "@authsome/ui-react";
import { cn } from "../lib/utils";
import {
  InputOTP,
  InputOTPGroup,
  InputOTPSlot,
  InputOTPSeparator,
} from "../primitives/otp-input";
import { Button } from "../primitives/button";
import { AuthCard } from "./auth-card";
import { ErrorDisplay } from "./error-display";
import { LoadingSpinner } from "./loading-spinner";
import { MailCheck } from "lucide-react";

export interface EmailVerificationFormProps {
  /** The email address being verified. */
  email: string;
  /** Callback invoked after successful verification. */
  onSuccess?: () => void;
  /** Callback to resend the verification email. */
  onResend?: () => void;
  /** Optional logo element rendered above the title. */
  logo?: React.ReactNode;
  /** Additional CSS class names. */
  className?: string;
}

/**
 * A styled email-verification form.
 *
 * Displays a 6-digit OTP input for the user to enter the code sent to
 * their email address. Auto-submits when all digits are entered.
 * Includes a "Resend code" link and success confirmation state.
 */
export function EmailVerificationForm({
  email,
  onSuccess,
  onResend,
  logo,
  className,
}: EmailVerificationFormProps) {
  const { client } = useAuth();

  const [code, setCode] = React.useState("");
  const [error, setError] = React.useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = React.useState(false);
  const [isSuccess, setIsSuccess] = React.useState(false);
  const [resendCooldown, setResendCooldown] = React.useState(0);

  // Countdown timer for resend cooldown.
  React.useEffect(() => {
    if (resendCooldown <= 0) return;
    const timer = setInterval(() => {
      setResendCooldown((prev) => prev - 1);
    }, 1000);
    return () => clearInterval(timer);
  }, [resendCooldown]);

  const handleSubmit = React.useCallback(
    async (otpCode: string) => {
      if (otpCode.length !== 6 || isSubmitting) return;

      setError(null);
      setIsSubmitting(true);

      try {
        await client.verifyEmail({ token: otpCode });
        setIsSuccess(true);
        onSuccess?.();
      } catch (err) {
        const message =
          err instanceof Error
            ? err.message
            : "Invalid verification code. Please try again.";
        setError(message);
        setCode("");
      } finally {
        setIsSubmitting(false);
      }
    },
    [client, isSubmitting, onSuccess],
  );

  const handleChange = React.useCallback((value: string) => {
    setCode(value);
    setError(null);
  }, []);

  const handleComplete = React.useCallback(
    (value: string) => {
      void handleSubmit(value);
    },
    [handleSubmit],
  );

  const handleFormSubmit = React.useCallback(
    (e: React.FormEvent) => {
      e.preventDefault();
      void handleSubmit(code);
    },
    [code, handleSubmit],
  );

  const handleResend = React.useCallback(async () => {
    if (resendCooldown > 0) return;
    try {
      onResend?.();
      setResendCooldown(60);
    } catch {
      // Silently handle resend errors
    }
  }, [onResend, resendCooldown]);

  if (isSuccess) {
    return (
      <AuthCard
        title="Email verified"
        description="Your email has been verified successfully."
        logo={logo}
        className={cn(className)}
      >
        <div className="grid gap-3 text-center">
          <div className="mx-auto flex h-10 w-10 items-center justify-center rounded-full bg-green-100 dark:bg-green-900/30">
            <MailCheck className="h-5 w-5 text-green-600 dark:text-green-400" />
          </div>
          <p className="text-[13px] text-muted-foreground">
            You can now continue to your account.
          </p>
        </div>
      </AuthCard>
    );
  }

  return (
    <AuthCard
      title="Verify your email"
      description={`Enter the 6-digit code sent to ${email}`}
      logo={logo}
      className={cn(className)}
    >
      <form onSubmit={handleFormSubmit} className="grid gap-3">
        <div className="flex justify-center">
          <InputOTP
            maxLength={6}
            pattern={REGEXP_ONLY_DIGITS}
            value={code}
            onChange={handleChange}
            onComplete={handleComplete}
            disabled={isSubmitting}
          >
            <InputOTPGroup>
              <InputOTPSlot index={0} />
              <InputOTPSlot index={1} />
              <InputOTPSlot index={2} />
            </InputOTPGroup>
            <InputOTPSeparator />
            <InputOTPGroup>
              <InputOTPSlot index={3} />
              <InputOTPSlot index={4} />
              <InputOTPSlot index={5} />
            </InputOTPGroup>
          </InputOTP>
        </div>

        <ErrorDisplay error={error} />

        <Button
          type="submit"
          className="w-full"
          disabled={code.length !== 6 || isSubmitting}
        >
          {isSubmitting ? (
            <>
              <LoadingSpinner size="sm" className="mr-2" />
              Verifying...
            </>
          ) : (
            "Verify email"
          )}
        </Button>

        <p className="text-center text-[13px] text-muted-foreground">
          Didn&apos;t receive a code?{" "}
          {resendCooldown > 0 ? (
            <span className="text-muted-foreground/70">
              Resend in {resendCooldown}s
            </span>
          ) : (
            <button
              type="button"
              className="font-medium text-foreground underline-offset-4 hover:underline"
              onClick={() => void handleResend()}
            >
              Resend
            </button>
          )}
        </p>
      </form>
    </AuthCard>
  );
}
