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

function FacebookIcon({ className }: { className?: string }) {
  return (
    <svg className={className} viewBox="0 0 24 24" fill="#1877F2">
      <path d="M24 12.073c0-6.627-5.373-12-12-12s-12 5.373-12 12c0 5.99 4.388 10.954 10.125 11.854v-8.385H7.078v-3.47h3.047V9.43c0-3.007 1.792-4.669 4.533-4.669 1.312 0 2.686.235 2.686.235v2.953H15.83c-1.491 0-1.956.925-1.956 1.874v2.25h3.328l-.532 3.47h-2.796v8.385C19.612 23.027 24 18.062 24 12.073z" />
    </svg>
  );
}

function LinkedInIcon({ className }: { className?: string }) {
  return (
    <svg className={className} viewBox="0 0 24 24" fill="#0A66C2">
      <path d="M20.447 20.452h-3.554v-5.569c0-1.328-.027-3.037-1.852-3.037-1.853 0-2.136 1.445-2.136 2.939v5.667H9.351V9h3.414v1.561h.046c.477-.9 1.637-1.85 3.37-1.85 3.601 0 4.267 2.37 4.267 5.455v6.286zM5.337 7.433a2.062 2.062 0 0 1-2.063-2.065 2.064 2.064 0 1 1 2.063 2.065zm1.782 13.019H3.555V9h3.564v11.452zM22.225 0H1.771C.792 0 0 .774 0 1.729v20.542C0 23.227.792 24 1.771 24h20.451C23.2 24 24 23.227 24 22.271V1.729C24 .774 23.2 0 22.222 0h.003z" />
    </svg>
  );
}

function DiscordIcon({ className }: { className?: string }) {
  return (
    <svg className={className} viewBox="0 0 24 24" fill="#5865F2">
      <path d="M20.317 4.37a19.791 19.791 0 0 0-4.885-1.515.074.074 0 0 0-.079.037c-.21.375-.444.864-.608 1.25a18.27 18.27 0 0 0-5.487 0 12.64 12.64 0 0 0-.617-1.25.077.077 0 0 0-.079-.037A19.736 19.736 0 0 0 3.677 4.37a.07.07 0 0 0-.032.027C.533 9.046-.32 13.58.099 18.057a.082.082 0 0 0 .031.057 19.9 19.9 0 0 0 5.993 3.03.078.078 0 0 0 .084-.028c.462-.63.874-1.295 1.226-1.994a.076.076 0 0 0-.041-.106 13.107 13.107 0 0 1-1.872-.892.077.077 0 0 1-.008-.128 10.2 10.2 0 0 0 .372-.292.074.074 0 0 1 .077-.01c3.928 1.793 8.18 1.793 12.062 0a.074.074 0 0 1 .078.01c.12.098.246.198.373.292a.077.077 0 0 1-.006.127 12.299 12.299 0 0 1-1.873.892.077.077 0 0 0-.041.107c.36.698.772 1.362 1.225 1.993a.076.076 0 0 0 .084.028 19.839 19.839 0 0 0 6.002-3.03.077.077 0 0 0 .032-.054c.5-5.177-.838-9.674-3.549-13.66a.061.061 0 0 0-.031-.03zM8.02 15.33c-1.183 0-2.157-1.085-2.157-2.419 0-1.333.956-2.419 2.157-2.419 1.21 0 2.176 1.096 2.157 2.42 0 1.333-.956 2.418-2.157 2.418zm7.975 0c-1.183 0-2.157-1.085-2.157-2.419 0-1.333.955-2.419 2.157-2.419 1.21 0 2.176 1.096 2.157 2.42 0 1.333-.946 2.418-2.157 2.418z" />
    </svg>
  );
}

function SlackIcon({ className }: { className?: string }) {
  return (
    <svg className={className} viewBox="0 0 24 24" fill="none">
      <path d="M5.042 15.165a2.528 2.528 0 0 1-2.52 2.523A2.528 2.528 0 0 1 0 15.165a2.527 2.527 0 0 1 2.522-2.52h2.52v2.52zm1.271 0a2.527 2.527 0 0 1 2.521-2.52 2.527 2.527 0 0 1 2.521 2.52v6.313A2.528 2.528 0 0 1 8.834 24a2.528 2.528 0 0 1-2.521-2.522v-6.313z" fill="#E01E5A" />
      <path d="M8.834 5.042a2.528 2.528 0 0 1-2.521-2.52A2.528 2.528 0 0 1 8.834 0a2.528 2.528 0 0 1 2.521 2.522v2.52H8.834zm0 1.271a2.528 2.528 0 0 1 2.521 2.521 2.528 2.528 0 0 1-2.521 2.521H2.522A2.528 2.528 0 0 1 0 8.834a2.528 2.528 0 0 1 2.522-2.521h6.312z" fill="#36C5F0" />
      <path d="M18.956 8.834a2.528 2.528 0 0 1 2.522-2.521A2.528 2.528 0 0 1 24 8.834a2.528 2.528 0 0 1-2.522 2.521h-2.522V8.834zm-1.27 0a2.528 2.528 0 0 1-2.523 2.521 2.527 2.527 0 0 1-2.52-2.521V2.522A2.527 2.527 0 0 1 15.163 0a2.528 2.528 0 0 1 2.523 2.522v6.312z" fill="#2EB67D" />
      <path d="M15.163 18.956a2.528 2.528 0 0 1 2.523 2.522A2.528 2.528 0 0 1 15.163 24a2.527 2.527 0 0 1-2.52-2.522v-2.522h2.52zm0-1.27a2.527 2.527 0 0 1-2.52-2.523 2.527 2.527 0 0 1 2.52-2.52h6.315A2.528 2.528 0 0 1 24 15.163a2.528 2.528 0 0 1-2.522 2.523h-6.315z" fill="#ECB22E" />
    </svg>
  );
}

function GitLabIcon({ className }: { className?: string }) {
  return (
    <svg className={className} viewBox="0 0 24 24" fill="none">
      <path d="m23.546 10.93-.963-2.962-1.907-5.866a.45.45 0 0 0-.856 0l-1.907 5.866H6.088L4.18 2.102a.45.45 0 0 0-.856 0L1.417 7.968.454 10.93a.896.896 0 0 0 .326 1.003L12 19.93l11.22-7.997a.896.896 0 0 0 .326-1.003" fill="#E24329" />
      <path d="M12 19.93 17.913 7.968H6.088z" fill="#FC6D26" />
      <path d="m12 19.93-5.912-11.962H1.417L12 19.93z" fill="#FCA326" />
      <path d="m12 19.93 5.913-11.962h4.67L12 19.93z" fill="#FCA326" />
    </svg>
  );
}

function SpotifyIcon({ className }: { className?: string }) {
  return (
    <svg className={className} viewBox="0 0 24 24" fill="#1DB954">
      <path d="M12 0C5.4 0 0 5.4 0 12s5.4 12 12 12 12-5.4 12-12S18.66 0 12 0zm5.521 17.34c-.24.359-.66.48-1.021.24-2.82-1.74-6.36-2.101-10.561-1.141-.418.122-.779-.179-.899-.539-.12-.421.18-.78.54-.9 4.56-1.021 8.52-.6 11.64 1.32.42.18.479.659.301 1.02zm1.44-3.3c-.301.42-.841.6-1.262.3-3.239-1.98-8.159-2.58-11.939-1.38-.479.12-1.02-.12-1.14-.6-.12-.48.12-1.021.6-1.141C9.6 9.9 15 10.561 18.72 12.84c.361.181.54.78.241 1.2zm.12-3.36C15.24 8.4 8.82 8.16 5.16 9.301c-.6.179-1.2-.181-1.38-.721-.18-.601.18-1.2.72-1.381 4.26-1.26 11.28-1.02 15.721 1.621.539.3.719 1.02.419 1.56-.299.421-1.02.599-1.559.3z" />
    </svg>
  );
}

function TwitchIcon({ className }: { className?: string }) {
  return (
    <svg className={className} viewBox="0 0 24 24" fill="#9146FF">
      <path d="M11.571 4.714h1.715v5.143H11.57zm4.715 0H18v5.143h-1.714zM6 0 1.714 4.286v15.428h5.143V24l4.286-4.286h3.428L22.286 12V0zm14.571 11.143-3.428 3.428h-3.429l-3 3v-3H6.857V1.714h13.714z" />
    </svg>
  );
}

function AmazonIcon({ className }: { className?: string }) {
  return (
    <svg className={className} viewBox="0 0 24 24" fill="#FF9900">
      <path d="M.045 18.02c.072-.116.187-.124.348-.064 2.729 1.353 5.706 2.03 8.926 2.03 2.235 0 4.407-.373 6.512-1.118.309-.109.576-.165.801-.165.347 0 .594.165.734.495.14.33.07.635-.21.91-.656.625-1.833 1.18-3.529 1.665-1.697.485-3.314.727-4.853.727-2.142 0-4.19-.372-6.143-1.118C.93 20.71.045 19.638.045 18.02zm6.137-7.052c0-1.197.314-2.175.94-2.935.628-.76 1.465-1.14 2.512-1.14 1.092 0 1.947.39 2.566 1.17.62.78.93 1.745.93 2.895 0 1.196-.316 2.18-.95 2.946-.634.766-1.473 1.148-2.516 1.148-1.065 0-1.906-.38-2.524-1.14-.618-.762-.928-1.746-.928-2.944h-.03z" />
      <path d="M21.54 16.943c-.423-.072-.705.089-.846.484-.548 1.523-1.26 2.285-2.136 2.285-.29 0-.543-.1-.756-.3a.98.98 0 0 1-.321-.735c0-.291.096-.695.29-1.212l1.485-3.836c.19-.496.285-.963.285-1.402 0-.653-.224-1.193-.671-1.62-.448-.428-1.009-.642-1.684-.642-.894 0-1.66.405-2.297 1.214h-.03v-1.04h-2.953v8.823h2.982v-4.86c0-.456.132-.85.396-1.185.264-.334.59-.501.98-.501.353 0 .628.127.827.38.2.254.299.594.299 1.02v5.147h2.983v-4.86c0-.91-.228-1.638-.683-2.182h.03c.523-.85 1.213-1.273 2.073-1.273.258 0 .486.05.683.148.198.098.354.232.47.4.117.168.2.364.25.584.05.22.076.455.076.703 0 .456-.086.945-.256 1.468l-1.497 3.87a4.164 4.164 0 0 0-.29 1.438c0 .735.26 1.343.78 1.827.519.483 1.17.725 1.95.725 1.207 0 2.122-.683 2.744-2.048.103-.226.072-.415-.09-.567z" fill="currentColor" />
    </svg>
  );
}

function DropboxIcon({ className }: { className?: string }) {
  return (
    <svg className={className} viewBox="0 0 24 24" fill="#0061FF">
      <path d="m12 6.596-6.86 4.353L12 15.302l6.86-4.353zM5.14 12.083 0 8.364l5.14-3.268L12 8.364zM0 15.636l5.14-3.268L12 15.636 5.14 18.72zm12 0 6.86-3.268L24 15.636 17.14 18.72zM24 8.364l-5.14 3.268L12 8.364l6.86-4.268z" />
    </svg>
  );
}

function BitbucketIcon({ className }: { className?: string }) {
  return (
    <svg className={className} viewBox="0 0 24 24" fill="#2684FF">
      <path d="M.778 1.213a.768.768 0 0 0-.768.892l3.263 19.81c.084.5.515.868 1.022.873H19.95a.772.772 0 0 0 .77-.646l3.27-20.03a.768.768 0 0 0-.768-.891zM14.52 15.53H9.522L8.17 8.466h7.561z" />
    </svg>
  );
}

function ZoomIcon({ className }: { className?: string }) {
  return (
    <svg className={className} viewBox="0 0 24 24" fill="#2D8CFF">
      <path d="M24 12c0 6.627-5.373 12-12 12S0 18.627 0 12 5.373 0 12 0s12 5.373 12 12zM5.2 8.8v4.453c0 1.365 1.106 2.472 2.472 2.472h5.904l2.224-2.224V9.048a.248.248 0 0 0-.248-.248H5.2zm12.296-.247L15.2 10.848v2.304l2.296 2.296a.495.495 0 0 0 .704-.352V8.904a.495.495 0 0 0-.704-.351z" />
    </svg>
  );
}

function PatreonIcon({ className }: { className?: string }) {
  return (
    <svg className={className} viewBox="0 0 24 24" fill="currentColor">
      <path d="M14.82 2.41c3.96 0 7.18 3.24 7.18 7.21 0 3.96-3.22 7.18-7.18 7.18-3.97 0-7.21-3.22-7.21-7.18 0-3.97 3.24-7.21 7.21-7.21M2 21.6h3.5V2.41H2V21.6z" />
    </svg>
  );
}

function InstagramIcon({ className }: { className?: string }) {
  return (
    <svg className={className} viewBox="0 0 24 24" fill="currentColor">
      <path d="M12 2.163c3.204 0 3.584.012 4.85.07 3.252.148 4.771 1.691 4.919 4.919.058 1.265.069 1.645.069 4.849 0 3.205-.012 3.584-.069 4.849-.149 3.225-1.664 4.771-4.919 4.919-1.266.058-1.644.07-4.85.07-3.204 0-3.584-.012-4.849-.07-3.26-.149-4.771-1.699-4.919-4.92-.058-1.265-.07-1.644-.07-4.849 0-3.204.013-3.583.07-4.849.149-3.227 1.664-4.771 4.919-4.919 1.266-.057 1.645-.069 4.849-.069zM12 0C8.741 0 8.333.014 7.053.072 2.695.272.273 2.69.073 7.052.014 8.333 0 8.741 0 12c0 3.259.014 3.668.072 4.948.2 4.358 2.618 6.78 6.98 6.98C8.333 23.986 8.741 24 12 24c3.259 0 3.668-.014 4.948-.072 4.354-.2 6.782-2.618 6.979-6.98.059-1.28.073-1.689.073-4.948 0-3.259-.014-3.667-.072-4.947-.196-4.354-2.617-6.78-6.979-6.98C15.668.014 15.259 0 12 0zm0 5.838a6.162 6.162 0 1 0 0 12.324 6.162 6.162 0 0 0 0-12.324zM12 16a4 4 0 1 1 0-8 4 4 0 0 1 0 8zm6.406-11.845a1.44 1.44 0 1 0 0 2.881 1.44 1.44 0 0 0 0-2.881z" />
    </svg>
  );
}

function PinterestIcon({ className }: { className?: string }) {
  return (
    <svg className={className} viewBox="0 0 24 24" fill="#E60023">
      <path d="M12.017 0C5.396 0 .029 5.367.029 11.987c0 5.079 3.158 9.417 7.618 11.162-.105-.949-.199-2.403.042-3.441.218-.937 1.407-5.965 1.407-5.965s-.359-.719-.359-1.782c0-1.668.967-2.914 2.171-2.914 1.023 0 1.518.769 1.518 1.69 0 1.029-.655 2.568-.994 3.995-.283 1.194.599 2.169 1.777 2.169 2.133 0 3.772-2.249 3.772-5.495 0-2.873-2.064-4.882-5.012-4.882-3.414 0-5.418 2.561-5.418 5.207 0 1.031.397 2.138.893 2.738a.36.36 0 0 1 .083.345l-.333 1.36c-.053.22-.174.267-.402.161-1.499-.698-2.436-2.889-2.436-4.649 0-3.785 2.75-7.262 7.929-7.262 4.163 0 7.398 2.967 7.398 6.931 0 4.136-2.607 7.464-6.227 7.464-1.216 0-2.359-.631-2.75-1.378l-.748 2.853c-.271 1.043-1.002 2.35-1.492 3.146C9.57 23.812 10.763 24 12.017 24 18.635 24 24 18.633 24 12.013 24 5.367 18.635 0 12.017 0z" />
    </svg>
  );
}

function LineIcon({ className }: { className?: string }) {
  return (
    <svg className={className} viewBox="0 0 24 24" fill="#00C300">
      <path d="M19.365 9.864c.058 0 .104.048.104.108v.646c0 .058-.048.104-.104.104h-1.666v.514h1.666c.058 0 .104.046.104.104v.646c0 .058-.048.104-.104.104h-2.418a.104.104 0 0 1-.104-.104V9.078c0-.058.048-.104.104-.104h2.418c.058 0 .104.048.104.108v.646c0 .058-.048.104-.104.104h-1.666v.514h1.666zm-3.95 2.016a.108.108 0 0 1-.07.026.104.104 0 0 1-.104-.108V9.078c0-.058.048-.104.104-.104h.646c.058 0 .104.048.104.108v2.174l1.692-2.232a.104.104 0 0 1 .081-.05h.646c.058 0 .104.048.104.108v2.766c0 .058-.048.104-.104.104h-.646a.104.104 0 0 1-.104-.108V9.69l-1.702 2.234a.104.104 0 0 1-.081.05h-.566v-.094zM24 10.363C24 4.634 18.614.122 12 .122S0 4.634 0 10.363c0 5.066 4.494 9.308 10.562 10.108.41.088.97.27 1.112.618.128.316.084.81.042 1.128l-.18 1.08c-.054.33-.252 1.29 1.132.702 1.384-.586 7.468-4.398 10.188-7.528C24.546 13.578 24 12.048 24 10.363z" />
    </svg>
  );
}

function StravaIcon({ className }: { className?: string }) {
  return (
    <svg className={className} viewBox="0 0 24 24" fill="#FC4C02">
      <path d="M15.387 17.944l-2.089-4.116h-3.065L15.387 24l5.15-10.172h-3.066m-7.008-5.599l2.836 5.598h4.172L10.463 0l-7 13.828h4.169" />
    </svg>
  );
}

function YahooIcon({ className }: { className?: string }) {
  return (
    <svg className={className} viewBox="0 0 24 24" fill="#6001D2">
      <path d="M0 7.1h4.6L7.4 13l2.8-5.9H15L9.2 18.4V24H5.5v-5.6zm17.5-.3h4.4L24 12.1 22 7.1h-4.6L12 18.5H8.3l5.2-11.5h4zm-3.4 10.3c1.4 0 2.5 1.1 2.5 2.5s-1.1 2.5-2.5 2.5-2.5-1.1-2.5-2.5 1.1-2.5 2.5-2.5z" />
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
  facebook: FacebookIcon,
  linkedin: LinkedInIcon,
  discord: DiscordIcon,
  slack: SlackIcon,
  gitlab: GitLabIcon,
  spotify: SpotifyIcon,
  twitch: TwitchIcon,
  amazon: AmazonIcon,
  dropbox: DropboxIcon,
  bitbucket: BitbucketIcon,
  zoom: ZoomIcon,
  patreon: PatreonIcon,
  instagram: InstagramIcon,
  pinterest: PinterestIcon,
  line: LineIcon,
  strava: StravaIcon,
  yahoo: YahooIcon,
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
