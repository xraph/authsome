"use client";

import * as React from "react";
import { AuthContext } from "@authsome/ui-react";
import { cn } from "../lib/utils";
import {
  Card,
  CardHeader,
  CardTitle,
  CardDescription,
  CardContent,
  CardFooter,
} from "../primitives/card";

export type AuthCardAlign = "center" | "left";
export type AuthCardVariant = "default" | "flat" | "bordered" | "borderless";

export interface AuthCardProps {
  title: string;
  description?: string;
  /**
   * Optional logo element rendered above the title. When omitted, the card
   * falls back to `branding.logo_url` from the client config. Pass `null` to
   * render no logo at all.
   */
  logo?: React.ReactNode;
  footer?: React.ReactNode;
  /** Title and description alignment (only affects title/description, footer is always centered). */
  align?: AuthCardAlign;
  /** Card visual style. */
  variant?: AuthCardVariant;
  className?: string;
  children: React.ReactNode;
}

const variantClasses: Record<AuthCardVariant, string> = {
  default: "border-border/40 shadow-sm",
  flat: "border-border/40 shadow-none",
  bordered: "border-border shadow-none",
  borderless: "border-transparent shadow-none",
};

export function AuthCard({
  title,
  description,
  logo,
  footer,
  align = "center",
  variant = "default",
  className,
  children,
}: AuthCardProps) {
  const isCenter = align === "center";

  // Read the context directly rather than through useClientConfig, which
  // throws outside an AuthProvider. The card is also rendered standalone
  // (Storybook, host apps composing their own screens).
  const branding = React.useContext(AuthContext)?.clientConfig?.branding;
  const resolvedLogo =
    logo === undefined && branding?.logo_url ? (
      <img
        src={branding.logo_url}
        alt={branding.app_name ?? ""}
        className="h-8 w-auto"
      />
    ) : (
      logo
    );

  return (
    <Card
      className={cn(
        // Definite width so the card never sizes to its content — otherwise a
        // shrink-to-fit/centered parent makes each step (email vs. password)
        // a different width. max-w-full keeps it from overflowing small screens.
        "mx-auto w-[400px] max-w-full",
        variantClasses[variant],
        className,
      )}
    >
      <CardHeader
        className={cn(
          "space-y-1 px-7 pb-6 pt-7",
          isCenter ? "text-center" : "text-left",
        )}
      >
        {resolvedLogo && (
          <div className={cn("mb-3", isCenter && "mx-auto")}>
            {resolvedLogo}
          </div>
        )}
        <CardTitle className="text-lg font-semibold tracking-tight">
          {title}
        </CardTitle>
        {description && (
          <CardDescription className="text-[13px] text-muted-foreground">
            {description}
          </CardDescription>
        )}
      </CardHeader>
      <CardContent className="px-7 pb-6">{children}</CardContent>
      {footer && (
        <CardFooter className="flex justify-center text-center px-7 py-4">
          {footer}
        </CardFooter>
      )}
    </Card>
  );
}
