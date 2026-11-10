"use client";

import * as React from "react";
import {
  Badge,
  Button,
  ErrorDisplay,
  Input,
  Label,
  LoadingSpinner,
  Skeleton,
  cn,
} from "@authsome/ui-components";
import { expiresIn } from "./index";
import { useCreateInvitation, useInvitations } from "./react";

export interface InvitationListProps {
  orgId: string | null;
  /** Roles a new invitation can be sent for. */
  roles?: { value: string; label: string }[];
  title?: React.ReactNode;
  emptyState?: React.ReactNode;
  /** Hides the invite form, for a read-only view. */
  readOnly?: boolean;
  className?: string;
}

/** Pending invitations, and a form to send another. */
export function InvitationList({
  orgId,
  roles,
  title = "Invitations",
  emptyState = "No pending invitations.",
  readOnly = false,
  className,
}: InvitationListProps) {
  const { invitations, isLoading, error, reload } = useInvitations(orgId);
  const { invite, isSaving, error: inviteError } = useCreateInvitation(orgId);
  const [email, setEmail] = React.useState("");
  const [role, setRole] = React.useState(roles?.[0]?.value ?? "");

  const pending = invitations.filter((i) => i.status === "pending");

  return (
    <div className={cn("w-full", className)}>
      {title && (
        <h3 className="mb-3 text-[13px] font-medium uppercase tracking-wide text-muted-foreground">
          {title}
        </h3>
      )}

      <ErrorDisplay error={(error ?? inviteError)?.message ?? null} />

      {!readOnly && (
        <form
          className="mb-4 flex items-end gap-2"
          onSubmit={async (e) => {
            e.preventDefault();
            if (!email.trim()) return;
            try {
              await invite({ email: email.trim(), role: role || undefined });
              setEmail("");
              await reload();
            } catch {
              // Surfaced through inviteError above.
            }
          }}
        >
          <div className="grid flex-1 gap-1.5">
            <Label htmlFor="authsome-invite-email">Invite someone</Label>
            <Input
              id="authsome-invite-email"
              type="email"
              autoComplete="email"
              placeholder="them@company.com"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              disabled={isSaving}
            />
          </div>
          {roles && roles.length > 0 && (
            <select
              aria-label="Role"
              className="h-[30px] rounded-md border border-input bg-background px-2 text-[13px] outline-none focus-visible:border-ring focus-visible:ring-[3px] focus-visible:ring-ring/40"
              value={role}
              onChange={(e) => setRole(e.target.value)}
              disabled={isSaving}
            >
              {roles.map((r) => (
                <option key={r.value} value={r.value}>
                  {r.label}
                </option>
              ))}
            </select>
          )}
          <Button type="submit" disabled={!email.trim() || isSaving}>
            {isSaving ? <LoadingSpinner /> : "Send"}
          </Button>
        </form>
      )}

      {isLoading ? (
        <div className="space-y-3 py-2">
          <Skeleton className="h-4 w-2/3" />
          <Skeleton className="h-4 w-1/2" />
        </div>
      ) : pending.length === 0 ? (
        <p className="py-2 text-[13px] text-muted-foreground">{emptyState}</p>
      ) : (
        <ul className="divide-y divide-border/60">
          {pending.map((inv) => {
            const expiry = expiresIn(inv.expires_at);
            return (
              <li
                key={inv.id}
                className="flex items-center justify-between gap-4 py-3"
              >
                <div className="min-w-0 flex-1">
                  <p className="truncate text-sm font-medium">{inv.email}</p>
                  <p className="text-[13px] text-muted-foreground">{inv.role}</p>
                </div>
                <Badge
                  variant={expiry === "expired" ? "destructive" : "outline"}
                  className="shrink-0 font-normal"
                >
                  {expiry === "expired" ? "expired" : `expires ${expiry}`}
                </Badge>
              </li>
            );
          })}
        </ul>
      )}
    </div>
  );
}
