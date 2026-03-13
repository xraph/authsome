"use client";

import { motion } from "framer-motion";
import { cn } from "@/lib/cn";
import { SectionHeader } from "./section-header";

interface SidebarItem {
  label: string;
  active?: boolean;
  indent?: boolean;
}

interface StatCard {
  label: string;
  value: string;
  change?: string;
  up?: boolean;
}

const sidebarSections: { heading: string; items: SidebarItem[] }[] = [
  {
    heading: "Security",
    items: [
      { label: "SSO" },
      { label: "Anomaly Detection" },
    ],
  },
  {
    heading: "Configuration",
    items: [
      { label: "Applications" },
      { label: "Settings" },
      { label: "Environments" },
      { label: "Signup Forms" },
    ],
  },
  {
    heading: "Authentication",
    items: [
      { label: "Organizations" },
      { label: "Social Login" },
    ],
  },
  {
    heading: "Provisioning",
    items: [
      { label: "SCIM" },
      { label: "SCIM Logs" },
    ],
  },
  {
    heading: "Billing",
    items: [
      { label: "Plans" },
      { label: "Subscriptions" },
    ],
  },
  {
    heading: "Authsome",
    items: [
      { label: "Overview", active: true },
      { label: "Passkeys" },
    ],
  },
];

const stats: StatCard[] = [
  { label: "Total Users", value: "2,847", change: "+12%", up: true },
  { label: "Active Sessions", value: "1,203", change: "+8%", up: true },
  { label: "Devices", value: "4,521", change: "+5%", up: true },
  { label: "Plugins", value: "10" },
];

const recentSignups = [
  {
    name: "Sarah Chen",
    email: "sarah@acme.com",
    method: "Google",
    time: "2 min ago",
  },
  {
    name: "James Wilson",
    email: "james@startup.io",
    method: "Password",
    time: "8 min ago",
  },
  {
    name: "Maria Garcia",
    email: "maria@corp.dev",
    method: "Passkey",
    time: "15 min ago",
  },
  {
    name: "Alex Kumar",
    email: "alex@team.co",
    method: "Magic Link",
    time: "23 min ago",
  },
];

const summaryCards = [
  { label: "API Keys", value: "6 active" },
  { label: "Organizations", value: "3 configured" },
  { label: "Social Providers", value: "3 configured" },
  { label: "SSO Providers", value: "1 configured" },
  { label: "OAuth2 Clients", value: "2 active" },
];

const features = [
  {
    title: "User Management",
    description:
      "View, search, and manage users. Reset passwords, revoke sessions, and manage MFA per user.",
  },
  {
    title: "Real-Time Analytics",
    description:
      "Monitor sign-in activity, failed attempts, device fingerprints, and anomaly scores in real time.",
  },
  {
    title: "SSO & SCIM Setup",
    description:
      "Configure SAML and OIDC identity providers per organization. Auto-provision users via SCIM.",
  },
  {
    title: "Billing & Plans",
    description:
      "Manage subscription plans, invoices, and feature flags for multi-tenant SaaS applications.",
  },
];

export function DashboardSection() {
  return (
    <section className="relative w-full py-20 sm:py-28">
      <div className="absolute inset-0 bg-gradient-to-b from-transparent via-indigo-500/[0.02] to-transparent" />

      <div className="relative container max-w-(--fd-layout-width) mx-auto px-4 sm:px-6">
        <SectionHeader
          badge="Free Dashboard"
          title="Manage everything from one place"
          description="A full admin dashboard ships with Authsome — users, sessions, organizations, SSO, SCIM, billing, and more. Free and open source, powered by Forge."
        />

        {/* Dashboard Mock */}
        <motion.div
          initial={{ opacity: 0, y: 30 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true, margin: "-50px" }}
          transition={{ duration: 0.6 }}
          className="mt-14 rounded-xl border border-fd-border bg-fd-card/60 backdrop-blur-sm overflow-hidden shadow-xl shadow-black/5 dark:shadow-black/20"
        >
          {/* Title bar */}
          <div className="flex items-center gap-2 px-4 py-2.5 border-b border-fd-border bg-fd-muted/30">
            <div className="flex items-center gap-1.5">
              <div className="size-2.5 rounded-full bg-red-400/80" />
              <div className="size-2.5 rounded-full bg-yellow-400/80" />
              <div className="size-2.5 rounded-full bg-green-400/80" />
            </div>
            <div className="flex-1 text-center">
              <div className="inline-flex items-center gap-1.5 rounded-md bg-fd-background/60 border border-fd-border/50 px-3 py-0.5 text-[10px] text-fd-muted-foreground">
                <svg
                  className="size-2.5"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="2"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  aria-hidden="true"
                >
                  <rect x="3" y="11" width="18" height="11" rx="2" ry="2" />
                  <path d="M7 11V7a5 5 0 0110 0v4" />
                </svg>
                localhost:8080/admin
              </div>
            </div>
          </div>

          {/* Dashboard body */}
          <div className="flex min-h-[420px]">
            {/* Sidebar */}
            <div className="hidden md:block w-48 border-r border-fd-border bg-fd-muted/10 py-3 shrink-0">
              {/* Logo */}
              <div className="px-3 mb-3 flex items-center gap-2">
                <div className="size-6 rounded-md bg-indigo-500 flex items-center justify-center">
                  <svg
                    className="size-3.5 text-white"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    strokeWidth="2"
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    aria-hidden="true"
                  >
                    <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" />
                  </svg>
                </div>
                <span className="text-[11px] font-semibold text-fd-foreground">
                  Authsome
                </span>
              </div>

              {/* Nav sections */}
              <div className="space-y-3">
                {sidebarSections.map((section) => (
                  <div key={section.heading}>
                    <div className="px-3 mb-1 text-[9px] font-semibold uppercase tracking-wider text-fd-muted-foreground/60">
                      {section.heading}
                    </div>
                    {section.items.map((item) => (
                      <div
                        key={item.label}
                        className={cn(
                          "px-3 py-1 text-[11px] cursor-default",
                          item.active
                            ? "text-indigo-600 dark:text-indigo-400 bg-indigo-500/10 border-r-2 border-indigo-500"
                            : "text-fd-muted-foreground hover:text-fd-foreground",
                        )}
                      >
                        {item.label}
                      </div>
                    ))}
                  </div>
                ))}
              </div>
            </div>

            {/* Main content */}
            <div className="flex-1 p-4 sm:p-5 overflow-hidden">
              {/* Page header */}
              <div className="flex items-center justify-between mb-4">
                <div>
                  <h3 className="text-sm font-semibold text-fd-foreground">
                    Authentication Overview
                  </h3>
                  <p className="text-[10px] text-fd-muted-foreground mt-0.5">
                    Monitor your authentication system
                  </p>
                </div>
                <div className="hidden sm:flex items-center gap-2">
                  <div className="rounded-md border border-fd-border px-2 py-1 text-[10px] text-fd-muted-foreground">
                    Last 7 days
                  </div>
                  <div className="rounded-md bg-indigo-500 px-2 py-1 text-[10px] text-white font-medium">
                    Export
                  </div>
                </div>
              </div>

              {/* Stat cards */}
              <div className="grid grid-cols-2 lg:grid-cols-4 gap-3 mb-4">
                {stats.map((stat) => (
                  <div
                    key={stat.label}
                    className="rounded-lg border border-fd-border bg-fd-background/50 p-3"
                  >
                    <div className="text-[10px] text-fd-muted-foreground">
                      {stat.label}
                    </div>
                    <div className="text-lg font-bold text-fd-foreground mt-0.5">
                      {stat.value}
                    </div>
                    {stat.change && (
                      <div
                        className={cn(
                          "text-[10px] mt-0.5",
                          stat.up
                            ? "text-green-600 dark:text-green-400"
                            : "text-red-600 dark:text-red-400",
                        )}
                      >
                        {stat.up ? "↑" : "↓"} {stat.change} this week
                      </div>
                    )}
                  </div>
                ))}
              </div>

              {/* Recent signups table */}
              <div className="rounded-lg border border-fd-border bg-fd-background/50 overflow-hidden mb-4">
                <div className="px-3 py-2 border-b border-fd-border">
                  <span className="text-[11px] font-semibold text-fd-foreground">
                    Recent Signups
                  </span>
                </div>
                <div className="divide-y divide-fd-border">
                  {recentSignups.map((user) => (
                    <div
                      key={user.email}
                      className="flex items-center justify-between px-3 py-2"
                    >
                      <div className="flex items-center gap-2.5 min-w-0">
                        <div className="size-6 rounded-full bg-gradient-to-br from-indigo-400 to-purple-500 flex items-center justify-center text-[9px] font-bold text-white shrink-0">
                          {user.name
                            .split(" ")
                            .map((n) => n[0])
                            .join("")}
                        </div>
                        <div className="min-w-0">
                          <div className="text-[11px] font-medium text-fd-foreground truncate">
                            {user.name}
                          </div>
                          <div className="text-[10px] text-fd-muted-foreground truncate">
                            {user.email}
                          </div>
                        </div>
                      </div>
                      <div className="flex items-center gap-3 shrink-0">
                        <span className="hidden sm:inline text-[9px] rounded-full bg-fd-muted px-2 py-0.5 text-fd-muted-foreground">
                          {user.method}
                        </span>
                        <span className="text-[10px] text-fd-muted-foreground">
                          {user.time}
                        </span>
                      </div>
                    </div>
                  ))}
                </div>
              </div>

              {/* Summary row */}
              <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-5 gap-2">
                {summaryCards.map((card) => (
                  <div
                    key={card.label}
                    className="rounded-lg border border-fd-border bg-fd-background/50 p-2.5"
                  >
                    <div className="text-[10px] text-fd-muted-foreground">
                      {card.label}
                    </div>
                    <div className="text-[11px] font-semibold text-fd-foreground mt-0.5">
                      {card.value}
                    </div>
                  </div>
                ))}
              </div>
            </div>
          </div>
        </motion.div>

        {/* Feature highlights */}
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }}
          transition={{ duration: 0.5, delay: 0.2 }}
          className="mt-10 grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4"
        >
          {features.map((feature) => (
            <div
              key={feature.title}
              className="rounded-xl border border-fd-border bg-fd-card/50 backdrop-blur-sm p-5 hover:border-indigo-500/20 hover:bg-fd-card/80 transition-all duration-300"
            >
              <h3 className="text-sm font-semibold text-fd-foreground">
                {feature.title}
              </h3>
              <p className="text-xs text-fd-muted-foreground mt-1.5 leading-relaxed">
                {feature.description}
              </p>
            </div>
          ))}
        </motion.div>
      </div>
    </section>
  );
}
