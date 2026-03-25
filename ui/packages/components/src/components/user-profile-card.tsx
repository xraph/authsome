"use client";

import * as React from "react";
import { useState } from "react";
import { useAuth, useUser } from "@authsome/ui-react";
import { Pencil, Check, X, Mail, CheckCircle2 } from "lucide-react";
import { cn } from "../lib/utils";
import { Button } from "../primitives/button";
import { Input } from "../primitives/input";
import { Label } from "../primitives/label";
import { Badge } from "../primitives/badge";
import {
  Card,
  CardHeader,
  CardContent,
} from "../primitives/card";
import { Avatar, AvatarFallback, AvatarImage } from "../primitives/avatar";
import { Separator } from "../primitives/separator";
import { ErrorDisplay } from "./error-display";
import { LoadingSpinner } from "./loading-spinner";

export interface UserProfileCardProps {
  /** Callback invoked after profile updates. */
  onUpdate?: () => void;
  /** Additional CSS class names. */
  className?: string;
}

function getDisplayName(user: { first_name?: string; last_name?: string }): string {
  return [user.first_name, user.last_name].filter(Boolean).join(" ");
}

/**
 * Displays the current user's profile with inline editing.
 *
 * Shows avatar, name, email, username. Allows editing first/last name and username
 * using `client.updateMe()`.
 */
export function UserProfileCard({
  onUpdate,
  className,
}: UserProfileCardProps) {
  const { client, session } = useAuth();
  const { user, reload } = useUser();

  const [isEditing, setIsEditing] = useState(false);
  const [editFirstName, setEditFirstName] = useState("");
  const [editLastName, setEditLastName] = useState("");
  const [editUsername, setEditUsername] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [isSaving, setIsSaving] = useState(false);

  const startEditing = () => {
    setEditFirstName(user?.first_name ?? "");
    setEditLastName(user?.last_name ?? "");
    setEditUsername(user?.username ?? "");
    setError(null);
    setIsEditing(true);
  };

  const cancelEditing = () => {
    setIsEditing(false);
    setError(null);
  };

  const handleSave = async () => {
    setError(null);
    setIsSaving(true);

    try {
      if (session) {
        await client.updateMe(
          {
            first_name: editFirstName || undefined,
            last_name: editLastName || undefined,
            username: editUsername || undefined,
          },
          session.session_token,
        );
      }
      await reload();
      setIsEditing(false);
      onUpdate?.();
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to update profile",
      );
    } finally {
      setIsSaving(false);
    }
  };

  if (!user) {
    return null;
  }

  const displayName = getDisplayName(user);
  const initials = user.first_name
    ? user.first_name[0].toUpperCase()
    : user.email?.[0]?.toUpperCase() ?? "?";

  return (
    <Card className={cn("mx-auto w-full max-w-md", className)}>
      <CardHeader className="flex flex-row items-start justify-between space-y-0 pb-4">
        <div className="flex items-center gap-4">
          <Avatar className="h-16 w-16">
            {user.image && <AvatarImage src={user.image} alt={displayName || user.email} />}
            <AvatarFallback className="text-lg">{initials}</AvatarFallback>
          </Avatar>
          <div className="space-y-1">
            <h3 className="text-lg font-semibold leading-none">{displayName || "Unnamed"}</h3>
            <div className="flex items-center gap-1.5">
              <Mail className="h-3.5 w-3.5 text-muted-foreground" />
              <span className="text-sm text-muted-foreground">{user.email}</span>
            </div>
            {user.email_verified && (
              <Badge variant="secondary" className="gap-1 text-xs">
                <CheckCircle2 className="h-3 w-3" />
                Verified
              </Badge>
            )}
          </div>
        </div>
        {!isEditing && (
          <Button variant="ghost" size="icon" onClick={startEditing}>
            <Pencil className="h-4 w-4" />
          </Button>
        )}
      </CardHeader>

      <Separator />

      <CardContent className="pt-4">
        {isEditing ? (
          <div className="grid gap-4">
            <ErrorDisplay error={error} />

            <div className="grid gap-2">
              <Label htmlFor="profile-first-name">First name</Label>
              <Input
                id="profile-first-name"
                value={editFirstName}
                onChange={(e) => setEditFirstName(e.target.value)}
                disabled={isSaving}
                placeholder="First name"
              />
            </div>

            <div className="grid gap-2">
              <Label htmlFor="profile-last-name">Last name</Label>
              <Input
                id="profile-last-name"
                value={editLastName}
                onChange={(e) => setEditLastName(e.target.value)}
                disabled={isSaving}
                placeholder="Last name"
              />
            </div>

            <div className="grid gap-2">
              <Label htmlFor="profile-username">Username</Label>
              <Input
                id="profile-username"
                value={editUsername}
                onChange={(e) => setEditUsername(e.target.value)}
                disabled={isSaving}
                placeholder="username"
              />
            </div>

            <div className="flex justify-end gap-2">
              <Button
                variant="outline"
                size="sm"
                onClick={cancelEditing}
                disabled={isSaving}
              >
                <X className="mr-1 h-4 w-4" />
                Cancel
              </Button>
              <Button
                size="sm"
                onClick={() => void handleSave()}
                disabled={isSaving}
              >
                {isSaving ? (
                  <LoadingSpinner size="sm" className="mr-1" />
                ) : (
                  <Check className="mr-1 h-4 w-4" />
                )}
                Save
              </Button>
            </div>
          </div>
        ) : (
          <dl className="grid gap-3 text-sm">
            <div className="flex justify-between">
              <dt className="text-muted-foreground">Name</dt>
              <dd className="font-medium">{displayName || "-"}</dd>
            </div>
            {user.username && (
              <div className="flex justify-between">
                <dt className="text-muted-foreground">Username</dt>
                <dd className="font-medium">@{user.username}</dd>
              </div>
            )}
            <div className="flex justify-between">
              <dt className="text-muted-foreground">Email</dt>
              <dd className="font-medium">{user.email}</dd>
            </div>
            {user.phone && (
              <div className="flex justify-between">
                <dt className="text-muted-foreground">Phone</dt>
                <dd className="font-medium">{user.phone}</dd>
              </div>
            )}
            <div className="flex justify-between">
              <dt className="text-muted-foreground">Member since</dt>
              <dd className="font-medium">
                {new Date(user.created_at).toLocaleDateString()}
              </dd>
            </div>
          </dl>
        )}
      </CardContent>
    </Card>
  );
}
