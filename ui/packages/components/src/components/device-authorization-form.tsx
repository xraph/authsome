"use client";

import * as React from "react";
import { useState, useCallback, useEffect, useRef } from "react";
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
import { AuthCard, type AuthCardAlign, type AuthCardVariant } from "./auth-card";
import { ErrorDisplay } from "./error-display";
import { LoadingSpinner } from "./loading-spinner";

export interface DeviceAuthorizationFormProps {
  /** Custom submit handler. When provided, called instead of the built-in
   *  OAuth device complete flow. */
  onSubmit?: (code: string) => Promise<void>;
  /** Called after successful authorization. */
  onSuccess?: () => void;
  /** Called when an error occurs. */
  onError?: (error: Error) => void;
  /** Additional CSS class names. */
  className?: string;
  /** Optional logo element. */
  logo?: React.ReactNode;
  /** Title and description alignment. */
  align?: AuthCardAlign;
  /** Card visual style. */
  variant?: AuthCardVariant;
  /** Number of characters in the device code. Defaults to 8. */
  codeLength?: number;
  /** Pre-fill the code input. When omitted, auto-reads from URL
   *  query params `user_code` or `code`. */
  initialCode?: string;
  /** Auto-submit the code when pre-filled from URL or props. Defaults to true. */
  autoSubmit?: boolean;
}

const REGEXP_ALPHANUMERIC = /^[A-Z0-9]*$/;

/**
 * OAuth device authorization form (RFC 8628).
 *
 * Users enter the code displayed on their external device (CLI, TV, etc.)
 * to authorize it. The form handles the entire flow automatically:
 *
 * 1. Reads `user_code` from the URL query params (or `initialCode` prop)
 * 2. On submit, calls `POST {baseURL}/v1/oauth/device/complete` with the
 *    session token from the AuthProvider context
 * 3. Shows success/error states
 * 4. When code is pre-filled from URL (e.g. `?user_code=ABCD-EFGH`),
 *    auto-submits for quick validation
 *
 * ```tsx
 * <DeviceAuthorizationForm />
 * ```
 */
export function DeviceAuthorizationForm({
  onSubmit: onSubmitProp,
  onSuccess,
  onError,
  className,
  logo,
  align = "center",
  variant = "default",
  codeLength = 8,
  initialCode: initialCodeProp,
  autoSubmit = true,
}: DeviceAuthorizationFormProps): React.ReactElement {
  const { client, session, isLoading } = useAuth();

  // Auto-read code from URL when not explicitly provided.
  const autoCode = useCodeFromURL();
  const initialCode = initialCodeProp ?? autoCode;

  const [code, setCode] = useState(initialCode?.toUpperCase() ?? "");
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [isSuccess, setIsSuccess] = useState(false);
  const autoSubmittedRef = useRef(false);

  const token = session?.session_token ?? null;
  const halfLength = Math.floor(codeLength / 2);

  const handleSubmit = useCallback(
    async (rawCode: string) => {
      const clean = rawCode.replace(/[^A-Z0-9]/gi, "").toUpperCase();
      if (clean.length !== codeLength || isSubmitting) return;

      setError(null);
      setIsSubmitting(true);

      try {
        if (onSubmitProp) {
          await onSubmitProp(clean);
        } else if (client) {
          // Use the AuthClient method — routes through the client's baseURL.
          // Sends Bearer token when available; the method also sets
          // credentials: "include" so cookies are sent for same-origin setups.
          await (client as any).completeDeviceAuthorization(
            clean,
            "approve",
            token ?? undefined,
          );
        } else {
          throw new Error("Auth client not available");
        }

        setIsSuccess(true);
        onSuccess?.();
      } catch (err) {
        const message =
          err instanceof Error ? err.message : "Failed to authorize device";
        setError(message);
        setCode("");
        onError?.(err instanceof Error ? err : new Error(message));
      } finally {
        setIsSubmitting(false);
      }
    },
    [client, codeLength, isSubmitting, onError, onSubmitProp, onSuccess, token],
  );

  // Update code if initialCode changes.
  useEffect(() => {
    const newCode = (initialCodeProp ?? autoCode)?.toUpperCase();
    if (newCode && newCode !== code && !isSubmitting) {
      setCode(newCode);
    }
  }, [initialCodeProp, autoCode]);

  // Auto-submit when code is pre-filled from URL and is complete.
  // Wait for auth to finish loading so the token is available.
  useEffect(() => {
    if (
      autoSubmit &&
      !autoSubmittedRef.current &&
      !isSubmitting &&
      !isSuccess &&
      !isLoading &&
      code.length === codeLength
    ) {
      autoSubmittedRef.current = true;
      void handleSubmit(code);
    }
  }, [autoSubmit, code, codeLength, handleSubmit, isLoading, isSubmitting, isSuccess]);

  const handleChange = useCallback(
    (value: string) => {
      setCode(value.toUpperCase());
      setError(null);
    },
    [],
  );

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
        title="Device Authorized"
        description="Your device has been successfully authorized"
        logo={logo}
        align={align}
        variant={variant}
        className={cn(className)}
      >
        <div className="grid gap-3 text-center">
          <div className="mx-auto flex h-12 w-12 items-center justify-center rounded-full bg-green-100 dark:bg-green-900/30">
            <CheckCircle2 className="h-6 w-6 text-green-600 dark:text-green-400" />
          </div>
          <p className="text-sm font-medium">Device authorized successfully</p>
          <p className="text-sm text-muted-foreground">
            You can close this window and return to your device.
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
      align={align}
      variant={variant}
      className={cn(className)}
    >
      <form onSubmit={handleFormSubmit} className="space-y-5">
        {/* OTP Code Input */}
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
                <InputOTPSlot
                  key={i}
                  index={i}
                  className="h-11 w-12 text-base font-semibold uppercase sm:h-12 sm:text-lg"
                />
              ))}
            </InputOTPGroup>
            <InputOTPSeparator />
            <InputOTPGroup>
              {Array.from({ length: codeLength - halfLength }).map((_, i) => (
                <InputOTPSlot
                  key={halfLength + i}
                  index={halfLength + i}
                  className="h-11 w-12 text-base font-semibold uppercase sm:h-12 sm:text-lg"
                />
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

/**
 * Reads user_code or code from the current URL query params.
 * Supports both raw codes (`ABCDEFGH`) and dash-formatted (`ABCD-EFGH`).
 */
function useCodeFromURL(): string | undefined {
  const [code, setCode] = useState<string | undefined>(() => {
    if (typeof window === "undefined") return undefined;
    return parseCodeFromSearch(window.location.search);
  });

  useEffect(() => {
    setCode(parseCodeFromSearch(window.location.search));

    const handlePopState = () => {
      setCode(parseCodeFromSearch(window.location.search));
    };
    window.addEventListener("popstate", handlePopState);
    return () => window.removeEventListener("popstate", handlePopState);
  }, []);

  return code;
}

function parseCodeFromSearch(search: string): string | undefined {
  const params = new URLSearchParams(search);
  const raw = params.get("user_code") ?? params.get("code");
  if (!raw) return undefined;
  const cleaned = raw.replace(/[^A-Z0-9]/gi, "").toUpperCase();
  return cleaned || undefined;
}
