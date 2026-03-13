"use client";

import { motion } from "framer-motion";
import { cn } from "@/lib/cn";
import { SectionHeader } from "./section-header";
import { CodeBlock } from "./code-block";

const securityCode = `engine, _ := authsome.NewEngine(
  authsome.WithPlugins(
    riskengine.New(),
    anomaly.New(),
    geofence.New(geofence.Config{
      AllowedCountries: []string{"US", "CA", "GB"},
    }),
    impossibletravel.New(),
    ipreputation.New(maxmindDB),
    vpndetect.New(),
  ),
  authsome.WithLockout(authsome.LockoutConfig{
    MaxAttempts: 5,
    Window:      15 * time.Minute,
  }),
  authsome.WithRateLimit(limiter),
)`;

const securityFeatures = [
  {
    title: "Risk Engine",
    description:
      "Unified risk scoring across all auth events with configurable thresholds.",
    icon: (
      <svg
        className="size-5"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        strokeWidth="1.5"
        strokeLinecap="round"
        strokeLinejoin="round"
        aria-hidden="true"
      >
        <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" />
        <path d="M9 12l2 2 4-4" />
      </svg>
    ),
  },
  {
    title: "Anomaly Detection",
    description:
      "ML-powered detection of suspicious login patterns and behavioral anomalies.",
    icon: (
      <svg
        className="size-5"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        strokeWidth="1.5"
        strokeLinecap="round"
        strokeLinejoin="round"
        aria-hidden="true"
      >
        <path d="M22 12h-4l-3 9L9 3l-3 9H2" />
      </svg>
    ),
  },
  {
    title: "Geofencing",
    description:
      "Enforce geographic boundaries. Block or challenge logins from unauthorized regions.",
    icon: (
      <svg
        className="size-5"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        strokeWidth="1.5"
        strokeLinecap="round"
        strokeLinejoin="round"
        aria-hidden="true"
      >
        <path d="M21 10c0 7-9 13-9 13s-9-6-9-13a9 9 0 0118 0z" />
        <circle cx="12" cy="10" r="3" />
      </svg>
    ),
  },
  {
    title: "Impossible Travel",
    description:
      "Detect physically impossible login sequences across geographic locations.",
    icon: (
      <svg
        className="size-5"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        strokeWidth="1.5"
        strokeLinecap="round"
        strokeLinejoin="round"
        aria-hidden="true"
      >
        <circle cx="12" cy="12" r="10" />
        <path d="M2 12h20" />
        <path d="M12 2a15.3 15.3 0 014 10 15.3 15.3 0 01-4 10 15.3 15.3 0 01-4-10 15.3 15.3 0 014-10z" />
      </svg>
    ),
  },
  {
    title: "IP Reputation",
    description:
      "Real-time IP risk scoring with MaxMind integration and blocklist support.",
    icon: (
      <svg
        className="size-5"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        strokeWidth="1.5"
        strokeLinecap="round"
        strokeLinejoin="round"
        aria-hidden="true"
      >
        <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z" />
        <circle cx="12" cy="12" r="3" />
      </svg>
    ),
  },
  {
    title: "Account Lockout",
    description:
      "Configurable failure thresholds with automatic lockout and admin unlock.",
    icon: (
      <svg
        className="size-5"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        strokeWidth="1.5"
        strokeLinecap="round"
        strokeLinejoin="round"
        aria-hidden="true"
      >
        <rect x="3" y="11" width="18" height="11" rx="2" ry="2" />
        <path d="M7 11V7a5 5 0 0110 0v4" />
      </svg>
    ),
  },
];

const cardContainerVariants = {
  hidden: {},
  visible: {
    transition: {
      staggerChildren: 0.08,
    },
  },
};

const cardItemVariants = {
  hidden: { opacity: 0, y: 16 },
  visible: {
    opacity: 1,
    y: 0,
    transition: { duration: 0.4, ease: "easeOut" as const },
  },
};

export function SecuritySection() {
  return (
    <section className="relative w-full py-20 sm:py-28">
      {/* Subtle background */}
      <div className="absolute inset-0 bg-gradient-to-b from-transparent via-red-500/[0.02] to-transparent" />

      <div className="relative container max-w-(--fd-layout-width) mx-auto px-4 sm:px-6">
        <SectionHeader
          badge="Security"
          title="Defense in depth, built in"
          description="Enterprise-grade security with intelligent risk assessment. Every auth action is monitored, scored, and auditable."
        />

        <div className="mt-14 grid grid-cols-1 lg:grid-cols-2 gap-6 items-start">
          {/* Left: Code example */}
          <motion.div
            initial={{ opacity: 0, x: -20 }}
            whileInView={{ opacity: 1, x: 0 }}
            viewport={{ once: true }}
            transition={{ duration: 0.5, delay: 0.1 }}
          >
            <CodeBlock
              code={securityCode}
              filename="security.go"
              language="go"
            />
          </motion.div>

          {/* Right: Feature cards grid */}
          <motion.div
            variants={cardContainerVariants}
            initial="hidden"
            whileInView="visible"
            viewport={{ once: true, margin: "-50px" }}
            className="grid grid-cols-2 gap-3"
          >
            {securityFeatures.map((feature) => (
              <motion.div
                key={feature.title}
                variants={cardItemVariants}
                className={cn(
                  "rounded-lg border border-fd-border bg-fd-card/30 p-4",
                  "hover:border-red-500/20 hover:bg-fd-card/60 transition-all duration-300",
                )}
              >
                <div className="flex items-center justify-center size-8 rounded-md bg-red-500/10 text-red-600 dark:text-red-400 mb-3">
                  {feature.icon}
                </div>
                <h3 className="text-sm font-medium text-fd-foreground">
                  {feature.title}
                </h3>
                <p className="text-xs text-fd-muted-foreground mt-1.5 leading-relaxed">
                  {feature.description}
                </p>
              </motion.div>
            ))}
          </motion.div>
        </div>
      </div>
    </section>
  );
}
