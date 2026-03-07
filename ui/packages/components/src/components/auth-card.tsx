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

export interface AuthCardProps {
  title: string;
  description?: string;
  /** Optional logo element rendered above the title. */
  logo?: React.ReactNode;
  footer?: React.ReactNode;
  className?: string;
  children: React.ReactNode;
}

export function AuthCard({
  title,
  description,
  logo,
  footer,
  className,
  children,
}: AuthCardProps) {
  return (
    <Card
      className={cn(
        "mx-auto w-full max-w-[400px] border-border/40 shadow-sm",
        className,
      )}
    >
      <CardHeader className="space-y-1 px-7 pb-4 pt-7 text-center">
        {logo && <div className="mx-auto mb-3">{logo}</div>}
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
        <CardFooter className="flex justify-center border-t border-border/40 px-7 py-4 text-center">
          {footer}
        </CardFooter>
      )}
    </Card>
  );
}
