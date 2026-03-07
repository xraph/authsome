"use client";

import * as React from "react";
import { useState, useCallback } from "react";
import { useAuth } from "@authsome/ui-react";
import { CheckCircle2 } from "lucide-react";
import { cn } from "../lib/utils";
import { Button } from "../primitives/button";
import {
  InputOTP,
  InputOTPGroup,
  InputOTPSlot,
  InputOTPSeparator,
} from "../primitives/otp-input";
import { AuthCard } from "./auth-card";
import { ErrorDisplay } from "./error-display";
import { LoadingSpinner } from "./loading-spinner";

export interface DeviceAuthorizationFormProps {
  onSuccess?: () => void;
  onError?: (error: Error) => void;
  className?: string;
  logo?: React.ReactNode;
  /** Number of characters in the device code. Defaults to 8. */
  codeLength?: number;
}

const REGEXP_ALPHANUMERIC = /^[A-Z0-9]*$/;

/**
 * OAuth device authorization flow form. Users enter the code displayed on
 * their external device (TV, CLI tool, etc.) to authorize it.
 */
export function DeviceAuthorizationForm({
  onSuccess,
  onError,
  className,
  logo,
  codeLength = 8,
}: DeviceAuthorizationFormProps): React.ReactElement {
  const { client, session } = useAuth();

  const [code, setCode] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [isSuccess, setIsSuccess] = useState(false);

  const token = session?.session_token;
  const halfLength = Math.floor(codeLength / 2);

  const handleSubmit = useCallback(
    async (otpCode: string) => {
      if (otpCode.length !== codeLength || isSubmitting || !token) return;

      setError(null);
      setIsSubmitting(true);

      try {
        await client.trustDevice(otpCode, { DeviceID: otpCode }, token);
        setIsSuccess(true);
        onSuccess?.();
      } catch (err) {
        const message =
          err instanceof Error ? err.message : "Failed to authorize device";
        setError(message);
        setCode("");
        if (err instanceof Error) {
          onError?.(err);
        } else {
          onError?.(new Error(message));
        }
      } finally {
        setIsSubmitting(false);
      }
    },
    [client, codeLength, isSubmitting, onError, onSuccess, token],
  );

  const handleChange = useCallback((value: string) => {
    setCode(value.toUpperCase());
    setError(null);
  }, []);

  const handleComplete = useCallback(
    (value: string) => {
      void handleSubmit(value.toUpperCase());
    },
    [handleSubmit],
  );

  const handleFormSubmit = useCallback(
    (e: React.FormEvent) => {
      e.preventDefault();
      void handleSubmit(code);
    },
    [code, handleSubmit],
  );

  if (isSuccess) {
    return (
      <AuthCard
        title="Authorize Device"
        description="Enter the code shown on your device"
        logo={logo}
        className={cn(className)}
      >
        <div className="grid gap-3 text-center">
          <div className="mx-auto flex h-12 w-12 items-center justify-center rounded-full bg-green-100 dark:bg-green-900/30">
            <CheckCircle2 className="h-6 w-6 text-green-600 dark:text-green-400" />
          </div>
          <p className="text-sm text-muted-foreground">
            Device authorized successfully
          </p>
        </div>
      </AuthCard>
    );
  }

  return (
    <AuthCard
      title="Authorize Device"
      description="Enter the code shown on your device"
      logo={logo}
      className={cn(className)}
    >
      <form onSubmit={handleFormSubmit} className="space-y-5">
        <div className="flex justify-center">
          <InputOTP
            maxLength={codeLength}
            pattern={REGEXP_ALPHANUMERIC.source}
            value={code}
            onChange={handleChange}
            onComplete={handleComplete}
            disabled={isSubmitting}
          >
            <InputOTPGroup>
              {Array.from({ length: halfLength }).map((_, i) => (
                <InputOTPSlot key={i} index={i} />
              ))}
            </InputOTPGroup>
            <InputOTPSeparator />
            <InputOTPGroup>
              {Array.from({ length: codeLength - halfLength }).map((_, i) => (
                <InputOTPSlot key={halfLength + i} index={halfLength + i} />
              ))}
            </InputOTPGroup>
          </InputOTP>
        </div>

        <ErrorDisplay error={error} />

        <Button
          type="submit"
          size="lg"
          className="w-full"
          disabled={code.length !== codeLength || isSubmitting}
        >
          {isSubmitting ? (
            <>
              <LoadingSpinner size="sm" className="mr-2" />
              Authorizing...
            </>
          ) : (
            "Authorize"
          )}
        </Button>
      </form>
    </AuthCard>
  );
}
