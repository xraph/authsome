"use client";

import { motion } from "framer-motion";
import { cn } from "@/lib/cn";
import { SectionHeader } from "./section-header";
import Link from "next/link";

interface UICard {
  title: string;
  description: string;
  preview: React.ReactNode;
}

function MockInput({
  label,
  placeholder,
  type = "text",
}: {
  label: string;
  placeholder: string;
  type?: string;
}) {
  return (
    <div className="space-y-1.5">
      <label className="text-[11px] font-medium text-fd-foreground">
        {label}
      </label>
      <div className="rounded-md border border-fd-border bg-fd-background px-3 py-1.5 text-[11px] text-fd-muted-foreground">
        {placeholder}
      </div>
    </div>
  );
}

function MockButton({
  children,
  variant = "primary",
}: {
  children: React.ReactNode;
  variant?: "primary" | "outline" | "social";
}) {
  return (
    <div
      className={cn(
        "rounded-md px-3 py-1.5 text-[11px] font-medium text-center",
        variant === "primary" && "bg-indigo-500 text-white",
        variant === "outline" && "border border-fd-border text-fd-foreground",
        variant === "social" &&
          "border border-fd-border text-fd-foreground flex items-center justify-center gap-1.5",
      )}
    >
      {children}
    </div>
  );
}

const uiCards: UICard[] = [
  {
    title: "Sign In Form",
    description: "Multi-strategy authentication with social providers",
    preview: (
      <div className="space-y-2.5 p-3">
        <div className="text-xs font-semibold text-fd-foreground text-center mb-3">
          Sign in to your account
        </div>
        <MockInput label="Email" placeholder="user@example.com" />
        <MockInput label="Password" placeholder="••••••••" type="password" />
        <MockButton>Sign In</MockButton>
        <div className="flex items-center gap-2 my-1">
          <div className="flex-1 h-px bg-fd-border" />
          <span className="text-[10px] text-fd-muted-foreground">or</span>
          <div className="flex-1 h-px bg-fd-border" />
        </div>
        <MockButton variant="social">
          <svg className="size-3" viewBox="0 0 24 24" aria-hidden="true">
            <path
              d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92a5.06 5.06 0 01-2.2 3.32v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.1z"
              fill="#4285F4"
            />
            <path
              d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"
              fill="#34A853"
            />
            <path
              d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z"
              fill="#FBBC05"
            />
            <path
              d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z"
              fill="#EA4335"
            />
          </svg>
          Continue with Google
        </MockButton>
      </div>
    ),
  },
  {
    title: "MFA Challenge",
    description: "TOTP, SMS OTP, and recovery code verification",
    preview: (
      <div className="space-y-3 p-3">
        <div className="text-center">
          <div className="text-xs font-semibold text-fd-foreground">
            Two-Factor Authentication
          </div>
          <div className="text-[10px] text-fd-muted-foreground mt-1">
            Enter the code from your authenticator app
          </div>
        </div>
        <div className="flex items-center justify-center gap-1.5">
          {Array.from({ length: 6 }).map((_, i) => (
            <div
              // biome-ignore lint/suspicious/noArrayIndexKey: static mock UI
              key={i}
              className={cn(
                "size-7 rounded-md border text-center text-sm font-mono font-bold flex items-center justify-center",
                i < 3
                  ? "border-indigo-500/40 bg-indigo-500/5 text-fd-foreground"
                  : "border-fd-border bg-fd-background text-fd-muted-foreground",
              )}
            >
              {i < 3 ? ["4", "8", "2"][i] : ""}
            </div>
          ))}
        </div>
        <MockButton>Verify</MockButton>
        <div className="text-center">
          <span className="text-[10px] text-indigo-500 cursor-pointer">
            Use recovery code instead
          </span>
        </div>
      </div>
    ),
  },
  {
    title: "Session Manager",
    description: "View and revoke active sessions across devices",
    preview: (
      <div className="space-y-2 p-3">
        <div className="text-xs font-semibold text-fd-foreground mb-2">
          Active Sessions
        </div>
        {[
          { device: "Chrome on macOS", location: "San Francisco", current: true },
          { device: "Safari on iPhone", location: "San Francisco", current: false },
          { device: "Firefox on Windows", location: "New York", current: false },
        ].map((session) => (
          <div
            key={session.device}
            className="flex items-center justify-between rounded-md border border-fd-border p-2"
          >
            <div>
              <div className="text-[11px] font-medium text-fd-foreground flex items-center gap-1.5">
                {session.device}
                {session.current && (
                  <span className="text-[9px] bg-green-500/10 text-green-600 dark:text-green-400 rounded px-1 py-0.5">
                    current
                  </span>
                )}
              </div>
              <div className="text-[10px] text-fd-muted-foreground">
                {session.location}
              </div>
            </div>
            {!session.current && (
              <div className="text-[10px] text-red-500 cursor-pointer">
                Revoke
              </div>
            )}
          </div>
        ))}
      </div>
    ),
  },
  {
    title: "Org Switcher",
    description: "Multi-org support with role-based access",
    preview: (
      <div className="space-y-2 p-3">
        <div className="text-xs font-semibold text-fd-foreground mb-2">
          Organizations
        </div>
        {[
          { name: "Acme Corp", role: "Owner", active: true },
          { name: "Startup Inc", role: "Admin", active: false },
          { name: "Dev Team", role: "Member", active: false },
        ].map((org) => (
          <div
            key={org.name}
            className={cn(
              "flex items-center justify-between rounded-md border p-2 cursor-pointer",
              org.active
                ? "border-indigo-500/30 bg-indigo-500/5"
                : "border-fd-border hover:bg-fd-muted/30",
            )}
          >
            <div className="flex items-center gap-2">
              <div
                className={cn(
                  "size-6 rounded-md flex items-center justify-center text-[10px] font-bold",
                  org.active
                    ? "bg-indigo-500 text-white"
                    : "bg-fd-muted text-fd-muted-foreground",
                )}
              >
                {org.name[0]}
              </div>
              <div>
                <div className="text-[11px] font-medium text-fd-foreground">
                  {org.name}
                </div>
                <div className="text-[10px] text-fd-muted-foreground">
                  {org.role}
                </div>
              </div>
            </div>
            {org.active && (
              <svg
                className="size-3.5 text-indigo-500"
                viewBox="0 0 12 12"
                fill="none"
                aria-hidden="true"
              >
                <path
                  d="M2 6l3 3 5-5"
                  stroke="currentColor"
                  strokeWidth="1.5"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                />
              </svg>
            )}
          </div>
        ))}
      </div>
    ),
  },
];

const containerVariants = {
  hidden: {},
  visible: {
    transition: {
      staggerChildren: 0.1,
    },
  },
};

const itemVariants = {
  hidden: { opacity: 0, y: 20 },
  visible: {
    opacity: 1,
    y: 0,
    transition: { duration: 0.5, ease: "easeOut" as const },
  },
};

export function UIShowcase() {
  return (
    <section className="relative w-full py-20 sm:py-28">
      {/* Subtle background */}
      <div className="absolute inset-0 bg-gradient-to-b from-transparent via-blue-500/[0.02] to-transparent" />

      <div className="relative container max-w-(--fd-layout-width) mx-auto px-4 sm:px-6">
        <SectionHeader
          badge="Authsome UI"
          title="Beautiful, ready-to-use auth components"
          description="40+ pre-built React components for authentication flows. Fully styled, accessible, and customizable. Also available as headless primitives."
        />

        <motion.div
          variants={containerVariants}
          initial="hidden"
          whileInView="visible"
          viewport={{ once: true, margin: "-50px" }}
          className="mt-14 grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4"
        >
          {uiCards.map((card) => (
            <motion.div
              key={card.title}
              variants={itemVariants}
              className="group rounded-xl border border-fd-border bg-fd-card/50 backdrop-blur-sm overflow-hidden hover:border-indigo-500/20 hover:bg-fd-card/80 transition-all duration-300"
            >
              {/* Preview */}
              <div className="border-b border-fd-border bg-fd-muted/20 min-h-[200px] flex items-center justify-center">
                <div className="w-full max-w-[200px]">{card.preview}</div>
              </div>

              {/* Info */}
              <div className="p-4">
                <h3 className="text-sm font-semibold text-fd-foreground">
                  {card.title}
                </h3>
                <p className="text-xs text-fd-muted-foreground mt-1">
                  {card.description}
                </p>
              </div>
            </motion.div>
          ))}
        </motion.div>

        {/* Playground link */}
        <motion.div
          initial={{ opacity: 0, y: 12 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }}
          transition={{ duration: 0.5, delay: 0.4 }}
          className="mt-8 text-center"
        >
          <Link
            href="/docs/ui/playground"
            className="inline-flex items-center gap-2 text-sm text-indigo-600 dark:text-indigo-400 hover:underline underline-offset-4"
          >
            Explore all components in the Storybook playground
            <svg
              className="size-4"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="1.5"
              strokeLinecap="round"
              strokeLinejoin="round"
              aria-hidden="true"
            >
              <path d="M5 12h14M12 5l7 7-7 7" />
            </svg>
          </Link>
        </motion.div>
      </div>
    </section>
  );
}
