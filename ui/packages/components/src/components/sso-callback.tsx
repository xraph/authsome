"use client";

import * as React from "react";
import { useEffect, useRef, useState, useSyncExternalStore } from "react";
import { useAuth } from "@authsome/ui-react";
import { redirectAfterAuth } from "../lib/redirect-after-auth";
import { cn } from "../lib/utils";
import { subscribeToPopState } from "../lib/pop-state";
import { AuthCard, type AuthCardAlign, type AuthCardVariant } from "./auth-card";
import { ErrorDisplay } from "./error-display";
import { LoadingSpinner } from "./loading-spinner";

export interface SSOCallbackProps {
  /**
   * Called once the SSO code has been exchanged for a session. When omitted,
   * the browser is redirected to the `redirect` query param (same-origin
   * only) or "/".
   */
  onSuccess?: () => void;
  /** Called when the IdP reported an error, the code is missing, or the exchange fails. */
  onError?: (error: Error) => void;
  /** The one-time SSO code. Read from `?code=` in the URL when omitted. */
  code?: string;
  /** An SSO failure reason. Read from `?sso_error=` in the URL when omitted. */
  error?: string;
  /** URL to the sign-in page. Renders a "Back to sign in" link when sign-on fails. */
  signInUrl?: string;
  /** Optional logo element rendered above the title. */
  logo?: React.ReactNode;
  /** Title and description alignment. */
  align?: AuthCardAlign;
  /** Card visual style. */
  variant?: AuthCardVariant;
  /** Additional CSS class names. */
  className?: string;
}

const readSearch = () => window.location.search;

/**
 * Landing page for the SSO round trip.
 *
 * After the IdP signs the user in, the backend redirects here with
 * `?code=<one-time code>` (or `?sso_error=<reason>` on failure). This
 * component exchanges the code for a session exactly once, then calls
 * `onSuccess` or redirects.
 *
 * ```tsx
 * // app/sso/callback/page.tsx
 * export default function Page() {
 *   return <SSOCallback />;
 * }
 * ```
 */
export function SSOCallback({
  onSuccess,
  onError,
  code: codeProp,
  error: errorProp,
  signInUrl,
  logo,
  align,
  variant,
  className,
}: SSOCallbackProps) {
  const { completeSSOLogin } = useAuth();

  // The location is an external store. The server snapshot is null, so SSR
  // and the hydrating render both show the pending card and only the client
  // decides what the URL says.
  const search = useSyncExternalStore(
    subscribeToPopState,
    readSearch,
    () => null,
  );

  const params = search === null ? null : new URLSearchParams(search);
  const code = codeProp ?? params?.get("code") ?? null;
  const reportedError = errorProp ?? params?.get("sso_error") ?? null;
  // Until there is either an explicit prop or a URL to read, nothing is known.
  const resolved =
    codeProp !== undefined || errorProp !== undefined || params !== null;

  // Failures known without a network round trip.
  const immediateError = !resolved
    ? null
    : reportedError
      ? reportedError
      : !code
        ? "The sign-in response did not include an authorization code."
        : null;

  const [exchangeError, setExchangeError] = useState<string | null>(null);
  // The code is single use: exchanging it twice fails the second time, so
  // StrictMode's double-invoked effect must not send it again.
  const handledRef = useRef(false);

  useEffect(() => {
    if (!resolved || handledRef.current) return;
    handledRef.current = true;

    if (immediateError) {
      onError?.(new Error(immediateError));
      return;
    }

    completeSSOLogin(code!).then(
      () => {
        if (onSuccess) {
          onSuccess();
          return;
        }
        redirectAfterAuth();
      },
      (err: unknown) => {
        const failure =
          err instanceof Error ? err : new Error("Single sign-on failed.");
        setExchangeError(
          failure.message || "Could not complete single sign-on.",
        );
        onError?.(failure);
      },
    );
  }, [resolved, immediateError, code, completeSSOLogin, onSuccess, onError]);

  const failure = immediateError ?? exchangeError;

  if (failure) {
    const footer = signInUrl ? (
      <a
        href={signInUrl}
        className="text-[13px] font-medium text-foreground underline-offset-4 hover:underline"
      >
        Back to sign in
      </a>
    ) : undefined;

    return (
      <AuthCard
        title="Single sign-on failed"
        description="We couldn't sign you in with your identity provider."
        logo={logo}
        footer={footer}
        align={align}
        variant={variant}
        className={cn(className)}
      >
        <ErrorDisplay error={failure} />
      </AuthCard>
    );
  }

  return (
    <AuthCard
      title="Signing you in…"
      description="Completing single sign-on."
      logo={logo}
      align={align}
      variant={variant}
      className={cn(className)}
    >
      <div className="flex justify-center py-4">
        <LoadingSpinner />
      </div>
    </AuthCard>
  );
}
