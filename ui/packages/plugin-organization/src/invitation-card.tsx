"use client";

import * as React from "react";
import {
  AuthCard,
  Avatar,
  AvatarFallback,
  Badge,
  Button,
  ErrorDisplay,
  LoadingSpinner,
  Skeleton,
  type AuthCardAlign,
  type AuthCardVariant,
} from "@authsome/ui-components";
import { expiresIn } from "./index";
import {
  useAcceptInvitation,
  useDeclineInvitation,
  useInvitation,
} from "./react";

export interface InvitationCardProps {
  /** The invitation token, usually from the URL. */
  token: string | null;
  /**
   * Rendered instead of the accept button when nobody is signed in. Give it
   * your sign-in and sign-up controls: accepting needs a session, and this
   * card deliberately does not try to create one.
   */
  signInSlot?: React.ReactNode;
  onAccepted?: (orgId: string) => void;
  onDeclined?: () => void;
  align?: AuthCardAlign;
  variant?: AuthCardVariant;
  className?: string;
  /** Readable names for roles, keyed by role string. */
  roleLabels?: Record<string, string>;
  /** True when a session exists, so the card knows which controls to show. */
  isSignedIn?: boolean;
}

function initials(name: string): string {
  const parts = name.split(/[\s@._-]+/).filter(Boolean);
  return (parts[0]?.[0] ?? "?").concat(parts[1]?.[0] ?? "").toUpperCase();
}

/**
 * The page someone lands on from an invitation email.
 *
 * It shows who invited them, to what, as what, and when it runs out, before
 * asking for anything. Accepting needs a session, so when nobody is signed
 * in the card renders `signInSlot` instead of the accept button rather than
 * trying to build a sign-in flow of its own.
 */
export function InvitationCard({
  token,
  signInSlot,
  onAccepted,
  onDeclined,
  align,
  variant,
  className,
  roleLabels,
  isSignedIn = false,
}: InvitationCardProps) {
  const { invitation, isLoading, error } = useInvitation(token);
  const { accept, isSaving: isAccepting, error: acceptError } =
    useAcceptInvitation();
  const { decline, isSaving: isDeclining, error: declineError } =
    useDeclineInvitation();
  const [done, setDone] = React.useState<"accepted" | "declined" | null>(null);

  if (isLoading) {
    return (
      <AuthCard title="One moment" align={align} variant={variant} className={className}>
        <div className="space-y-3">
          <Skeleton className="h-4 w-2/3" />
          <Skeleton className="h-4 w-1/2" />
        </div>
      </AuthCard>
    );
  }

  if (!invitation) {
    return (
      <AuthCard
        title="This invitation is not valid"
        description="It may have been used already, or it may have expired. Ask whoever invited you to send another."
        align={align}
        variant={variant}
        className={className}
      >
        <ErrorDisplay error={error?.message ?? null} />
      </AuthCard>
    );
  }

  if (done === "accepted") {
    return (
      <AuthCard
        title={`You're in`}
        description={`You joined ${invitation.org_name}.`}
        align={align}
        variant={variant}
        className={className}
      >
        <span />
      </AuthCard>
    );
  }

  if (done === "declined") {
    return (
      <AuthCard
        title="Invitation declined"
        description={`You turned down the invitation to ${invitation.org_name}.`}
        align={align}
        variant={variant}
        className={className}
      >
        <span />
      </AuthCard>
    );
  }

  const expiry = expiresIn(invitation.expires_at);
  const role = roleLabels?.[invitation.role] ?? invitation.role;

  return (
    <AuthCard
      title={`Join ${invitation.org_name}`}
      description={`You'll join as a ${role}.`}
      align={align}
      variant={variant}
      className={className}
    >
      <div className="grid gap-4">
        <div className="flex items-center gap-2.5">
          <Avatar className="h-7 w-7 shrink-0">
            <AvatarFallback className="text-[11px]">
              {initials(invitation.inviter_name)}
            </AvatarFallback>
          </Avatar>
          <p className="min-w-0 flex-1 truncate text-[13px] text-muted-foreground">
            {invitation.inviter_name} invited {invitation.email_masked}
          </p>
          <Badge
            variant={expiry === "expired" ? "destructive" : "secondary"}
            className="shrink-0 font-normal"
          >
            {expiry === "expired" ? "expired" : `expires ${expiry}`}
          </Badge>
        </div>

        <ErrorDisplay error={(acceptError ?? declineError)?.message ?? null} />

        {isSignedIn ? (
          <Button
            className="w-full"
            disabled={isAccepting || expiry === "expired"}
            onClick={async () => {
              if (!token) return;
              try {
                await accept(token);
                setDone("accepted");
                onAccepted?.(invitation.org_id);
              } catch {
                // Surfaced through acceptError.
              }
            }}
          >
            {isAccepting ? <LoadingSpinner /> : "Accept and join"}
          </Button>
        ) : (
          signInSlot
        )}

        <button
          type="button"
          className="text-[13px] text-muted-foreground underline underline-offset-4 hover:text-foreground disabled:opacity-50"
          disabled={isDeclining}
          onClick={async () => {
            if (!token) return;
            try {
              await decline(token);
              setDone("declined");
              onDeclined?.();
            } catch {
              // Surfaced through declineError.
            }
          }}
        >
          {isDeclining ? "Declining..." : "Not expecting this? Decline the invite"}
        </button>
      </div>
    </AuthCard>
  );
}
