"use client";

import * as React from "react";
import {
  Badge,
  Button,
  ErrorDisplay,
  LoadingSpinner,
  Separator,
  Skeleton,
  cn,
} from "@authsome/ui-components";
import { useConsents, useRevokeConsent } from "./react";

export interface ConsentListProps {
  /** Readable names for purposes, keyed by purpose string. */
  labels?: Record<string, { title: string; description?: string }>;
  /** Only show these purposes, in this order. */
  purposes?: string[];
  title?: React.ReactNode;
  /** Shown when nothing has been agreed to. */
  emptyState?: React.ReactNode;
  className?: string;
}

function formatDate(iso?: string): string | null {
  if (!iso) return null;
  const d = new Date(iso);
  return Number.isNaN(d.getTime())
    ? null
    : d.toLocaleDateString(undefined, {
        day: "numeric",
        month: "short",
        year: "numeric",
      });
}

/**
 * What you have agreed to, and a way to take it back.
 *
 * A revoked record stays in the list rather than disappearing, because the
 * point of a consent record is that it is a history. Someone checking whether
 * they ever agreed to something needs to see that they did and then stopped.
 */
export function ConsentList({
  labels,
  purposes,
  title = "Your permissions",
  emptyState = "You have not agreed to anything yet.",
  className,
}: ConsentListProps) {
  const { consents, isLoading, error, reload } = useConsents();
  const { revoke, isSaving } = useRevokeConsent();
  const [pending, setPending] = React.useState<string | null>(null);

  const shown = React.useMemo(() => {
    if (!purposes?.length) return consents;
    const order = new Map(purposes.map((p, i) => [p, i]));
    return consents
      .filter((c) => order.has(c.purpose))
      .sort((a, b) => (order.get(a.purpose) ?? 0) - (order.get(b.purpose) ?? 0));
  }, [consents, purposes]);

  const handleRevoke = async (purpose: string) => {
    setPending(purpose);
    try {
      await revoke({ purpose });
      await reload();
    } catch {
      // The error is surfaced by the hook and rendered below.
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
          <Skeleton className="h-4 w-2/3" />
          <Skeleton className="h-4 w-1/2" />
        </div>
      ) : shown.length === 0 ? (
        <p className="py-2 text-[13px] text-muted-foreground">{emptyState}</p>
      ) : (
        <ul className="divide-y divide-border/60">
          {shown.map((c) => {
            const label = labels?.[c.purpose];
            const granted = c.granted && !c.revoked_at;
            return (
              <li
                key={c.id}
                className="flex items-start justify-between gap-4 py-3"
              >
                <div className="min-w-0 flex-1">
                  <p className="text-sm font-medium">
                    {label?.title ?? c.purpose}
                  </p>
                  {label?.description && (
                    <p className="mt-0.5 text-[13px] text-muted-foreground">
                      {label.description}
                    </p>
                  )}
                  <p className="mt-1 text-[13px] text-muted-foreground">
                    {granted
                      ? `Agreed ${formatDate(c.granted_at) ?? ""} · version ${c.version}`
                      : `Withdrawn ${formatDate(c.revoked_at) ?? ""}`}
                  </p>
                </div>
                <div className="flex shrink-0 items-center gap-2">
                  <Badge variant={granted ? "secondary" : "outline"} className="font-normal">
                    {granted ? "On" : "Off"}
                  </Badge>
                  {granted && (
                    <Button
                      variant="outline"
                      size="sm"
                      disabled={isSaving && pending === c.purpose}
                      onClick={() => void handleRevoke(c.purpose)}
                    >
                      {isSaving && pending === c.purpose ? (
                        <LoadingSpinner />
                      ) : (
                        "Withdraw"
                      )}
                    </Button>
                  )}
                </div>
              </li>
            );
          })}
        </ul>
      )}
      <Separator className="mt-1 opacity-0" />
    </div>
  );
}
