"use client";

import * as React from "react";
import {
  Badge,
  Button,
  ErrorDisplay,
  Input,
  Label,
  LoadingSpinner,
  Skeleton,
  cn,
} from "@authsome/ui-components";
import { isTokenExpired, type ScimTokenCreated } from "./index";
import {
  useGenerateScimToken,
  useRevokeScimToken,
  useRotateScimToken,
  useScimTokens,
} from "./react";

export interface ScimTokenListProps {
  configId: string | null;
  title?: React.ReactNode;
  emptyState?: React.ReactNode;
  /** Seconds the replaced token keeps working after a rotate. */
  rotateGraceSeconds?: number;
  className?: string;
}

function formatDate(iso?: string): string {
  if (!iso) return "never";
  const d = new Date(iso);
  return Number.isNaN(d.getTime())
    ? "never"
    : d.toLocaleDateString(undefined, {
        day: "numeric",
        month: "short",
        year: "numeric",
      });
}

/**
 * Shows a freshly minted token, once.
 *
 * It is deliberately loud and deliberately not dismissible by accident: the
 * plaintext exists nowhere else, and someone who closes this without copying
 * it has to rotate to get another.
 */
function NewTokenBanner({
  created,
  onDone,
}: {
  created: ScimTokenCreated;
  onDone: () => void;
}) {
  const [copied, setCopied] = React.useState(false);
  return (
    <div className="mb-4 rounded-lg border border-primary/40 bg-primary/5 p-3">
      <p className="text-sm font-medium">Copy this token now</p>
      <p className="mt-0.5 text-[13px] text-muted-foreground">
        It will not be shown again. If you lose it, rotate to get another.
      </p>
      <div className="mt-2 flex items-center gap-2">
        <Input
          readOnly
          value={created.plaintext}
          className="font-mono"
          onFocus={(e) => e.currentTarget.select()}
        />
        <Button
          type="button"
          variant="outline"
          size="sm"
          className="shrink-0"
          onClick={async () => {
            try {
              await globalThis.navigator?.clipboard?.writeText(created.plaintext);
              setCopied(true);
            } catch {
              // The value is on screen and selectable either way.
            }
          }}
        >
          {copied ? "Copied" : "Copy"}
        </Button>
      </div>
      <Button type="button" size="sm" className="mt-3" onClick={onDone}>
        I've saved it
      </Button>
    </div>
  );
}

/** A connection's tokens, with mint, rotate and revoke. */
export function ScimTokenList({
  configId,
  title = "Tokens",
  emptyState = "No tokens yet. Create one to connect your identity provider.",
  rotateGraceSeconds = 3600,
  className,
}: ScimTokenListProps) {
  const { tokens, isLoading, error, reload } = useScimTokens(configId);
  const { generate, isSaving: isGenerating, error: generateError } =
    useGenerateScimToken();
  const { rotate, isSaving: isRotating, error: rotateError } =
    useRotateScimToken();
  const { revoke, isSaving: isRevoking, error: revokeError } =
    useRevokeScimToken();

  const [name, setName] = React.useState("");
  const [created, setCreated] = React.useState<ScimTokenCreated | null>(null);
  const [pending, setPending] = React.useState<string | null>(null);
  const [confirmRevoke, setConfirmRevoke] = React.useState<string | null>(null);

  const anyError = generateError ?? rotateError ?? revokeError ?? error;

  return (
    <div className={cn("w-full", className)}>
      {title && (
        <h3 className="mb-3 text-[13px] font-medium uppercase tracking-wide text-muted-foreground">
          {title}
        </h3>
      )}

      <ErrorDisplay error={anyError?.message ?? null} />

      {created && (
        <NewTokenBanner created={created} onDone={() => setCreated(null)} />
      )}

      <form
        className="mb-4 flex items-end gap-2"
        onSubmit={async (e) => {
          e.preventDefault();
          if (!configId || !name.trim()) return;
          try {
            setCreated(await generate(configId, { name: name.trim() }));
            setName("");
            await reload();
          } catch {
            // Surfaced through generateError.
          }
        }}
      >
        <div className="grid flex-1 gap-1.5">
          <Label htmlFor="authsome-scim-token-name">New token</Label>
          <Input
            id="authsome-scim-token-name"
            placeholder="Okta production"
            value={name}
            onChange={(e) => setName(e.target.value)}
            disabled={isGenerating}
          />
        </div>
        <Button type="submit" disabled={!name.trim() || isGenerating}>
          {isGenerating ? <LoadingSpinner /> : "Create"}
        </Button>
      </form>

      {isLoading ? (
        <div className="space-y-3 py-2">
          <Skeleton className="h-4 w-2/3" />
          <Skeleton className="h-4 w-1/2" />
        </div>
      ) : tokens.length === 0 ? (
        <p className="py-2 text-[13px] text-muted-foreground">{emptyState}</p>
      ) : (
        <ul className="divide-y divide-border/60">
          {tokens.map((t) => {
            const expired = isTokenExpired(t);
            return (
              <li key={t.id} className="py-3">
                <div className="flex items-center justify-between gap-4">
                  <div className="min-w-0 flex-1">
                    <p className="truncate text-sm font-medium">{t.name}</p>
                    <p className="text-[13px] text-muted-foreground">
                      Last used {formatDate(t.last_used_at)} · expires{" "}
                      {formatDate(t.expires_at)}
                    </p>
                  </div>
                  <div className="flex shrink-0 items-center gap-2">
                    {expired && (
                      <Badge variant="destructive" className="font-normal">
                        expired
                      </Badge>
                    )}
                    <Button
                      variant="outline"
                      size="sm"
                      disabled={isRotating && pending === t.id}
                      onClick={async () => {
                        setPending(t.id);
                        try {
                          setCreated(
                            await rotate(t.id, {
                              name: t.name,
                              grace_seconds: rotateGraceSeconds,
                            }),
                          );
                          await reload();
                        } catch {
                          // Surfaced through rotateError.
                        } finally {
                          setPending(null);
                        }
                      }}
                    >
                      {isRotating && pending === t.id ? (
                        <LoadingSpinner />
                      ) : (
                        "Rotate"
                      )}
                    </Button>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => setConfirmRevoke(t.id)}
                    >
                      Revoke
                    </Button>
                  </div>
                </div>

                {confirmRevoke === t.id && (
                  <div className="mt-2 rounded-lg border border-input p-3">
                    <p className="text-[13px]">
                      Revoke {t.name}? Any identity provider using it stops
                      provisioning straight away.
                    </p>
                    <div className="mt-2 flex gap-2">
                      <Button
                        size="sm"
                        variant="destructive"
                        disabled={isRevoking}
                        onClick={async () => {
                          try {
                            await revoke(t.id);
                            await reload();
                          } catch {
                            // Surfaced through revokeError.
                          } finally {
                            setConfirmRevoke(null);
                          }
                        }}
                      >
                        {isRevoking ? <LoadingSpinner /> : "Yes, revoke"}
                      </Button>
                      <Button
                        size="sm"
                        variant="ghost"
                        onClick={() => setConfirmRevoke(null)}
                      >
                        Keep it
                      </Button>
                    </div>
                  </div>
                )}
              </li>
            );
          })}
        </ul>
      )}
    </div>
  );
}
