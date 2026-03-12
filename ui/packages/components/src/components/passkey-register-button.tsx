"use client";

import * as React from "react";
import { useState } from "react";
import { useAuth } from "@authsome/ui-react";
import {
  prepareCreationOptions,
  serializeCredential,
} from "@authsome/ui-core";
import { Key, Plus } from "lucide-react";
import { cn } from "../lib/utils";
import { Button } from "../primitives/button";
import { LoadingSpinner } from "./loading-spinner";

export interface PasskeyRegisterButtonProps {
  /** Display name for the new passkey credential. */
  displayName?: string;
  /** Callback invoked after a successful registration. */
  onSuccess?: (credential: { id: string; display_name: string }) => void;
  /** Callback invoked when an error occurs during registration. */
  onError?: (error: Error) => void;
  /** Additional CSS class names. */
  className?: string;
  /** Button label text. */
  label?: string;
}

/**
 * A button for authenticated users to register a new WebAuthn passkey.
 *
 * Flow: passkeyRegisterBegin -> navigator.credentials.create -> passkeyRegisterFinish.
 * Requires an active session with a valid session token.
 */
export function PasskeyRegisterButton({
  displayName,
  onSuccess,
  onError,
  className,
  label = "Register passkey",
}: PasskeyRegisterButtonProps): React.ReactNode {
  const { client, session } = useAuth();

  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function handleRegister(): Promise<void> {
    if (!session?.session_token) {
      const err = new Error("You must be signed in to register a passkey");
      setError(err.message);
      onError?.(err);
      return;
    }

    setError(null);
    setIsLoading(true);

    try {
      const token = session.session_token;

      const { options } = await client.passkeyRegisterBegin(
        { display_name: displayName },
        token,
      );

      const publicKey = prepareCreationOptions(
        options as Record<string, unknown>,
      );
      const credential = await navigator.credentials.create({ publicKey });

      if (!credential) {
        throw new Error("No credential returned from authenticator");
      }

      const result = await client.passkeyRegisterFinish(
        serializeCredential(credential),
        token,
      );
      onSuccess?.({ id: result.id, display_name: result.display_name });
    } catch (err) {
      const error =
        err instanceof Error ? err : new Error("Passkey registration failed");
      setError(error.message);
      onError?.(error);
    } finally {
      setIsLoading(false);
    }
  }

  return (
    <div className={cn("grid gap-1.5", className)}>
      <Button
        type="button"
        variant="outline"
        disabled={isLoading}
        onClick={() => void handleRegister()}
      >
        {isLoading ? (
          <LoadingSpinner size="sm" className="mr-2" />
        ) : (
          <span className="mr-2 inline-flex items-center">
            <Key className="h-4 w-4" />
            <Plus className="h-3 w-3 -ml-1" />
          </span>
        )}
        {label}
      </Button>
      {error && (
        <p className="text-xs text-destructive">{error}</p>
      )}
    </div>
  );
}
