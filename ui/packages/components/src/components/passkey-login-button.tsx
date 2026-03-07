"use client";

import * as React from "react";
import { useState } from "react";
import { useAuth } from "@authsome/ui-react";
import { Fingerprint } from "lucide-react";
import { cn } from "../lib/utils";
import { Button } from "../primitives/button";
import { LoadingSpinner } from "./loading-spinner";

export interface PasskeyLoginButtonProps {
  /** Callback invoked after a successful passkey login. */
  onSuccess?: () => void;
  /** Callback invoked when an error occurs during passkey login. */
  onError?: (error: Error) => void;
  /** Additional CSS class names. */
  className?: string;
  /** Button variant. */
  variant?: "default" | "outline" | "ghost";
  /** Button size. */
  size?: "default" | "sm" | "lg";
  /** Button label text. */
  label?: string;
}

/**
 * A standalone button that initiates WebAuthn passkey login.
 *
 * Flow: passkeyLoginBegin -> navigator.credentials.get -> passkeyLoginFinish.
 * Intended to be placed inside existing sign-in forms rather than wrapped
 * in its own AuthCard.
 */
export function PasskeyLoginButton({
  onSuccess,
  onError,
  className,
  variant = "default",
  size = "default",
  label = "Sign in with passkey",
}: PasskeyLoginButtonProps): React.ReactNode {
  const { client } = useAuth();

  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const isWebAuthnAvailable =
    typeof window !== "undefined" &&
    window.PublicKeyCredential !== undefined &&
    navigator.credentials !== undefined;

  async function handlePasskeyLogin(): Promise<void> {
    setError(null);
    setIsLoading(true);

    try {
      const { options } = await client.passkeyLoginBegin({});

      const credential = await navigator.credentials.get({
        publicKey: options as unknown as PublicKeyCredentialRequestOptions,
      });

      if (!credential) {
        throw new Error("No credential returned from authenticator");
      }

      await client.passkeyLoginFinish(credential);
      onSuccess?.();
    } catch (err) {
      const error =
        err instanceof Error ? err : new Error("Passkey login failed");
      setError(error.message);
      onError?.(error);
    } finally {
      setIsLoading(false);
    }
  }

  if (!isWebAuthnAvailable) {
    return null;
  }

  return (
    <div className={cn("grid gap-1.5", className)}>
      <Button
        type="button"
        variant={variant}
        size={size}
        disabled={isLoading}
        onClick={() => void handlePasskeyLogin()}
      >
        {isLoading ? (
          <LoadingSpinner size="sm" className="mr-2" />
        ) : (
          <Fingerprint className="mr-2 h-3.5 w-3.5" />
        )}
        {label}
      </Button>
      {error && (
        <p className="text-xs text-destructive">{error}</p>
      )}
    </div>
  );
}
