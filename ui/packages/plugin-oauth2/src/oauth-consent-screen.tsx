"use client";

import * as React from "react";
import { Check } from "lucide-react";
import {
  AuthCard,
  Button,
  ErrorDisplay,
  LoadingSpinner,
  Skeleton,
  type AuthCardAlign,
  type AuthCardVariant,
} from "@authsome/ui-components";
import { DEFAULT_SCOPE_LABELS } from "./index";
import { useConsentDecision, useConsentRequest } from "./react";

export interface OAuthConsentScreenProps {
  /** The `cid` query parameter the authorize endpoint redirected with. */
  cid: string | null;
  /**
   * Readable names for scopes, merged over the built-in map. Anything still
   * unknown falls back to the raw scope string, because asking someone to
   * approve a blank line is worse than showing them "custom:widgets".
   */
  scopeLabels?: Record<string, string>;
  /**
   * Where to send the browser after a decision. Defaults to assigning
   * `window.location.href`, which is what a real authorize flow needs.
   */
  onDecided?: (redirectUrl: string, decision: "allow" | "deny") => void;
  align?: AuthCardAlign;
  variant?: AuthCardVariant;
  className?: string;
}

/**
 * The "App X wants access to your account" screen.
 *
 * Deny is a real button rather than a link out, because refusing is a
 * decision the server needs to hear: it is what sends `access_denied` back
 * to the client instead of leaving the request hanging.
 */
export function OAuthConsentScreen({
  cid,
  scopeLabels,
  onDecided,
  align,
  variant,
  className,
}: OAuthConsentScreenProps) {
  const { request, isLoading, error } = useConsentRequest(cid);
  const { decide, isDeciding, error: decisionError } = useConsentDecision();
  const [choice, setChoice] = React.useState<"allow" | "deny" | null>(null);

  const labels = React.useMemo(
    () => ({ ...DEFAULT_SCOPE_LABELS, ...scopeLabels }),
    [scopeLabels],
  );

  const submit = async (decision: "allow" | "deny") => {
    if (!request) return;
    setChoice(decision);
    try {
      const res = await decide({
        cid: request.cid,
        csrf: request.csrf,
        decision,
      });
      if (onDecided) onDecided(res.redirect_url, decision);
      else globalThis.location?.assign(res.redirect_url);
    } catch {
      // Surfaced through decisionError below.
    } finally {
      setChoice(null);
    }
  };

  if (isLoading) {
    return (
      <AuthCard title="One moment" align={align} variant={variant} className={className}>
        <div className="space-y-3">
          <Skeleton className="h-4 w-2/3" />
          <Skeleton className="h-4 w-1/2" />
          <Skeleton className="h-4 w-3/5" />
        </div>
      </AuthCard>
    );
  }

  if (!request) {
    return (
      <AuthCard
        title="This request is not valid"
        description="It may have expired. Start again from the app that sent you here."
        align={align}
        variant={variant}
        className={className}
      >
        <ErrorDisplay error={error?.message ?? null} />
      </AuthCard>
    );
  }

  return (
    <AuthCard
      title={`${request.client_name} wants access`}
      description="Check what it is asking for before you allow it."
      align={align}
      variant={variant}
      className={className}
    >
      <div className="grid gap-4">
        <ul className="grid gap-2">
          {request.scopes.map((scope) => (
            <li key={scope} className="flex items-start gap-2 text-[13px]">
              <Check className="mt-0.5 h-3.5 w-3.5 shrink-0 text-muted-foreground" />
              <span>{labels[scope] ?? scope}</span>
            </li>
          ))}
        </ul>

        <ErrorDisplay error={decisionError?.message ?? null} />

        <div className="grid gap-2">
          <Button
            className="w-full"
            disabled={isDeciding}
            onClick={() => void submit("allow")}
          >
            {isDeciding && choice === "allow" ? <LoadingSpinner /> : "Allow"}
          </Button>
          <Button
            variant="outline"
            className="w-full"
            disabled={isDeciding}
            onClick={() => void submit("deny")}
          >
            {isDeciding && choice === "deny" ? <LoadingSpinner /> : "Deny"}
          </Button>
        </div>
      </div>
    </AuthCard>
  );
}
