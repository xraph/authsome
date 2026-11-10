"use client";

import * as React from "react";
import {
  Badge,
  Button,
  ErrorDisplay,
  LoadingSpinner,
  Skeleton,
  cn,
} from "@authsome/ui-components";
import { DEFAULT_SCOPE_LABELS } from "./index";
import { useMyGrants, useRevokeGrant } from "./react";

export interface ConnectedAppsListProps {
  scopeLabels?: Record<string, string>;
  title?: React.ReactNode;
  emptyState?: React.ReactNode;
  className?: string;
}

function formatDate(iso: string): string | null {
  const d = new Date(iso);
  return Number.isNaN(d.getTime())
    ? null
    : d.toLocaleDateString(undefined, {
        day: "numeric",
        month: "short",
        year: "numeric",
      });
}

/** Apps you have authorised, and a way to withdraw access. */
export function ConnectedAppsList({
  scopeLabels,
  title = "Connected apps",
  emptyState = "No apps have access to your account.",
  className,
}: ConnectedAppsListProps) {
  const { grants, isLoading, error, reload } = useMyGrants();
  const { revoke, isSaving } = useRevokeGrant();
  const [pending, setPending] = React.useState<string | null>(null);

  const labels = React.useMemo(
    () => ({ ...DEFAULT_SCOPE_LABELS, ...scopeLabels }),
    [scopeLabels],
  );

  const handleRevoke = async (clientId: string) => {
    setPending(clientId);
    try {
      await revoke(clientId);
      await reload();
    } catch {
      // Surfaced by the hook and rendered above the list.
    } finally {
      setPending(null);
    }
  };

  return (
    <div className={cn("w-full", className)}>
      {title && (
        <h3 className="mb-3 text-[13px] font-medium uppercase tracking-wide text-muted-foreground">
          {title}
        </h3>
      )}

      <ErrorDisplay error={error?.message ?? null} />

      {isLoading ? (
        <div className="space-y-3 py-2">
          <Skeleton className="h-4 w-1/2" />
          <Skeleton className="h-4 w-2/3" />
        </div>
      ) : grants.length === 0 ? (
        <p className="py-2 text-[13px] text-muted-foreground">{emptyState}</p>
      ) : (
        <ul className="divide-y divide-border/60">
          {grants.map((g) => (
            <li
              key={g.client_id}
              className="flex items-start justify-between gap-4 py-3"
            >
              <div className="min-w-0 flex-1">
                <p className="text-sm font-medium">{g.client_name}</p>
                <p className="mt-0.5 text-[13px] text-muted-foreground">
                  Connected {formatDate(g.created_at) ?? ""}
                </p>
                <div className="mt-1.5 flex flex-wrap gap-1">
                  {g.scopes.map((s) => (
                    <Badge key={s} variant="outline" className="font-normal">
                      {labels[s] ?? s}
                    </Badge>
                  ))}
                </div>
              </div>
              <Button
                variant="outline"
                size="sm"
                className="shrink-0"
                disabled={isSaving && pending === g.client_id}
                onClick={() => void handleRevoke(g.client_id)}
              >
                {isSaving && pending === g.client_id ? (
                  <LoadingSpinner />
                ) : (
                  "Remove"
                )}
              </Button>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
