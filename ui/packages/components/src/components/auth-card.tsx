import * as React from "react";
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
  /** Optional logo element rendered above the title. */
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

  return (
    <Card
      className={cn(
        "mx-auto w-full max-w-[400px]",
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
        {logo && (
          <div className={cn("mb-3", isCenter && "mx-auto")}>{logo}</div>
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
