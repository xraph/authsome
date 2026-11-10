"use client";

import * as React from "react";
import { Button, ErrorDisplay, LoadingSpinner, cn } from "@authsome/ui-components";
import { useConsents, useGrantConsent } from "./react";

export interface ConsentGateProps {
  /** The purpose that has to be agreed to. */
  purpose: string;
  /** The wording's version. Bump it and everyone is asked again. */
  version?: string;
  title: React.ReactNode;
  description?: React.ReactNode;
  acceptLabel?: string;
  /** Rendered once consent is in place. */
  children: React.ReactNode;
  /** Rendered while the answer is unknown. Defaults to nothing. */
  fallback?: React.ReactNode;
  className?: string;
}

/**
 * Holds its children back until a named purpose has been agreed to.
 *
 * Nothing renders while the answer is still loading, because flashing the
 * gated content and then hiding it would defeat the point of gating it.
 *
 * When `version` is given, a consent recorded against an older version does
 * not count and the prompt comes back. That is the behaviour you want when
 * the wording changes in a way that matters, and the reason to bump it only
 * when it does.
 */
export function ConsentGate({
  purpose,
  version,
  title,
  description,
  acceptLabel = "I agree",
  children,
  fallback = null,
  className,
}: ConsentGateProps) {
  const { consents, isLoading, error, reload } = useConsents(purpose);
  const { grant, isSaving, error: grantError } = useGrantConsent();

  const satisfied = React.useMemo(
    () =>
      consents.some(
        (c) =>
          c.purpose === purpose &&
          c.granted &&
          !c.revoked_at &&
          (version === undefined || c.version === version),
      ),
    [consents, purpose, version],
  );

  if (isLoading) return <>{fallback}</>;
  if (satisfied) return <>{children}</>;

  return (
    <div className={cn("rounded-lg border border-input bg-card p-4", className)}>
      <p className="text-sm font-medium">{title}</p>
      {description && (
        <p className="mt-1 text-[13px] leading-relaxed text-muted-foreground">
          {description}
        </p>
      )}

      <ErrorDisplay error={(grantError ?? error)?.message ?? null} />

      <Button
        className="mt-3"
        disabled={isSaving}
        onClick={async () => {
          try {
            await grant({ purpose, version });
            await reload();
          } catch {
            // Surfaced through grantError above.
          }
        }}
      >
        {isSaving ? <LoadingSpinner /> : acceptLabel}
      </Button>
    </div>
  );
}
