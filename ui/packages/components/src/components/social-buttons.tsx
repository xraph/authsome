import * as React from "react";
import { cn } from "../lib/utils";
import { Button } from "../primitives/button";
import { Separator } from "../primitives/separator";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "../primitives/tooltip";
import { Globe } from "lucide-react";

export interface SocialProvider {
  id: string;
  name: string;
  icon?: React.ReactNode;
}

/** Layout mode for social login buttons. */
export type SocialButtonLayout = "grid" | "icon-row" | "vertical";

export interface SocialButtonsProps {
  providers: SocialProvider[];
  onProviderClick: (providerId: string) => void;
  isLoading?: boolean;
  /** Layout mode: "grid" (default 2-col), "icon-row" (icons only horizontal), "vertical" (stacked full-width). */
  layout?: SocialButtonLayout;
  /** Whether to show the "or" divider above the buttons. */
  showDivider?: boolean;
  className?: string;
}

/* ── Brand SVG icons ─────────────────────────────────── */

function GoogleIcon({ className }: { className?: string }) {
  return (
    <svg className={className} viewBox="0 0 24 24" fill="none">
      <path d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92a5.06 5.06 0 0 1-2.2 3.32v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.1z" fill="#4285F4" />
      <path d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z" fill="#34A853" />
      <path d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z" fill="#FBBC05" />
      <path d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z" fill="#EA4335" />
    </svg>
  );
}

function GitHubIcon({ className }: { className?: string }) {
  return (
    <svg className={className} viewBox="0 0 24 24" fill="currentColor">
      <path d="M12 2C6.477 2 2 6.484 2 12.017c0 4.425 2.865 8.18 6.839 9.504.5.092.682-.217.682-.483 0-.237-.008-.868-.013-1.703-2.782.605-3.369-1.343-3.369-1.343-.454-1.158-1.11-1.466-1.11-1.466-.908-.62.069-.608.069-.608 1.003.07 1.531 1.032 1.531 1.032.892 1.53 2.341 1.088 2.91.832.092-.647.35-1.088.636-1.338-2.22-.253-4.555-1.113-4.555-4.951 0-1.093.39-1.988 1.029-2.688-.103-.253-.446-1.272.098-2.65 0 0 .84-.27 2.75 1.026A9.564 9.564 0 0 1 12 6.844a9.59 9.59 0 0 1 2.504.337c1.909-1.296 2.747-1.027 2.747-1.027.546 1.379.202 2.398.1 2.651.64.7 1.028 1.595 1.028 2.688 0 3.848-2.339 4.695-4.566 4.943.359.309.678.92.678 1.855 0 1.338-.012 2.419-.012 2.747 0 .268.18.58.688.482A10.02 10.02 0 0 0 22 12.017C22 6.484 17.522 2 12 2z" />
    </svg>
  );
}

function AppleIcon({ className }: { className?: string }) {
  return (
    <svg className={className} viewBox="0 0 24 24" fill="currentColor">
      <path d="M17.05 20.28c-.98.95-2.05.88-3.08.4-1.09-.5-2.08-.48-3.24 0-1.44.62-2.2.44-3.06-.4C2.79 15.25 3.51 7.59 9.05 7.31c1.35.07 2.29.74 3.08.8 1.18-.24 2.31-.93 3.57-.84 1.51.12 2.65.72 3.4 1.8-3.12 1.87-2.38 5.98.48 7.13-.57 1.5-1.31 2.99-2.54 4.09zM12.03 7.25c-.15-2.23 1.66-4.07 3.74-4.25.29 2.58-2.34 4.5-3.74 4.25z" />
    </svg>
  );
}

function MicrosoftIcon({ className }: { className?: string }) {
  return (
    <svg className={className} viewBox="0 0 24 24" fill="none">
      <rect x="2" y="2" width="9.5" height="9.5" fill="#F25022" />
      <rect x="12.5" y="2" width="9.5" height="9.5" fill="#7FBA00" />
      <rect x="2" y="12.5" width="9.5" height="9.5" fill="#00A4EF" />
      <rect x="12.5" y="12.5" width="9.5" height="9.5" fill="#FFB900" />
    </svg>
  );
}

function TwitterIcon({ className }: { className?: string }) {
  return (
    <svg className={className} viewBox="0 0 24 24" fill="currentColor">
      <path d="M18.244 2.25h3.308l-7.227 8.26 8.502 11.24H16.17l-5.214-6.817L4.99 21.75H1.68l7.73-8.835L1.254 2.25H8.08l4.713 6.231zm-1.161 17.52h1.833L7.084 4.126H5.117z" />
    </svg>
  );
}

const BRAND_ICONS: Record<string, React.ComponentType<{ className?: string }>> = {
  google: GoogleIcon,
  github: GitHubIcon,
  apple: AppleIcon,
  microsoft: MicrosoftIcon,
  twitter: TwitterIcon,
  x: TwitterIcon,
};

function ProviderIcon({ provider }: { provider: SocialProvider }) {
  if (provider.icon) {
    return (
      <span className="inline-flex h-4 w-4 items-center justify-center">
        {provider.icon}
      </span>
    );
  }

  const BrandIcon = BRAND_ICONS[provider.id.toLowerCase()];
  if (BrandIcon) {
    return <BrandIcon className="h-4 w-4" />;
  }

  return <Globe className="h-4 w-4" />;
}

export function OrDivider({ className }: { className?: string }) {
  return (
    <div className={cn("relative my-4", className)}>
      <div className="absolute inset-0 flex items-center">
        <Separator className="w-full" />
      </div>
      <div className="relative flex justify-center text-xs uppercase">
        <span className="bg-card px-2 text-muted-foreground">or</span>
      </div>
    </div>
  );
}

function GridLayout({
  providers,
  onProviderClick,
  isLoading,
}: {
  providers: SocialProvider[];
  onProviderClick: (id: string) => void;
  isLoading: boolean;
}) {
  return (
    <div
      className={cn(
        "grid gap-2",
        providers.length === 1 ? "grid-cols-1" : "grid-cols-2",
      )}
    >
      {providers.map((provider) => (
        <Button
          key={provider.id}
          variant="outline"
          size="default"
          type="button"
          disabled={isLoading}
          className="w-full gap-2 text-[13px] font-normal"
          onClick={() => onProviderClick(provider.id)}
        >
          <ProviderIcon provider={provider} />
          {provider.name}
        </Button>
      ))}
    </div>
  );
}

function IconRowLayout({
  providers,
  onProviderClick,
  isLoading,
}: {
  providers: SocialProvider[];
  onProviderClick: (id: string) => void;
  isLoading: boolean;
}) {
  return (
    <TooltipProvider delayDuration={300}>
      <div className="flex flex-row flex-wrap items-center justify-center gap-2">
        {providers.map((provider) => (
          <Tooltip key={provider.id}>
            <TooltipTrigger asChild>
              <Button
                variant="outline"
                size="icon"
                type="button"
                disabled={isLoading}
                className="h-[30px] w-[30px]"
                onClick={() => onProviderClick(provider.id)}
              >
                <ProviderIcon provider={provider} />
                <span className="sr-only">{provider.name}</span>
              </Button>
            </TooltipTrigger>
            <TooltipContent>
              <p>{provider.name}</p>
            </TooltipContent>
          </Tooltip>
        ))}
      </div>
    </TooltipProvider>
  );
}

function VerticalLayout({
  providers,
  onProviderClick,
  isLoading,
}: {
  providers: SocialProvider[];
  onProviderClick: (id: string) => void;
  isLoading: boolean;
}) {
  return (
    <div className="flex flex-col gap-2">
      {providers.map((provider) => (
        <Button
          key={provider.id}
          variant="outline"
          size="default"
          type="button"
          disabled={isLoading}
          className="w-full gap-2 text-[13px] font-normal"
          onClick={() => onProviderClick(provider.id)}
        >
          <ProviderIcon provider={provider} />
          {provider.name}
        </Button>
      ))}
    </div>
  );
}

export function SocialButtons({
  providers,
  onProviderClick,
  isLoading = false,
  layout = "grid",
  showDivider = true,
  className,
}: SocialButtonsProps) {
  if (providers.length === 0) {
    return null;
  }

  return (
    <div className={cn(className)}>
      {showDivider && <OrDivider />}
      {layout === "icon-row" ? (
        <IconRowLayout
          providers={providers}
          onProviderClick={onProviderClick}
          isLoading={isLoading}
        />
      ) : layout === "vertical" ? (
        <VerticalLayout
          providers={providers}
          onProviderClick={onProviderClick}
          isLoading={isLoading}
        />
      ) : (
        <GridLayout
          providers={providers}
          onProviderClick={onProviderClick}
          isLoading={isLoading}
        />
      )}
    </div>
  );
}
