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
    title: "Organizations",
    description:
      "User-created workspaces with metadata, branding, and slug-based URLs.",
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
    title: "Team Management",
    description:
      "Sub-teams within organizations. Invite, remove, and manage members.",
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
        <path d="M17 21v-2a4 4 0 00-4-4H5a4 4 0 00-4 4v2" />
        <circle cx="9" cy="7" r="4" />
        <path d="M23 21v-2a4 4 0 00-3-3.87M16 3.13a4 4 0 010 7.75" />
      </svg>
    ),
  },
  {
    title: "Role Hierarchy",
    description:
      "Parent roles, custom permissions, and organization-scoped role assignments.",
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
        <path d="M12 2L2 7l10 5 10-5-10-5z" />
        <path d="M2 17l10 5 10-5" />
        <path d="M2 12l10 5 10-5" />
      </svg>
    ),
  },
  {
    title: "Invitation System",
    description:
      "Token-based invitations with TTL expiration, accept/decline tracking.",
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
        <path d="M4 4h16c1.1 0 2 .9 2 2v12c0 1.1-.9 2-2 2H4c-1.1 0-2-.9-2-2V6c0-1.1.9-2 2-2z" />
        <polyline points="22,6 12,13 2,6" />
      </svg>
    ),
  },
];

const orgCode = `// Create organization
org, _ := engine.CreateOrganization(ctx,
  authsome.CreateOrgInput{
    Name: "Acme Corp",
    Slug: "acme",
  })

// Invite member with role
engine.InviteOrgMember(ctx, authsome.InviteInput{
  OrgID: org.ID,
  Email: "jane@acme.com",
  Role:  "admin",
})

// Assign RBAC permissions
engine.AssignRole(ctx, authsome.RoleAssignment{
  UserID: userID,
  Role:   "editor",
  Permissions: []authsome.Permission{
    {Action: "write", Resource: "documents"},
    {Action: "read", Resource: "analytics"},
  },
})`;

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

export function OrgsRbacSection() {
  return (
    <section className="relative w-full py-20 sm:py-28">
      <div className="container max-w-(--fd-layout-width) mx-auto px-4 sm:px-6">
        <SectionHeader
          badge="Organizations"
          title="Teams, roles, and permissions"
          description="Built-in organization management with hierarchical RBAC. Invite members, assign roles, and enforce permissions at every level."
        />

        <div className="mt-14 grid grid-cols-1 lg:grid-cols-2 gap-6">
          {/* Left: Code block */}
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            whileInView={{ opacity: 1, y: 0 }}
            viewport={{ once: true }}
            transition={{ duration: 0.5 }}
          >
            <CodeBlock
              code={orgCode}
              filename="orgs.go"
              language="go"
            />
          </motion.div>

          {/* Right: Feature cards */}
          <motion.div
            variants={containerVariants}
            initial="hidden"
            whileInView="visible"
            viewport={{ once: true, margin: "-50px" }}
            className="grid grid-cols-2 gap-3"
          >
            {features.map((feature) => (
              <motion.div
                key={feature.title}
                variants={itemVariants}
                className="rounded-lg border border-fd-border bg-fd-card/30 p-4"
              >
                <div className="flex items-center justify-center size-9 rounded-lg bg-violet-500/10 text-violet-600 dark:text-violet-400 mb-3">
                  {feature.icon}
                </div>
                <h3 className="text-sm font-semibold text-fd-foreground">
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
