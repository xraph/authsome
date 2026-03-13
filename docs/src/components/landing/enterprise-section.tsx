"use client";

import { motion } from "framer-motion";
import { cn } from "@/lib/cn";
import { SectionHeader } from "./section-header";
import { CodeBlock } from "./code-block";

interface FeatureCard {
  title: string;
  description: string;
  icon: React.ReactNode;
}

const features: FeatureCard[] = [
  {
    title: "SSO (SAML & OIDC)",
    description:
      "Per-organization identity provider configuration with SAML 2.0 and OpenID Connect support.",
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
        <path d="M6 22V4a2 2 0 012-2h8a2 2 0 012 2v18Z" />
        <path d="M6 12H4a2 2 0 00-2 2v6a2 2 0 002 2h2" />
        <path d="M18 9h2a2 2 0 012 2v9a2 2 0 01-2 2h-2" />
        <path d="M14 10a2 2 0 11-4 0 2 2 0 014 0z" />
        <path d="M12 12v4" />
        <path d="M10 16h4" />
      </svg>
    ),
  },
  {
    title: "SCIM Provisioning",
    description:
      "System for Cross-domain Identity Management. Sync users and groups from your identity provider.",
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
        <path d="M21.5 2v6h-6" />
        <path d="M2.5 22v-6h6" />
        <path d="M21.5 8A10 10 0 003.3 5.3L2.5 6" />
        <path d="M2.5 16a10 10 0 0018.2 2.7l.8-.7" />
      </svg>
    ),
  },
  {
    title: "31+ Webhook Events",
    description:
      "Every auth action emits typed events — user.created, session.started, mfa.challenged, and 28 more.",
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
        <path d="M18 8A6 6 0 006 8c0 7-3 9-3 9h18s-3-2-3-9" />
        <path d="M13.73 21a2 2 0 01-3.46 0" />
      </svg>
    ),
  },
  {
    title: "GDPR & Compliance",
    description:
      "Built-in data export, consent tracking, account deletion, and audit trails for regulatory compliance.",
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
        <path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z" />
        <polyline points="14 2 14 8 20 8" />
        <path d="M9 15l2 2 4-4" />
      </svg>
    ),
  },
  {
    title: "Device Management",
    description:
      "Fingerprinting, trusted device recognition, max devices per user, and device-session binding.",
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
        <rect x="5" y="2" width="14" height="20" rx="2" ry="2" />
        <line x1="12" y1="18" x2="12.01" y2="18" />
      </svg>
    ),
  },
  {
    title: "Admin Dashboard",
    description:
      "Full management UI — users, sessions, organizations, API keys, security events, and SSO configuration.",
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
        <rect x="3" y="3" width="7" height="7" />
        <rect x="14" y="3" width="7" height="7" />
        <rect x="14" y="14" width="7" height="7" />
        <rect x="3" y="14" width="7" height="7" />
      </svg>
    ),
  },
];

const containerVariants = {
  hidden: {},
  visible: {
    transition: {
      staggerChildren: 0.08,
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

export function EnterpriseSection() {
  return (
    <section className="relative w-full py-20 sm:py-28">
      <div className="container max-w-(--fd-layout-width) mx-auto px-4 sm:px-6">
        <SectionHeader
          badge="Enterprise"
          title="Built for production at scale"
          description="SSO, SCIM provisioning, audit trails, GDPR compliance, and a full admin dashboard — everything enterprises need."
        />

        <motion.div
          variants={containerVariants}
          initial="hidden"
          whileInView="visible"
          viewport={{ once: true, margin: "-50px" }}
          className="mt-14 grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4"
        >
          {features.map((feature) => (
            <motion.div
              key={feature.title}
              variants={itemVariants}
              className="rounded-xl border border-fd-border bg-fd-card/50 backdrop-blur-sm p-6 hover:border-indigo-500/20 hover:bg-fd-card/80 transition-all duration-300"
            >
              <div className="flex items-center justify-center size-10 rounded-lg bg-emerald-500/10 text-emerald-600 dark:text-emerald-400">
                {feature.icon}
              </div>
              <h3 className="text-sm font-semibold text-fd-foreground mt-3">
                {feature.title}
              </h3>
              <p className="text-xs text-fd-muted-foreground mt-1.5 leading-relaxed">
                {feature.description}
              </p>
            </motion.div>
          ))}
        </motion.div>
      </div>
    </section>
  );
}
