"use client";

import { motion } from "framer-motion";
import Link from "next/link";
import { cn } from "@/lib/cn";
import { AnimatedTagline } from "./animated-tagline";
import {
  FloatingBadge,
  FlowLine,
  FlowNode,
  StatusBadge,
} from "./flow-primitives";

function GitHubIcon({ className }: { className?: string }) {
  return (
    <svg
      className={className}
      fill="currentColor"
      viewBox="0 0 24 24"
      aria-hidden="true"
    >
      <path d="M12 .297c-6.63 0-12 5.373-12 12 0 5.303 3.438 9.8 8.205 11.385.6.113.82-.258.82-.577 0-.285-.01-1.04-.015-2.04-3.338.724-4.042-1.61-4.042-1.61C4.422 18.07 3.633 17.7 3.633 17.7c-1.087-.744.084-.729.084-.729 1.205.084 1.838 1.236 1.838 1.236 1.07 1.835 2.809 1.305 3.495.998.108-.776.417-1.305.76-1.605-2.665-.3-5.466-1.332-5.466-5.93 0-1.31.465-2.38 1.235-3.286-.135-.303-.54-1.523.105-3.176 0 0 1.005-.322 3.3 1.23.96-.267 1.98-.399 3-.405 1.02.006 2.04.138 3 .405 2.28-1.552 3.285-1.23 3.285-1.23.645 1.653.24 2.873.12 3.176.765.84 1.23 1.91 1.23 3.22 0 4.61-2.805 5.625-5.475 5.92.42.36.81 1.096.81 2.22 0 1.606-.015 2.896-.015 3.286 0 .315.21.69.825.57C20.565 22.092 24 17.592 24 12.297c0-6.627-5.373-12-12-12" />
    </svg>
  );
}

// ─── Mini Auth Flow Diagram ─────────────────────────────────
function MiniAuthFlow() {
  return (
    <motion.div
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      transition={{ duration: 0.6, delay: 0.3 }}
      className="relative w-full max-w-md mx-auto"
    >
      {/* Background glow */}
      <div className="absolute inset-0 -m-8 bg-gradient-to-br from-indigo-500/5 via-transparent to-blue-500/5 rounded-3xl blur-2xl" />

      <div className="relative space-y-6 p-4">
        {/* Row 1: SignUp → Verify → SignIn → Session */}
        <div className="flex items-center justify-center gap-0">
          <FlowNode
            label="SignUp()"
            color="blue"
            size="sm"
            delay={0.4}
            icon={
              <svg
                className="size-3"
                viewBox="0 0 12 12"
                fill="none"
                aria-hidden="true"
              >
                <path
                  d="M6 2v8M2 6h8"
                  stroke="currentColor"
                  strokeWidth="1.5"
                  strokeLinecap="round"
                />
              </svg>
            }
          />
          <FlowLine length={24} color="blue" delay={1} />
          <FlowNode label="Verify" color="teal" size="sm" delay={0.55} />
          <FlowLine length={24} color="blue" delay={2} />
          <FlowNode label="SignIn()" color="blue" size="sm" delay={0.7} />
          <FlowLine length={24} color="green" delay={3} />
          <FlowNode
            label="Session"
            color="green"
            size="sm"
            pulse
            delay={0.85}
          />
        </div>

        {/* Row 2: Auth events */}
        <div className="flex items-start justify-center">
          <div className="space-y-2.5">
            {/* Event 1: user created */}
            <motion.div
              initial={{ opacity: 0, x: -10 }}
              animate={{ opacity: 1, x: 0 }}
              transition={{ delay: 1.0 }}
              className="flex items-center gap-0"
            >
              <FlowLine length={28} color="green" delay={3} />
              <FlowNode label="auth.signup" color="gray" size="sm" delay={1.1} />
              <FlowLine length={24} color="green" delay={4} />
              <StatusBadge status="delivered" label="created" />
            </motion.div>

            {/* Event 2: mfa challenge */}
            <motion.div
              initial={{ opacity: 0, x: -10 }}
              animate={{ opacity: 1, x: 0 }}
              transition={{ delay: 1.2 }}
              className="flex items-center gap-0"
            >
              <FlowLine length={28} color="violet" delay={5} />
              <FlowNode label="mfa.verify" color="gray" size="sm" delay={1.3} />
              <FlowLine length={24} color="violet" delay={6} />
              <StatusBadge status="retry" label="challenged" />
            </motion.div>

            {/* Event 3: session created */}
            <motion.div
              initial={{ opacity: 0, x: -10 }}
              animate={{ opacity: 1, x: 0 }}
              transition={{ delay: 1.4 }}
              className="flex items-center gap-0"
            >
              <FlowLine length={28} color="green" delay={7} />
              <FlowNode
                label="session.new"
                color="gray"
                size="sm"
                delay={1.5}
              />
              <FlowLine length={24} color="green" delay={8} />
              <StatusBadge status="delivered" label="active" />
            </motion.div>
          </div>
        </div>

        {/* Floating capability badges */}
        <div className="flex flex-wrap items-center justify-center gap-2 pt-2">
          <FloatingBadge label="14 Plugins" delay={1.6} />
          <FloatingBadge label="Multi-Tenant" delay={1.8} />
          <FloatingBadge label="Forge-Native" delay={2.0} />
          <FloatingBadge label="RBAC" delay={2.2} />
        </div>
      </div>
    </motion.div>
  );
}

// ─── Hero Section ────────────────────────────────────────────
export function Hero() {
  return (
    <section className="relative w-full overflow-hidden">
      {/* Dotted background */}
      <div className="absolute inset-0 bg-dotted opacity-40 dark:opacity-20" />

      {/* Radial gradient overlays */}
      <div className="absolute inset-0 bg-gradient-to-b from-fd-background via-transparent to-fd-background" />
      <div className="absolute top-0 left-1/2 -translate-x-1/2 w-[800px] h-[600px] bg-gradient-to-b from-indigo-500/8 to-transparent rounded-full blur-3xl" />

      <div className="relative container max-w-(--fd-layout-width) mx-auto px-4 sm:px-6">
        <div className="grid gap-12 lg:grid-cols-2 lg:gap-16 items-center py-20 sm:py-28 md:py-32">
          {/* Left: Text content */}
          <div className="flex flex-col items-start">
            {/* Pill badge */}
            <motion.div
              initial={{ opacity: 0, y: 12 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.4 }}
            >
              <span className="inline-flex items-center rounded-full border border-indigo-500/20 bg-indigo-500/10 px-3.5 py-1 text-xs font-medium text-indigo-600 dark:text-indigo-400 mb-6">
                Production-grade authentication for Go
              </span>
            </motion.div>

            {/* Animated headline */}
            <AnimatedTagline />

            {/* Description */}
            <motion.p
              initial={{ opacity: 0, y: 12 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.5, delay: 0.6 }}
              className="mt-6 text-lg text-fd-muted-foreground leading-relaxed max-w-lg"
            >
              Composable authentication engine with 14 plugins, session
              management, RBAC, SSO, passkeys, and MFA &mdash; multi-tenant,
              store-abstracted, and Forge-native.
            </motion.p>

            {/* Install command */}
            <motion.div
              initial={{ opacity: 0, y: 12 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.5, delay: 0.8 }}
              className="mt-6 flex items-center gap-2 rounded-lg border border-fd-border bg-fd-muted/40 px-4 py-2.5 font-mono text-sm"
            >
              <span className="text-fd-muted-foreground select-none">$</span>
              <code className="text-fd-foreground">
                go get github.com/xraph/authsome
              </code>
            </motion.div>

            {/* CTAs */}
            <motion.div
              initial={{ opacity: 0, y: 12 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.5, delay: 1.0 }}
              className="mt-8 flex items-center gap-3"
            >
              <Link
                href="/docs"
                className={cn(
                  "inline-flex items-center justify-center rounded-lg px-5 py-2.5 text-sm font-medium transition-colors",
                  "bg-indigo-500 text-white hover:bg-indigo-600",
                  "shadow-sm shadow-indigo-500/20",
                )}
              >
                Get Started
              </Link>
              <a
                href="https://github.com/xraph/authsome"
                target="_blank"
                rel="noreferrer"
                className={cn(
                  "inline-flex items-center gap-2 justify-center rounded-lg px-5 py-2.5 text-sm font-medium transition-colors",
                  "border border-fd-border bg-fd-background hover:bg-fd-muted/50 text-fd-foreground",
                )}
              >
                <GitHubIcon className="size-4" />
                GitHub
              </a>
            </motion.div>
          </div>

          {/* Right: Mini auth flow diagram */}
          <div className="relative lg:pl-8">
            <MiniAuthFlow />
          </div>
        </div>
      </div>
    </section>
  );
}
