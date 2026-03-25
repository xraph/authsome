import * as React from "react";
import { useUser } from "@authsome/ui-react";
import { cn } from "../lib/utils";
import { Avatar, AvatarImage, AvatarFallback } from "../primitives/avatar";

export interface UserAvatarProps {
  size?: "sm" | "md" | "lg";
  className?: string;
}

const sizeClasses = {
  sm: "h-8 w-8",
  md: "h-10 w-10",
  lg: "h-12 w-12",
} as const;

function getDisplayName(user: { first_name?: string; last_name?: string; email?: string } | null): string | undefined {
  if (!user) return undefined;
  const parts = [user.first_name, user.last_name].filter(Boolean);
  return parts.length > 0 ? parts.join(" ") : undefined;
}

function getInitials(user: { first_name?: string; last_name?: string; email?: string } | null): string {
  if (!user) return "?";
  if (user.first_name) {
    return user.first_name.charAt(0).toUpperCase();
  }
  if (user.email) {
    return user.email.charAt(0).toUpperCase();
  }
  return "?";
}

export function UserAvatar({ size = "md", className }: UserAvatarProps) {
  const { user } = useUser();

  return (
    <Avatar className={cn(sizeClasses[size], className)}>
      {user?.image && (
        <AvatarImage src={user.image} alt={getDisplayName(user) || user.email || "User"} />
      )}
      <AvatarFallback className="text-xs font-medium">
        {getInitials(user)}
      </AvatarFallback>
    </Avatar>
  );
}
