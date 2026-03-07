"use client";

import * as React from "react";
import { useState } from "react";
import { useAuth, useOrganizations } from "@authsome/ui-react";
import { ChevronsUpDown, Plus, Check, Building2 } from "lucide-react";
import { cn } from "../lib/utils";
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuLabel,
} from "../primitives/dropdown-menu";
import { Button } from "../primitives/button";
import { Avatar, AvatarFallback, AvatarImage } from "../primitives/avatar";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "../primitives/dialog";
import { Input } from "../primitives/input";
import { Label } from "../primitives/label";
import { LoadingSpinner } from "./loading-spinner";

export interface OrgSwitcherProps {
  /** Called when the user selects a different organization. */
  onOrgChange?: (orgId: string) => void;
  /** Called when the user creates a new organization. If not provided, the built-in create dialog is used. */
  onCreateOrg?: () => void;
  /** The currently selected organization ID. */
  activeOrgId?: string;
  /** Additional CSS class names. */
  className?: string;
}

/**
 * OrgSwitcher displays the current organization and allows switching between
 * organizations or creating a new one.
 */
export function OrgSwitcher({
  onOrgChange,
  onCreateOrg,
  activeOrgId,
  className,
}: OrgSwitcherProps) {
  const { client, session } = useAuth();
  const { organizations: orgsRaw, isLoading, reload } = useOrganizations();
  const organizations = orgsRaw ?? [];
  const [createOpen, setCreateOpen] = useState(false);
  const [newOrgName, setNewOrgName] = useState("");
  const [newOrgSlug, setNewOrgSlug] = useState("");
  const [isCreating, setIsCreating] = useState(false);
  const [createError, setCreateError] = useState<string | null>(null);

  const activeOrg = organizations.find(
    (o) => String(o.id) === activeOrgId,
  );

  const handleCreate = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setCreateError(null);
    setIsCreating(true);

    try {
      if (session) {
        await client.createOrganization(
          { name: newOrgName, slug: newOrgSlug },
          session.session_token,
        );
      }
      setCreateOpen(false);
      setNewOrgName("");
      setNewOrgSlug("");
      void reload();
    } catch (err) {
      setCreateError(
        err instanceof Error ? err.message : "Failed to create organization",
      );
    } finally {
      setIsCreating(false);
    }
  };

  const handleNameChange = (value: string) => {
    setNewOrgName(value);
    setNewOrgSlug(
      value
        .toLowerCase()
        .replace(/[^a-z0-9]+/g, "-")
        .replace(/^-|-$/g, ""),
    );
  };

  const initials = (name: string) =>
    name
      .split(" ")
      .map((w) => w[0])
      .join("")
      .slice(0, 2)
      .toUpperCase();

  return (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button
            variant="outline"
            className={cn(
              "flex w-full items-center justify-between gap-2 px-3",
              className,
            )}
          >
            <div className="flex items-center gap-2 truncate">
              <Avatar className="h-5 w-5">
                {activeOrg?.logo && <AvatarImage src={activeOrg.logo} />}
                <AvatarFallback className="text-[10px]">
                  {activeOrg ? initials(activeOrg.name) : <Building2 className="h-3 w-3" />}
                </AvatarFallback>
              </Avatar>
              <span className="truncate text-sm font-medium">
                {activeOrg?.name ?? "Select organization"}
              </span>
            </div>
            <ChevronsUpDown className="ml-auto h-4 w-4 shrink-0 opacity-50" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="start" className="w-[240px]">
          <DropdownMenuLabel className="text-xs text-muted-foreground font-normal">
            Organizations
          </DropdownMenuLabel>
          {isLoading ? (
            <div className="flex items-center justify-center py-4">
              <LoadingSpinner size="sm" />
            </div>
          ) : organizations.length === 0 ? (
            <div className="px-2 py-4 text-center text-sm text-muted-foreground">
              No organizations yet
            </div>
          ) : (
            organizations.map((org) => (
              <DropdownMenuItem
                key={String(org.id)}
                onClick={() => onOrgChange?.(String(org.id))}
                className="flex items-center gap-2"
              >
                <Avatar className="h-5 w-5">
                  {org.logo && <AvatarImage src={org.logo} />}
                  <AvatarFallback className="text-[10px]">
                    {initials(org.name)}
                  </AvatarFallback>
                </Avatar>
                <span className="truncate">{org.name}</span>
                {String(org.id) === activeOrgId && (
                  <Check className="ml-auto h-4 w-4 text-primary" />
                )}
              </DropdownMenuItem>
            ))
          )}
          <DropdownMenuSeparator />
          <DropdownMenuItem
            onClick={() => {
              if (onCreateOrg) {
                onCreateOrg();
              } else {
                setCreateOpen(true);
              }
            }}
            className="flex items-center gap-2"
          >
            <Plus className="h-4 w-4" />
            Create organization
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>

      <Dialog open={createOpen} onOpenChange={setCreateOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Create organization</DialogTitle>
            <DialogDescription>
              Add a new organization to manage your team.
            </DialogDescription>
          </DialogHeader>
          <form onSubmit={handleCreate} className="grid gap-4">
            {createError && (
              <p className="text-sm text-destructive">{createError}</p>
            )}
            <div className="grid gap-2">
              <Label htmlFor="org-name">Name</Label>
              <Input
                id="org-name"
                placeholder="Acme Inc."
                required
                disabled={isCreating}
                value={newOrgName}
                onChange={(e) => handleNameChange(e.target.value)}
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="org-slug">Slug</Label>
              <Input
                id="org-slug"
                placeholder="acme-inc"
                required
                disabled={isCreating}
                value={newOrgSlug}
                onChange={(e) => setNewOrgSlug(e.target.value)}
              />
              <p className="text-xs text-muted-foreground">
                Used in URLs. Only lowercase letters, numbers, and hyphens.
              </p>
            </div>
            <DialogFooter>
              <Button
                type="button"
                variant="outline"
                onClick={() => setCreateOpen(false)}
                disabled={isCreating}
              >
                Cancel
              </Button>
              <Button type="submit" disabled={isCreating}>
                {isCreating && <LoadingSpinner size="sm" className="mr-2" />}
                Create
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </>
  );
}
