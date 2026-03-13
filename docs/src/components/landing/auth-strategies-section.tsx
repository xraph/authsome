"use client";

import { motion } from "framer-motion";
import { cn } from "@/lib/cn";
import { SectionHeader } from "./section-header";

interface AuthStrategy {
  title: string;
  description: string;
  icon: React.ReactNode;
}

const strategies: AuthStrategy[] = [
  {
    title: "Password",
    description:
      "Traditional email + password with bcrypt hashing, configurable complexity, and secure reset flows.",
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
        <path d="M21 2l-2 2m-7.61 7.61a5.5 5.5 0 11-7.778 7.778 5.5 5.5 0 017.777-7.777zm0 0L15.5 7.5m0 0l3 3L22 7l-3-3m-3.5 3.5L19 4" />
      </svg>
    ),
  },
  {
    title: "Magic Link",
    description:
      "Passwordless email links with configurable TTL. One click to authenticate — no password to remember.",
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
        <path d="M15 4V2M15 16v-2M8 9h2M20 9h2M17.8 11.8l1.4 1.4M12.2 11.8l-1.4 1.4M17.8 6.2l1.4-1.4M12.2 6.2l-1.4-1.4" />
        <path d="M9 16a5 5 0 116-8l-1 1" />
        <path d="M15 14l-6 6" />
      </svg>
    ),
  },
  {
    title: "Social OAuth",
    description:
      "20+ providers — Google, GitHub, Apple, Microsoft, Facebook, Discord, Slack, and more. PKCE support.",
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
        <line x1="2" y1="12" x2="22" y2="12" />
        <path d="M12 2a15.3 15.3 0 014 10 15.3 15.3 0 01-4 10 15.3 15.3 0 01-4-10 15.3 15.3 0 014-10z" />
      </svg>
    ),
  },
  {
    title: "SSO (SAML & OIDC)",
    description:
      "Enterprise single sign-on with per-organization identity provider configuration.",
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
        <path d="M10 6h4" />
        <path d="M10 10h4" />
        <path d="M10 14h4" />
        <path d="M10 18h4" />
      </svg>
    ),
  },
  {
    title: "Passkeys / WebAuthn",
    description:
      "FIDO2 credential registration and biometric authentication. Built-in credential store.",
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
        <path d="M2 12C2 6.5 6.5 2 12 2a10 10 0 018 4" />
        <path d="M5 19.5C5.5 18 6.5 16.5 8 15.5" />
        <path d="M12 12a3 3 0 100-6 3 3 0 000 6z" />
        <path d="M12 12v4" />
        <path d="M12 22a10 10 0 006.3-2.3" />
        <path d="M18 14a4 4 0 10-2.6 7" />
        <path d="M21.8 16a4 4 0 00-5.2-2.4" />
      </svg>
    ),
  },
  {
    title: "MFA (TOTP & SMS)",
    description:
      "Authenticator apps, SMS codes, and recovery codes. Enrollable per-user with challenge verification.",
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
    title: "Phone / SMS",
    description:
      "Phone number verification and SMS-based authentication with configurable providers.",
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
    title: "API Keys",
    description:
      "Machine-to-machine authentication with SHA-256 hashing, prefix-based lookup, and scoped permissions.",
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
        <polyline points="4 17 10 11 4 5" />
        <line x1="12" y1="19" x2="20" y2="19" />
      </svg>
    ),
  },
  {
    title: "OAuth2 Provider",
    description:
      "Act as an OAuth2 authorization server. Authorization Code + PKCE, Client Credentials, token revocation.",
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
        <rect x="2" y="2" width="20" height="8" rx="2" ry="2" />
        <rect x="2" y="14" width="20" height="8" rx="2" ry="2" />
        <line x1="6" y1="6" x2="6.01" y2="6" />
        <line x1="6" y1="18" x2="6.01" y2="18" />
      </svg>
    ),
  },
];

const containerVariants = {
  hidden: {},
  visible: {
    transition: {
      staggerChildren: 0.05,
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

export function AuthStrategiesSection() {
  return (
    <section className="relative w-full py-20 sm:py-28">
      <div className="container max-w-(--fd-layout-width) mx-auto px-4 sm:px-6">
        <SectionHeader
          badge="Auth Strategies"
          title="Every way to authenticate, built in"
          description="From passwords to passkeys, social login to enterprise SSO — Authsome ships strategies for every authentication pattern your app will ever need."
        />

        <motion.div
          variants={containerVariants}
          initial="hidden"
          whileInView="visible"
          viewport={{ once: true, margin: "-50px" }}
          className="mt-14 grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4"
        >
          {strategies.map((strategy) => (
            <motion.div
              key={strategy.title}
              variants={itemVariants}
              className="group relative rounded-xl border border-fd-border bg-fd-card/50 backdrop-blur-sm p-5 hover:border-indigo-500/20 hover:bg-fd-card/80 transition-all duration-300"
            >
              <div className="flex items-center justify-center size-10 rounded-lg bg-indigo-500/10 text-indigo-600 dark:text-indigo-400 mb-3">
                {strategy.icon}
              </div>
              <h3 className="text-sm font-semibold text-fd-foreground">
                {strategy.title}
              </h3>
              <p className="text-xs text-fd-muted-foreground mt-1.5 leading-relaxed">
                {strategy.description}
              </p>
            </motion.div>
          ))}
        </motion.div>
      </div>
    </section>
  );
}
