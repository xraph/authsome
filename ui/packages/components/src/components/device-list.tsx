"use client";

import * as React from "react";
import { useState, useEffect, useCallback } from "react";
import { useAuth } from "@authsome/ui-react";
import {
  Monitor,
  Smartphone,
  Tablet,
  Trash2,
  ShieldCheck,
  ShieldOff,
  Globe,
} from "lucide-react";
import { cn } from "../lib/utils";
import { Button } from "../primitives/button";
import { Badge } from "../primitives/badge";
import { Skeleton } from "../primitives/skeleton";
import { Separator } from "../primitives/separator";
import {
  Card,
  CardHeader,
  CardTitle,
  CardDescription,
  CardContent,
} from "../primitives/card";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "../primitives/dialog";
import { ErrorDisplay } from "./error-display";

type Device = {
  id: string;
  name?: string;
  browser?: string;
  os?: string;
  ip_address?: string;
  last_seen_at: string;
  trusted: boolean;
  type?: string;
  created_at: string;
  user_id: string;
  app_id: string;
  fingerprint?: string;
  updated_at: string;
};

export interface DeviceListProps {
  className?: string;
  onTrust?: (deviceId: string) => void;
  onDelete?: (deviceId: string) => void;
}

function timeAgo(dateStr: string): string {
  const seconds = Math.floor(
    (Date.now() - new Date(dateStr).getTime()) / 1000,
  );

  if (seconds < 60) return "just now";
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m ago`;
  if (seconds < 86400) return `${Math.floor(seconds / 3600)}h ago`;
  if (seconds < 2592000) return `${Math.floor(seconds / 86400)}d ago`;
  return `${Math.floor(seconds / 2592000)}mo ago`;
}

function DeviceIcon({ type }: { type?: string }): React.ReactElement {
  const className = "h-5 w-5 text-muted-foreground";

  switch (type) {
    case "desktop":
      return <Monitor className={className} />;
    case "mobile":
      return <Smartphone className={className} />;
    case "tablet":
      return <Tablet className={className} />;
    default:
      return <Globe className={className} />;
  }
}

/**
 * Displays the current user's recognized devices with trust and delete actions.
 */
export function DeviceList({
  className,
  onTrust,
  onDelete,
}: DeviceListProps): React.ReactElement {
  const { client, session } = useAuth();

  const [devices, setDevices] = useState<Device[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [actionLoading, setActionLoading] = useState<string | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<Device | null>(null);

  const token = session?.session_token;

  const fetchDevices = useCallback(async () => {
    if (!token) return;

    setIsLoading(true);
    setError(null);

    try {
      const response = await client.listDevices(token);
      setDevices((response.devices ?? []) as unknown as Device[]);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to load devices",
      );
    } finally {
      setIsLoading(false);
    }
  }, [client, token]);

  useEffect(() => {
    void fetchDevices();
  }, [fetchDevices]);

  const handleTrust = async (device: Device) => {
    if (!token) return;

    setActionLoading(device.id);

    try {
      await client.trustDevice(device.id, { DeviceID: device.id }, token);
      await fetchDevices();
      onTrust?.(device.id);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to update device trust",
      );
    } finally {
      setActionLoading(null);
    }
  };

  const handleDelete = async (device: Device) => {
    if (!token) return;

    setActionLoading(device.id);
    setDeleteTarget(null);

    try {
      await client.deleteDevice(device.id, { DeviceID: device.id }, token);
      await fetchDevices();
      onDelete?.(device.id);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to remove device",
      );
    } finally {
      setActionLoading(null);
    }
  };

  return (
    <>
      <Card className={cn("mx-auto w-full max-w-md", className)}>
        <CardHeader>
          <CardTitle>Devices</CardTitle>
          <CardDescription>Manage your trusted devices</CardDescription>
        </CardHeader>

        <CardContent className="grid gap-1">
          <ErrorDisplay error={error} className="mb-3" />

          {isLoading ? (
            <div className="grid gap-4">
              {Array.from({ length: 3 }).map((_, i) => (
                <div key={i} className="flex items-center gap-3">
                  <Skeleton className="h-10 w-10 rounded-lg" />
                  <div className="flex-1 space-y-2">
                    <Skeleton className="h-4 w-32" />
                    <Skeleton className="h-3 w-48" />
                  </div>
                </div>
              ))}
            </div>
          ) : devices.length === 0 ? (
            <div className="flex flex-col items-center gap-2 py-8 text-center">
              <Monitor className="h-10 w-10 text-muted-foreground/50" />
              <p className="text-sm text-muted-foreground">No devices found</p>
            </div>
          ) : (
            <div className="grid gap-1">
              {devices.map((device, index) => (
                <React.Fragment key={device.id}>
                  {index > 0 && <Separator />}
                  <div className="flex items-center gap-3 py-3">
                    <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-muted">
                      <DeviceIcon type={device.type} />
                    </div>

                    <div className="min-w-0 flex-1">
                      <p className="truncate text-sm font-medium">
                        {device.name || device.browser || "Unknown device"}
                      </p>
                      <p className="truncate text-xs text-muted-foreground">
                        {[device.os, device.browser]
                          .filter(Boolean)
                          .join(" \u00B7 ")}
                      </p>
                      {device.ip_address && (
                        <p className="text-xs text-muted-foreground/70">
                          {device.ip_address}
                        </p>
                      )}
                      <p className="text-xs text-muted-foreground/70">
                        Last seen {timeAgo(device.last_seen_at)}
                      </p>
                    </div>

                    <div className="flex shrink-0 items-center gap-2">
                      {device.trusted ? (
                        <Badge
                          variant="secondary"
                          className="bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400"
                        >
                          Trusted
                        </Badge>
                      ) : (
                        <Badge variant="outline">Untrusted</Badge>
                      )}

                      <Button
                        variant="ghost"
                        size="icon"
                        disabled={actionLoading === device.id}
                        onClick={() => void handleTrust(device)}
                        title={device.trusted ? "Untrust device" : "Trust device"}
                      >
                        {device.trusted ? (
                          <ShieldOff className="h-4 w-4" />
                        ) : (
                          <ShieldCheck className="h-4 w-4" />
                        )}
                      </Button>

                      <Button
                        variant="ghost"
                        size="icon"
                        disabled={actionLoading === device.id}
                        onClick={() => setDeleteTarget(device)}
                        title="Remove device"
                      >
                        <Trash2 className="h-4 w-4 text-destructive" />
                      </Button>
                    </div>
                  </div>
                </React.Fragment>
              ))}
            </div>
          )}
        </CardContent>
      </Card>

      <Dialog
        open={deleteTarget !== null}
        onOpenChange={(open) => {
          if (!open) setDeleteTarget(null);
        }}
      >
        <DialogContent className="sm:max-w-sm">
          <DialogHeader>
            <DialogTitle>Remove device</DialogTitle>
            <DialogDescription>
              Are you sure you want to remove{" "}
              <span className="font-medium text-foreground">
                {deleteTarget?.name ||
                  deleteTarget?.browser ||
                  "this device"}
              </span>
              ? It will need to be re-authorized on next login.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeleteTarget(null)}>
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={() => {
                if (deleteTarget) void handleDelete(deleteTarget);
              }}
            >
              Remove
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
