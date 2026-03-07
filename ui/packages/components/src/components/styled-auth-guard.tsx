import * as React from "react";
import { useAuth } from "@authsome/ui-react";
import { cn } from "../lib/utils";
import { Skeleton } from "../primitives/skeleton";

export interface StyledAuthGuardProps {
  children: React.ReactNode;
  fallback?: React.ReactNode;
  className?: string;
}

function AuthGuardSkeleton({ className }: { className?: string }) {
  return (
    <div className={cn("flex items-center justify-center", className)}>
      <div className="h-[400px] w-full max-w-md mx-auto rounded-lg border border-border bg-card p-6">
        <div className="space-y-6">
          {/* Header skeleton */}
          <div className="space-y-2 text-center">
            <Skeleton className="mx-auto h-7 w-48" />
            <Skeleton className="mx-auto h-4 w-64" />
          </div>

          {/* Input skeletons */}
          <div className="space-y-4">
            <div className="space-y-2">
              <Skeleton className="h-4 w-16" />
              <Skeleton className="h-10 w-full" />
            </div>
            <div className="space-y-2">
              <Skeleton className="h-4 w-20" />
              <Skeleton className="h-10 w-full" />
            </div>
          </div>

          {/* Button skeleton */}
          <Skeleton className="h-10 w-full" />
        </div>
      </div>
    </div>
  );
}

export function StyledAuthGuard({
  children,
  fallback,
  className,
}: StyledAuthGuardProps) {
  const { state } = useAuth();

  if (state.status === "loading" || state.status === "idle") {
    return <AuthGuardSkeleton className={cn(className)} />;
  }

  if (state.status !== "authenticated") {
    return fallback ? <>{fallback}</> : null;
  }

  return <>{children}</>;
}
