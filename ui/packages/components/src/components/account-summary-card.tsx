"use client";

import * as React from "react";
import { useLastSignIn, type LastSignInSummary } from "@authsome/ui-react";
import { cn } from "../lib/utils";
import { Avatar, AvatarFallback, AvatarImage } from "../primitives/avatar";
import { Badge } from "../primitives/badge";
import { Separator } from "../primitives/separator";
import { Skeleton } from "../primitives/skeleton";

export interface AccountSummaryIdentity {
  name?: string;
  /** Email or phone, shown under the name. */
  identifier: string;
  avatarUrl?: string;
  /** Free text after the identifier, e.g. "Owner of Acme". */
  role?: string;
}

export interface AccountSummaryCardProps {
  /** Who the card is about. */
  identity: AccountSummaryIdentity;
  /**
   * The facts to show. Left out, the card calls useLastSignIn itself, which
   * needs an AuthProvider above it. Pass one to render from your own data.
   */
  summary?: LastSignInSummary | null;
  /** Heading inside the card. Pass null for none. */
  title?: React.ReactNode;
  /** Rendered under the rows, for a "sign out everywhere" pointer. */
  footer?: React.ReactNode;
  className?: string;
}

/** "Today 08:41", "Yesterday 23:02", "4 Mar 2026 09:15". */
function formatSignInTime(iso: string | null): string | null {
  if (!iso) return null;
  const at = new Date(iso);
  if (Number.isNaN(at.getTime())) return null;

  const time = at.toLocaleTimeString(undefined, {
    hour: "2-digit",
    minute: "2-digit",
    hour12: false,
  });

  // Compare calendar days rather than elapsed hours. Something at 00:30 is
  // "today" even though it was forty minutes ago, and something at 23:50
  // yesterday is "yesterday" even though it was ten minutes ago.
  const startOfDay = (d: Date) =>
    new Date(d.getFullYear(), d.getMonth(), d.getDate()).getTime();
  const days = Math.round((startOfDay(new Date()) - startOfDay(at)) / 86_400_000);

  if (days === 0) return `Today ${time}`;
  if (days === 1) return `Yesterday ${time}`;
  return `${at.toLocaleDateString(undefined, { day: "numeric", month: "short", year: "numeric" })} ${time}`;
}

function Row({
  label,
  children,
}: {
  label: string;
  children: React.ReactNode;
}) {
  return (
    <div className="flex items-center justify-between gap-4 py-2.5 text-[13px]">
      <span className="shrink-0 text-muted-foreground">{label}</span>
      <span className="min-w-0 truncate text-right font-medium">{children}</span>
    </div>
  );
}

function initials(identity: AccountSummaryIdentity): string {
  const source = identity.name?.trim() || identity.identifier;
  const parts = source.split(/[\s@._-]+/).filter(Boolean);
  return (parts[0]?.[0] ?? "?").concat(parts[1]?.[0] ?? "").toUpperCase();
}

/**
 * Who you are and when you were last here.
 *
 * This is reassurance, not authorization. It tells someone looking at a
 * two-step prompt or a password reset that the account is the one they think
 * it is, and nothing it shows decides what anyone is allowed to do.
 */
export function AccountSummaryCard({
  identity,
  summary,
  title = "Your account",
  footer,
  className,
}: AccountSummaryCardProps) {
  // The hook runs either way, because hooks cannot be called conditionally.
  // Its result is ignored when summary was passed, and the cost is one fetch
  // a host composing the card from its own data did not need. Worth it to
  // keep the component usable both ways.
  const hook = useLastSignIn();
  const resolved = summary === undefined ? hook.summary : summary;
  const isLoading = summary === undefined && hook.isLoading;

  const signedInAt = formatSignInTime(resolved?.lastSignInAt ?? null);
  const where = resolved?.ipAddress;

  return (
    <div
      className={cn(
        "w-full rounded-xl border border-border/60 bg-card p-5 shadow-sm",
        className,
      )}
    >
      {title && (
        <h3 className="mb-4 text-[13px] font-medium uppercase tracking-wide text-muted-foreground">
          {title}
        </h3>
      )}

      <div className="flex items-center gap-3">
        <Avatar className="h-10 w-10 shrink-0">
          {identity.avatarUrl && <AvatarImage src={identity.avatarUrl} alt="" />}
          <AvatarFallback className="text-xs">{initials(identity)}</AvatarFallback>
        </Avatar>
        <div className="min-w-0 flex-1">
          <p className="truncate text-sm font-semibold">
            {identity.name || identity.identifier}
          </p>
          <p className="truncate text-[13px] text-muted-foreground">
            {identity.name ? identity.identifier : null}
            {identity.name && identity.role ? " · " : null}
            {identity.role}
          </p>
        </div>
      </div>

      <Separator className="my-4" />

      {isLoading ? (
        <div className="space-y-3 py-1">
          <Skeleton className="h-3.5 w-full" />
          <Skeleton className="h-3.5 w-4/5" />
          <Skeleton className="h-3.5 w-2/3" />
        </div>
      ) : (
        <div className="divide-y divide-border/60">
          {signedInAt && (
            <Row label="Last sign-in">
              {where ? `${signedInAt} · ${where}` : signedInAt}
            </Row>
          )}
          {resolved?.device && <Row label="Device">{resolved.device}</Row>}
          {/* null means no endpoint answered yet, so the row stays off rather
              than claiming two-step is off when nobody asked. */}
          {resolved?.twoStepEnabled !== null &&
            resolved?.twoStepEnabled !== undefined && (
              <Row label="Two-step sign-in">
                <Badge
                  variant={resolved.twoStepEnabled ? "secondary" : "outline"}
                  className="font-normal"
                >
                  {resolved.twoStepEnabled ? "On" : "Off"}
                </Badge>
              </Row>
            )}
          {resolved && <Row label="Trusted devices">{resolved.trustedDeviceCount}</Row>}
        </div>
      )}

      {footer && (
        <p className="mt-4 text-[13px] leading-relaxed text-muted-foreground">
          {footer}
        </p>
      )}
    </div>
  );
}
