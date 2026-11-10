"use client";

import * as React from "react";
import { X } from "lucide-react";
import { cn } from "../lib/utils";
import {
  forgetLastAccount,
  getLastAccountServerSnapshot,
  getLastAccountSnapshot,
  subscribeToLastAccount,
  type LastUsedAccount as LastUsedAccountRecord,
} from "../lib/last-used-account";
import { Avatar, AvatarFallback, AvatarImage } from "../primitives/avatar";
import { Badge } from "../primitives/badge";

export interface LastUsedAccountProps {
  /**
   * The account to show. Left out, the component reads the one this browser
   * remembered, and renders nothing when there is none.
   */
  account?: LastUsedAccountRecord | null;
  /** Clicking the chip. Give it the strategy to resume. */
  onSelect?: (account: LastUsedAccountRecord) => void;
  /** Shown instead of a dismiss control when you want the chip permanent. */
  dismissible?: boolean;
  /** Called after the record is forgotten, so a parent can re-render. */
  onDismiss?: () => void;
  className?: string;
}

function initials(account: LastUsedAccountRecord): string {
  const source = account.displayName?.trim() || account.identifier;
  const parts = source.split(/[\s@._-]+/).filter(Boolean);
  return (parts[0]?.[0] ?? "?").concat(parts[1]?.[0] ?? "").toUpperCase();
}

/**
 * The returning-visitor shortcut: who signed in last on this browser.
 *
 * It renders nothing at all when there is no record, which is every first
 * visit and every browser that refuses to store one, so it is safe to drop
 * at the top of a sign-in form unconditionally.
 */
export function LastUsedAccount({
  account,
  onSelect,
  dismissible = true,
  onDismiss,
  className,
}: LastUsedAccountProps) {
  // localStorage is an external store, so it is read through the API React
  // provides for one. The alternative, setting state from an effect on mount,
  // costs a render pass and is what react-hooks/set-state-in-effect objects
  // to. The server snapshot is always null, so the markup React renders on
  // the server matches the first client render and the chip appears after
  // hydration rather than causing a mismatch.
  const stored = React.useSyncExternalStore(
    subscribeToLastAccount,
    getLastAccountSnapshot,
    getLastAccountServerSnapshot,
  );

  const resolved = account === undefined ? stored : account;
  if (!resolved) return null;

  const handleDismiss = () => {
    forgetLastAccount();
    onDismiss?.();
  };

  return (
    <div
      className={cn(
        "group flex w-full items-center gap-3 rounded-lg border border-input bg-background p-3 text-left transition-colors",
        onSelect && "cursor-pointer hover:bg-accent",
        className,
      )}
    >
      <button
        type="button"
        onClick={onSelect ? () => onSelect(resolved) : undefined}
        disabled={!onSelect}
        className="flex min-w-0 flex-1 items-center gap-3 text-left outline-none focus-visible:ring-[3px] focus-visible:ring-ring/40 rounded-md disabled:cursor-default"
      >
        <Avatar className="h-9 w-9 shrink-0">
          {resolved.avatarUrl && (
            <AvatarImage src={resolved.avatarUrl} alt="" />
          )}
          <AvatarFallback className="text-xs">{initials(resolved)}</AvatarFallback>
        </Avatar>
        <span className="min-w-0 flex-1">
          <span className="block truncate text-sm font-medium">
            {resolved.displayName || resolved.identifier}
          </span>
          <span className="block truncate text-[13px] text-muted-foreground">
            {resolved.displayName ? `${resolved.identifier}` : null}
            {resolved.displayName && resolved.strategyLabel ? " · " : null}
            {resolved.strategyLabel ? `signs in with ${resolved.strategyLabel}` : null}
          </span>
        </span>
      </button>
      <Badge variant="secondary" className="shrink-0 font-normal">
        Last used
      </Badge>
      {dismissible && (
        <button
          type="button"
          onClick={handleDismiss}
          aria-label="Forget this account"
          className="shrink-0 rounded-md p-1 text-muted-foreground opacity-0 transition-opacity outline-none hover:text-foreground focus-visible:opacity-100 focus-visible:ring-[3px] focus-visible:ring-ring/40 group-hover:opacity-100"
        >
          <X className="h-4 w-4" />
        </button>
      )}
    </div>
  );
}
