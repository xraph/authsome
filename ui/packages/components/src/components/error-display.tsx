import * as React from "react";
import { cn } from "../lib/utils";
import { AlertCircle } from "lucide-react";

export interface ErrorDisplayProps {
  error: string | null;
  className?: string;
}

/**
 * Compact, Clerk-style inline error display for auth forms.
 *
 * Renders a small red banner with an icon and message text.
 * Designed to sit inside form layouts without taking too much space.
 */
export function ErrorDisplay({ error, className }: ErrorDisplayProps) {
  if (!error) {
    return null;
  }

  return (
    <div
      role="alert"
      className={cn(
        "flex items-center gap-2 rounded-md border border-destructive/30 bg-destructive/5 px-3 py-2.5 text-[13px] text-destructive",
        className,
      )}
    >
      <AlertCircle className="h-3.5 w-3.5 shrink-0" />
      <span>{error}</span>
    </div>
  );
}
