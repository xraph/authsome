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
  // Borderless is the shape the refreshed design leads with: no chrome of its
  // own, so it sits flush inside a column the host app laid out.
  borderless: "border-transparent bg-transparent shadow-none",
};

// Borderless drops the horizontal padding too. A card with no border that
// still inset its contents by 28px would look misaligned against anything
// else in the host's column.
const padding: Record<AuthCardVariant, { header: string; content: string; footer: string }> = {
  default: { header: "px-7 pt-7 pb-6", content: "px-7 pb-6", footer: "px-7 py-4" },
  flat: { header: "px-7 pt-7 pb-6", content: "px-7 pb-6", footer: "px-7 py-4" },
  bordered: { header: "px-7 pt-7 pb-6", content: "px-7 pb-6", footer: "px-7 py-4" },
  borderless: { header: "px-0 pt-0 pb-7", content: "px-0 pb-0", footer: "px-0 pt-6" },
};

export function AuthCard({
  title,
  description,
  logo,
  footer,
  align = "left",
  variant = "default",
  className,
  children,
}: AuthCardProps) {
  const isCenter = align === "center";
  const pad = padding[variant];

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
        // Definite width so the card never sizes to its content. Otherwise a
        // shrink-to-fit or centered parent makes each step (email vs. password)
        // a different width. max-w-full keeps it from overflowing small screens.
        "mx-auto w-[400px] max-w-full",
        variantClasses[variant],
        className,
      )}
    >
      <CardHeader
        className={cn("space-y-2", pad.header, isCenter ? "text-center" : "text-left")}
      >
        {resolvedLogo && (
          <div className={cn("mb-4", isCenter && "mx-auto")}>{resolvedLogo}</div>
        )}
        <CardTitle className="text-[32px] font-bold leading-[1.15] tracking-tight">
          {title}
        </CardTitle>
        {description && (
          <CardDescription className="text-[15px] leading-relaxed text-muted-foreground">
            {description}
          </CardDescription>
        )}
      </CardHeader>
      <CardContent className={pad.content}>{children}</CardContent>
      {footer && (
        <CardFooter className={cn("flex justify-center text-center", pad.footer)}>
          {footer}
        </CardFooter>
      )}
    </Card>
  );
}
