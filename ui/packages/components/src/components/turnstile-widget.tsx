"use client";

import * as React from "react";

/**
 * Cloudflare Turnstile widget.
 *
 * Loads the official Turnstile script (once per page) and renders an
 * invisible-to-managed challenge. Calls `onToken` when a token is
 * obtained, and `onExpire` when the token expires so the host form can
 * gate submission until a fresh token arrives.
 *
 * The widget is cleaned up on unmount via `turnstile.remove(widgetId)`.
 */

const SCRIPT_SRC =
  "https://challenges.cloudflare.com/turnstile/v0/api.js?render=explicit";

type TurnstileGlobal = {
  render: (
    container: HTMLElement,
    options: {
      sitekey: string;
      callback: (token: string) => void;
      "error-callback"?: () => void;
      "expired-callback"?: () => void;
      theme?: "light" | "dark" | "auto";
      action?: string;
    },
  ) => string;
  remove: (widgetId: string) => void;
  reset: (widgetId?: string) => void;
};

declare global {
  interface Window {
    turnstile?: TurnstileGlobal;
  }
}

let scriptPromise: Promise<void> | null = null;

function loadScript(): Promise<void> {
  if (typeof window === "undefined") {
    return Promise.reject(new Error("Turnstile requires a browser"));
  }
  if (window.turnstile) {
    return Promise.resolve();
  }
  if (scriptPromise) {
    return scriptPromise;
  }
  scriptPromise = new Promise<void>((resolve, reject) => {
    const existing = document.querySelector<HTMLScriptElement>(
      `script[src^="https://challenges.cloudflare.com/turnstile/v0/api.js"]`,
    );
    if (existing) {
      existing.addEventListener("load", () => resolve(), { once: true });
      existing.addEventListener(
        "error",
        () => reject(new Error("Failed to load Turnstile")),
        { once: true },
      );
      return;
    }
    const script = document.createElement("script");
    script.src = SCRIPT_SRC;
    script.async = true;
    script.defer = true;
    script.onload = () => resolve();
    script.onerror = () => reject(new Error("Failed to load Turnstile"));
    document.head.appendChild(script);
  });
  return scriptPromise;
}

export interface TurnstileWidgetProps {
  /** Cloudflare Turnstile site key. */
  siteKey: string;
  /** Called with a token whenever the challenge completes. */
  onToken: (token: string) => void;
  /** Called when an existing token expires. */
  onExpire?: () => void;
  /** Called when the challenge errors. */
  onError?: () => void;
  /** Optional theme override. */
  theme?: "light" | "dark" | "auto";
  /** Action label sent to the backend (defaults to "auth"). */
  action?: string;
  /** Additional CSS class names for the container. */
  className?: string;
}

export function TurnstileWidget({
  siteKey,
  onToken,
  onExpire,
  onError,
  theme = "auto",
  action,
  className,
}: TurnstileWidgetProps) {
  const containerRef = React.useRef<HTMLDivElement | null>(null);
  const widgetIdRef = React.useRef<string | null>(null);

  // Stash callbacks so we don't re-render the widget every time a parent
  // closure is recreated.
  const onTokenRef = React.useRef(onToken);
  const onExpireRef = React.useRef(onExpire);
  const onErrorRef = React.useRef(onError);
  React.useEffect(() => {
    onTokenRef.current = onToken;
    onExpireRef.current = onExpire;
    onErrorRef.current = onError;
  }, [onToken, onExpire, onError]);

  React.useEffect(() => {
    let cancelled = false;
    loadScript()
      .then(() => {
        if (cancelled || !containerRef.current || !window.turnstile) return;
        widgetIdRef.current = window.turnstile.render(containerRef.current, {
          sitekey: siteKey,
          theme,
          action,
          callback: (token) => onTokenRef.current?.(token),
          "expired-callback": () => onExpireRef.current?.(),
          "error-callback": () => onErrorRef.current?.(),
        });
      })
      .catch(() => {
        onErrorRef.current?.();
      });

    return () => {
      cancelled = true;
      if (widgetIdRef.current && window.turnstile) {
        try {
          window.turnstile.remove(widgetIdRef.current);
        } catch {
          // Ignore — widget already torn down.
        }
        widgetIdRef.current = null;
      }
    };
  }, [siteKey, theme, action]);

  return <div ref={containerRef} className={className} />;
}
