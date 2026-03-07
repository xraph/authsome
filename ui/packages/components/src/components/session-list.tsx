"use client";

import * as React from "react";
import { useState, useEffect, useCallback } from "react";
import { useAuth } from "@authsome/ui-react";
import { Globe, LogOut } from "lucide-react";
import { cn } from "../lib/utils";
import { Button } from "../primitives/button";
import { Badge } from "../primitives/badge";
import { Skeleton } from "../primitives/skeleton";
import { Separator } from "../primitives/separator";
import {
  Card,
  CardHeader,
  CardTitle,
  CardDescription,
  CardContent,
  CardFooter,
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

interface Session {
  id: string;
  device?: string;
  browser?: string;
  ip_address?: string;
  last_active?: string;
  created_at: string;
  session_token?: string;
}

export interface SessionListProps {
  className?: string;
  currentSessionToken?: string;
  onRevoke?: (sessionId: string) => void;
}

function timeAgo(dateStr: string): string {
  const seconds = Math.floor(
    (Date.now() - new Date(dateStr).getTime()) / 1000,
  );

  if (seconds < 60) return "just now";
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m ago`;
  if (seconds < 86400) return `${Math.floor(seconds / 3600)}h ago`;
  if (seconds < 2592000) return `${Math.floor(seconds / 86400)}d ago`;
  return `${Math.floor(seconds / 2592000)}mo ago`;
}

/**
 * Displays the current user's active sessions with the ability to revoke
 * individual sessions or all other sessions at once.
 */
export function SessionList({
  className,
  currentSessionToken,
  onRevoke,
}: SessionListProps): React.ReactElement {
  const { client, session } = useAuth();

  const [sessions, setSessions] = useState<Session[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [actionLoading, setActionLoading] = useState<string | null>(null);
  const [revokeTarget, setRevokeTarget] = useState<Session | null>(null);
  const [isRevokingAll, setIsRevokingAll] = useState(false);

  const token = session?.session_token;
  const activeToken = currentSessionToken ?? token;

  const fetchSessions = useCallback(async () => {
    if (!token) return;

    setIsLoading(true);
    setError(null);

    try {
      const response = await client.listSessions(token);
      setSessions((response.sessions ?? []) as unknown as Session[]);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to load sessions",
      );
    } finally {
      setIsLoading(false);
    }
  }, [client, token]);

  useEffect(() => {
    void fetchSessions();
  }, [fetchSessions]);

  const isCurrentSession = (s: Session): boolean => {
    return Boolean(activeToken && s.session_token === activeToken);
  };

  const handleRevoke = async (s: Session) => {
    if (!token) return;

    setActionLoading(s.id);
    setRevokeTarget(null);

    try {
      await client.revokeSession(s.id, { SessionID: s.id }, token);
      await fetchSessions();
      onRevoke?.(s.id);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to revoke session",
      );
    } finally {
      setActionLoading(null);
    }
  };

  const handleRevokeAll = async () => {
    if (!token) return;

    setIsRevokingAll(true);
    setError(null);

    try {
      const otherSessions = sessions.filter((s) => !isCurrentSession(s));

      for (const s of otherSessions) {
        await client.revokeSession(s.id, { SessionID: s.id }, token);
      }

      await fetchSessions();
    } catch (err) {
      setError(
        err instanceof Error
          ? err.message
          : "Failed to revoke sessions",
      );
    } finally {
      setIsRevokingAll(false);
    }
  };

  const otherSessionCount = sessions.filter(
    (s) => !isCurrentSession(s),
  ).length;

  return (
    <>
      <Card className={cn("mx-auto w-full max-w-md", className)}>
        <CardHeader>
          <CardTitle>Active Sessions</CardTitle>
          <CardDescription>
            Manage your active sessions across devices
          </CardDescription>
        </CardHeader>

        <CardContent className="grid gap-1">
          <ErrorDisplay error={error} className="mb-3" />

          {isLoading ? (
            <div className="grid gap-4">
              {Array.from({ length: 3 }).map((_, i) => (
                <div key={i} className="flex items-center gap-3">
                  <Skeleton className="h-10 w-10 rounded-lg" />
                  <div className="flex-1 space-y-2">
                    <Skeleton className="h-4 w-32" />
                    <Skeleton className="h-3 w-48" />
                  </div>
                </div>
              ))}
            </div>
          ) : sessions.length === 0 ? (
            <div className="flex flex-col items-center gap-2 py-8 text-center">
              <Globe className="h-10 w-10 text-muted-foreground/50" />
              <p className="text-sm text-muted-foreground">
                No active sessions
              </p>
            </div>
          ) : (
            <div className="grid gap-1">
              {sessions.map((s, index) => {
                const isCurrent = isCurrentSession(s);

                return (
                  <React.Fragment key={s.id}>
                    {index > 0 && <Separator />}
                    <div className="flex items-center gap-3 py-3">
                      <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-muted">
                        <Globe className="h-5 w-5 text-muted-foreground" />
                      </div>

                      <div className="min-w-0 flex-1">
                        <p className="truncate text-sm font-medium">
                          {[s.device, s.browser]
                            .filter(Boolean)
                            .join(" \u00B7 ") || "Unknown session"}
                        </p>
                        {s.ip_address && (
                          <p className="text-xs text-muted-foreground">
                            {s.ip_address}
                          </p>
                        )}
                        <p className="text-xs text-muted-foreground/70">
                          Active{" "}
                          {timeAgo(s.last_active ?? s.created_at)}
                        </p>
                      </div>

                      <div className="flex shrink-0 items-center gap-2">
                        {isCurrent && (
                          <Badge
                            variant="secondary"
                            className="bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400"
                          >
                            Current
                          </Badge>
                        )}

                        <Button
                          variant="ghost"
                          size="icon"
                          disabled={isCurrent || actionLoading === s.id}
                          onClick={() => setRevokeTarget(s)}
                          title={
                            isCurrent
                              ? "Cannot revoke current session"
                              : "Revoke session"
                          }
                        >
                          <LogOut className="h-4 w-4" />
                        </Button>
                      </div>
                    </div>
                  </React.Fragment>
                );
              })}
            </div>
          )}
        </CardContent>

        {!isLoading && otherSessionCount > 0 && (
          <CardFooter className="border-t px-6 py-4">
            <Button
              variant="destructive"
              className="w-full"
              disabled={isRevokingAll}
              onClick={() => void handleRevokeAll()}
            >
              {isRevokingAll && (
                <LoadingSpinner size="sm" className="mr-2" />
              )}
              Revoke all other sessions
            </Button>
          </CardFooter>
        )}
      </Card>

      <Dialog
        open={revokeTarget !== null}
        onOpenChange={(open) => {
          if (!open) setRevokeTarget(null);
        }}
      >
        <DialogContent className="sm:max-w-sm">
          <DialogHeader>
            <DialogTitle>Revoke session</DialogTitle>
            <DialogDescription>
              Are you sure you want to revoke this session? The device will be
              signed out immediately.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setRevokeTarget(null)}>
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={() => {
                if (revokeTarget) void handleRevoke(revokeTarget);
              }}
            >
              Revoke
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
