"use client";

import * as React from "react";
import {
  Badge,
  Button,
  ErrorDisplay,
  Input,
  Label,
  LoadingSpinner,
  Separator,
  Skeleton,
  Switch,
  cn,
} from "@authsome/ui-components";
import type { ScimConfig } from "./index";
import { useScimConfigs, useUpdateScimConfig } from "./react";

export interface ScimConnectionCardProps {
  orgId: string | null;
  /** Which connection to show. Defaults to the first one. */
  configId?: string;
  title?: React.ReactNode;
  /** Rendered under the settings, usually the token list. */
  children?: React.ReactNode;
  /** Shown when the org has no connection yet. */
  emptyState?: React.ReactNode;
  className?: string;
}

function Row({ label, hint, children }: {
  label: string;
  hint?: string;
  children: React.ReactNode;
}) {
  return (
    <div className="flex items-start justify-between gap-4 py-3">
      <div className="min-w-0">
        <p className="text-sm font-medium">{label}</p>
        {hint && <p className="mt-0.5 text-[13px] text-muted-foreground">{hint}</p>}
      </div>
      <div className="shrink-0">{children}</div>
    </div>
  );
}

/**
 * One identity provider connection: the base URL to paste into the IdP, and
 * the switches that decide what a push is allowed to do.
 *
 * The base URL sits at the top with a copy button because it is the thing
 * someone is on this page to fetch. Everything else is settings they will
 * touch once.
 */
export function ScimConnectionCard({
  orgId,
  configId,
  title = "Provisioning",
  children,
  emptyState = "No identity provider is connected yet.",
  className,
}: ScimConnectionCardProps) {
  const { configs, isLoading, error, reload } = useScimConfigs(orgId);
  const { update, isSaving, error: updateError } = useUpdateScimConfig();
  const [copied, setCopied] = React.useState(false);

  const config: ScimConfig | undefined = configId
    ? configs.find((c) => c.id === configId)
    : configs[0];

  const toggle = async (field: keyof ScimConfig, value: boolean) => {
    if (!config) return;
    try {
      await update(config.id, { [field]: value });
      await reload();
    } catch {
      // Surfaced through updateError below.
    }
  };

  const copy = async (text: string) => {
    try {
      await globalThis.navigator?.clipboard?.writeText(text);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      // A browser that refuses clipboard access still shows the URL, which
      // is selectable. Nothing to report.
    }
  };

  return (
    <div
      className={cn(
        "w-full rounded-xl border border-border/60 bg-card p-5 shadow-sm",
        className,
      )}
    >
      {title && (
        <h3 className="mb-4 text-[13px] font-medium uppercase tracking-wide text-muted-foreground">
          {title}
        </h3>
      )}

      <ErrorDisplay error={(error ?? updateError)?.message ?? null} />

      {isLoading ? (
        <div className="space-y-3">
          <Skeleton className="h-4 w-1/2" />
          <Skeleton className="h-4 w-2/3" />
        </div>
      ) : !config ? (
        <p className="text-[13px] text-muted-foreground">{emptyState}</p>
      ) : (
        <>
          <div className="flex items-center justify-between gap-3">
            <div className="min-w-0">
              <p className="truncate font-semibold">{config.name}</p>
              <p className="mt-0.5 text-[13px] text-muted-foreground">
                Provisioned users join as {config.default_role || "member"}
              </p>
            </div>
            <Badge
              variant={config.enabled ? "secondary" : "outline"}
              className="shrink-0 font-normal"
            >
              {config.enabled ? "Connected" : "Paused"}
            </Badge>
          </div>

          <Separator className="my-4" />

          <div className="grid gap-1.5">
            <Label htmlFor="authsome-scim-url">SCIM base URL</Label>
            <div className="flex items-center gap-2">
              <Input
                id="authsome-scim-url"
                readOnly
                value={config.id ? "/scim/v2" : ""}
                className="font-mono"
                onFocus={(e) => e.currentTarget.select()}
              />
              <Button
                type="button"
                variant="outline"
                size="sm"
                className="shrink-0"
                onClick={() => void copy("/scim/v2")}
              >
                {copied ? "Copied" : "Copy"}
              </Button>
            </div>
            <p className="text-[13px] text-muted-foreground">
              Paste this and a token into your identity provider.
            </p>
          </div>

          <Separator className="my-4" />

          <div className="divide-y divide-border/60">
            <Row
              label="Accept provisioning"
              hint="Turn this off to stop the identity provider changing anything."
            >
              <Switch
                checked={config.enabled}
                disabled={isSaving}
                onCheckedChange={(v) => void toggle("enabled", v)}
              />
            </Row>
            <Row
              label="Create users"
              hint="Someone pushed for the first time gets an account."
            >
              <Switch
                checked={config.auto_create}
                disabled={isSaving}
                onCheckedChange={(v) => void toggle("auto_create", v)}
              />
            </Row>
            <Row
              label="Suspend on deactivate"
              hint="Someone deactivated in the identity provider loses access here."
            >
              <Switch
                checked={config.auto_suspend}
                disabled={isSaving}
                onCheckedChange={(v) => void toggle("auto_suspend", v)}
              />
            </Row>
            <Row label="Sync groups to teams">
              <Switch
                checked={config.group_sync}
                disabled={isSaving}
                onCheckedChange={(v) => void toggle("group_sync", v)}
              />
            </Row>
          </div>

          {isSaving && (
            <p className="mt-3 flex items-center gap-2 text-[13px] text-muted-foreground">
              <LoadingSpinner /> Saving
            </p>
          )}

          {children && (
            <>
              <Separator className="my-4" />
              {children}
            </>
          )}
        </>
      )}
    </div>
  );
}
