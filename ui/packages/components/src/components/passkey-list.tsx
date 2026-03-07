"use client";

import * as React from "react";
import { useEffect, useState, useCallback } from "react";
import { useAuth } from "@authsome/ui-react";
import {
  Key,
  Trash2,
  Usb,
  Bluetooth,
  Nfc,
  Smartphone,
  Shield,
  Plus,
} from "lucide-react";
import { cn } from "../lib/utils";
import { Button } from "../primitives/button";
import { Badge } from "../primitives/badge";
import { Skeleton } from "../primitives/skeleton";
import {
  Card,
  CardHeader,
  CardTitle,
  CardDescription,
  CardContent,
} from "../primitives/card";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "../primitives/dialog";
import { ErrorDisplay } from "./error-display";
import { LoadingSpinner } from "./loading-spinner";

interface CredentialInfo {
  id: string;
  display_name: string;
  created_at: string;
  transport: string[];
}

export interface PasskeyListProps {
  /** Additional CSS class names. */
  className?: string;
  /** Callback invoked after a passkey is deleted. */
  onDelete?: (credentialId: string) => void;
  /** Callback invoked when the register button is clicked. */
  onRegister?: () => void;
}

function formatRelativeTime(dateString: string): string {
  const now = Date.now();
  const then = new Date(dateString).getTime();
  const diffMs = now - then;

  const seconds = Math.floor(diffMs / 1000);
  const minutes = Math.floor(seconds / 60);
  const hours = Math.floor(minutes / 60);
  const days = Math.floor(hours / 24);

  if (days > 0) {
    return days === 1 ? "1 day ago" : `${days} days ago`;
  }
  if (hours > 0) {
    return hours === 1 ? "1 hour ago" : `${hours} hours ago`;
  }
  if (minutes > 0) {
    return minutes === 1 ? "1 minute ago" : `${minutes} minutes ago`;
  }
  return "Just now";
}

function TransportBadge({ transport }: { transport: string }): React.ReactNode {
  const iconClass = "h-3 w-3 mr-1";

  switch (transport) {
    case "internal":
      return (
        <Badge variant="secondary" className="gap-0.5 text-xs">
          <Smartphone className={iconClass} />
          Device
        </Badge>
      );
    case "usb":
      return (
        <Badge variant="secondary" className="gap-0.5 text-xs">
          <Usb className={iconClass} />
          USB
        </Badge>
      );
    case "ble":
      return (
        <Badge variant="secondary" className="gap-0.5 text-xs">
          <Bluetooth className={iconClass} />
          Bluetooth
        </Badge>
      );
    case "nfc":
      return (
        <Badge variant="secondary" className="gap-0.5 text-xs">
          <Nfc className={iconClass} />
          NFC
        </Badge>
      );
    default:
      return (
        <Badge variant="secondary" className="text-xs">
          {transport}
        </Badge>
      );
  }
}

function PasskeyRow({
  credential,
  onDeleteClick,
}: {
  credential: CredentialInfo;
  onDeleteClick: (credential: CredentialInfo) => void;
}): React.ReactNode {
  return (
    <div className="flex items-center justify-between gap-3 rounded-lg border p-3">
      <div className="flex items-center gap-3 min-w-0">
        <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-md bg-muted">
          <Key className="h-4 w-4 text-muted-foreground" />
        </div>
        <div className="min-w-0">
          <p className="truncate text-sm font-medium">
            {credential.display_name || "Unnamed passkey"}
          </p>
          <div className="flex flex-wrap items-center gap-1.5 mt-1">
            {credential.transport.map((t) => (
              <TransportBadge key={t} transport={t} />
            ))}
            <span className="text-xs text-muted-foreground">
              {formatRelativeTime(credential.created_at)}
            </span>
          </div>
        </div>
      </div>
      <Button
        variant="ghost"
        size="icon"
        className="shrink-0 text-muted-foreground hover:text-destructive"
        onClick={() => onDeleteClick(credential)}
      >
        <Trash2 className="h-4 w-4" />
        <span className="sr-only">Delete passkey</span>
      </Button>
    </div>
  );
}

function EmptyState({
  onRegister,
}: {
  onRegister?: () => void;
}): React.ReactNode {
  return (
    <div className="flex flex-col items-center gap-3 py-8 text-center">
      <div className="flex h-12 w-12 items-center justify-center rounded-full bg-muted">
        <Shield className="h-6 w-6 text-muted-foreground" />
      </div>
      <div className="space-y-1">
        <p className="text-sm font-medium">No passkeys registered</p>
        <p className="text-xs text-muted-foreground">
          Add a passkey for fast, passwordless sign-in
        </p>
      </div>
      {onRegister && (
        <Button variant="outline" size="sm" onClick={onRegister}>
          <Plus className="mr-1.5 h-4 w-4" />
          Register passkey
        </Button>
      )}
    </div>
  );
}

function LoadingSkeleton(): React.ReactNode {
  return (
    <div className="grid gap-2">
      {Array.from({ length: 2 }).map((_, i) => (
        <div key={i} className="flex items-center gap-3 rounded-lg border p-3">
          <Skeleton className="h-9 w-9 rounded-md" />
          <div className="flex-1 space-y-2">
            <Skeleton className="h-4 w-32" />
            <Skeleton className="h-3 w-48" />
          </div>
        </div>
      ))}
    </div>
  );
}

/**
 * Displays a card listing the current user's registered WebAuthn passkeys.
 *
 * Supports deleting passkeys (with confirmation dialog) and an optional
 * register button. Requires an active session.
 */
export function PasskeyList({
  className,
  onDelete,
  onRegister,
}: PasskeyListProps): React.ReactNode {
  const { client, session } = useAuth();

  const [credentials, setCredentials] = useState<CredentialInfo[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<CredentialInfo | null>(null);
  const [isDeleting, setIsDeleting] = useState(false);

  const fetchPasskeys = useCallback(async () => {
    if (!session?.session_token) {
      setIsLoading(false);
      return;
    }

    setError(null);
    setIsLoading(true);

    try {
      const { credentials } = await client.listPasskeys(
        session.session_token,
      );
      setCredentials(credentials);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to load passkeys",
      );
    } finally {
      setIsLoading(false);
    }
  }, [client, session?.session_token]);

  useEffect(() => {
    void fetchPasskeys();
  }, [fetchPasskeys]);

  async function handleDelete(): Promise<void> {
    if (!deleteTarget || !session?.session_token) {
      return;
    }

    setIsDeleting(true);

    try {
      await client.deletePasskey(
        deleteTarget.id,
        { CredentialID: deleteTarget.id },
        session.session_token,
      );
      setCredentials((prev) => prev.filter((c) => c.id !== deleteTarget.id));
      setDeleteTarget(null);
      onDelete?.(deleteTarget.id);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to delete passkey",
      );
      setDeleteTarget(null);
    } finally {
      setIsDeleting(false);
    }
  }

  return (
    <>
      <Card className={cn("mx-auto w-full max-w-md", className)}>
        <CardHeader className="flex flex-row items-start justify-between space-y-0">
          <div className="space-y-1.5">
            <CardTitle className="text-lg font-semibold">Passkeys</CardTitle>
            <CardDescription>
              Manage your passkeys for passwordless sign-in
            </CardDescription>
          </div>
          {onRegister && (
            <Button variant="outline" size="sm" onClick={onRegister}>
              <Plus className="mr-1.5 h-4 w-4" />
              Add
            </Button>
          )}
        </CardHeader>
        <CardContent>
          <ErrorDisplay error={error} className="mb-3" />

          {isLoading && <LoadingSkeleton />}

          {!isLoading && credentials.length === 0 && !error && (
            <EmptyState onRegister={onRegister} />
          )}

          {!isLoading && credentials.length > 0 && (
            <div className="grid gap-2">
              {credentials.map((credential) => (
                <PasskeyRow
                  key={credential.id}
                  credential={credential}
                  onDeleteClick={setDeleteTarget}
                />
              ))}
            </div>
          )}
        </CardContent>
      </Card>

      <Dialog
        open={deleteTarget !== null}
        onOpenChange={(open) => {
          if (!open) {
            setDeleteTarget(null);
          }
        }}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete passkey?</DialogTitle>
            <DialogDescription>
              This will permanently remove the passkey
              {deleteTarget?.display_name
                ? ` "${deleteTarget.display_name}"`
                : ""}
              . You will no longer be able to use it to sign in.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter className="gap-2 sm:gap-0">
            <Button
              variant="outline"
              onClick={() => setDeleteTarget(null)}
              disabled={isDeleting}
            >
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={() => void handleDelete()}
              disabled={isDeleting}
            >
              {isDeleting && <LoadingSpinner size="sm" className="mr-2" />}
              Delete
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
