"use client";

import { motion } from "framer-motion";
import { cn } from "@/lib/cn";
import { SectionHeader } from "./section-header";
import {
  FlowNode,
  FlowLine,
  FlowParticleStream,
  StatusBadge,
} from "./flow-primitives";

const features = [
  {
    title: "14 Auth Strategies",
    description:
      "Password, magic link, social OAuth, SSO, passkeys, MFA, phone, API keys, and more — composable and extensible.",
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
        <path d="M12 2v10M8 8l4 4 4-4" />
        <path d="M3 15v4a2 2 0 002 2h14a2 2 0 002-2v-4" />
      </svg>
    ),
  },
  {
    title: "Session Security",
    description:
      "JWT or opaque tokens, refresh rotation, device binding, max sessions, and automatic expiry management.",
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
      </svg>
    ),
  },
  {
    title: "31 Webhook Events",
    description:
      "Every auth action emits events — user.created, session.started, mfa.challenged, org.invited, and 27 more.",
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
];

function AuthPipelineDiagram() {
  return (
    <motion.div
      initial={{ opacity: 0 }}
      whileInView={{ opacity: 1 }}
      viewport={{ once: true }}
      transition={{ duration: 0.6 }}
      className="relative"
    >
      {/* Background glow */}
      <div className="absolute inset-0 -m-4 bg-gradient-to-br from-indigo-500/5 via-transparent to-blue-500/5 rounded-2xl blur-xl" />

      <div className="relative space-y-5 p-4">
        {/* Pipeline: Request → Plugins → Session */}
        <div className="flex items-center justify-center gap-0">
          <FlowNode
            label="Request"
            color="blue"
            size="sm"
            delay={0.2}
            icon={
              <svg
                className="size-3"
                viewBox="0 0 12 12"
                fill="none"
                aria-hidden="true"
              >
                <path
                  d="M2 6h8M7 3l3 3-3 3"
                  stroke="currentColor"
                  strokeWidth="1.5"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                />
              </svg>
            }
          />
          <FlowLine length={28} color="blue" delay={1} />
          <FlowNode label="Plugins" color="teal" size="sm" delay={0.4} />
          <FlowLine length={28} color="green" delay={2} />
          <FlowNode
            label="Session"
            color="green"
            size="sm"
            pulse
            delay={0.6}
          />
        </div>

        {/* Event stream */}
        <div className="space-y-2.5 pl-2">
          <motion.div
            initial={{ opacity: 0, x: -10 }}
            whileInView={{ opacity: 1, x: 0 }}
            viewport={{ once: true }}
            transition={{ delay: 0.8 }}
            className="flex items-center gap-0"
          >
            <FlowLine length={24} color="green" delay={3} />
            <FlowNode
              label="user.created"
              color="gray"
              size="sm"
              delay={0.9}
            />
            <FlowLine length={20} color="green" delay={4} />
            <StatusBadge status="delivered" label="webhook" />
          </motion.div>

          <motion.div
            initial={{ opacity: 0, x: -10 }}
            whileInView={{ opacity: 1, x: 0 }}
            viewport={{ once: true }}
            transition={{ delay: 1.0 }}
            className="flex items-center gap-0"
          >
            <FlowLine length={24} color="violet" delay={5} />
            <FlowNode
              label="mfa.challenged"
              color="gray"
              size="sm"
              delay={1.1}
            />
            <FlowLine length={20} color="violet" delay={6} />
            <StatusBadge status="retry" label="pending" />
          </motion.div>

          <motion.div
            initial={{ opacity: 0, x: -10 }}
            whileInView={{ opacity: 1, x: 0 }}
            viewport={{ once: true }}
            transition={{ delay: 1.2 }}
            className="flex items-center gap-0"
          >
            <FlowLine length={24} color="green" delay={7} />
            <FlowNode
              label="session.started"
              color="gray"
              size="sm"
              delay={1.3}
            />
            <FlowLine length={20} color="green" delay={8} />
            <StatusBadge status="delivered" label="active" />
          </motion.div>
        </div>
      </div>
    </motion.div>
  );
}

export function AuthFlowSection() {
  return (
    <section className="relative w-full py-20 sm:py-28">
      {/* Subtle background */}
      <div className="absolute inset-0 bg-gradient-to-b from-transparent via-indigo-500/[0.02] to-transparent" />

      <div className="relative container max-w-(--fd-layout-width) mx-auto px-4 sm:px-6">
        <SectionHeader
          badge="Auth Pipeline"
          title="From request to session, fully instrumented"
          description="Every authentication flow passes through your configured plugins, emits events, and creates auditable sessions."
        />

        <div className="mt-14 grid grid-cols-1 lg:grid-cols-2 gap-12 lg:gap-16 items-start">
          {/* Left: Feature list */}
          <motion.div
            initial={{ opacity: 0, x: -20 }}
            whileInView={{ opacity: 1, x: 0 }}
            viewport={{ once: true }}
            transition={{ duration: 0.5 }}
            className="space-y-8"
          >
            {features.map((feature, i) => (
              <motion.div
                key={feature.title}
                initial={{ opacity: 0, y: 16 }}
                whileInView={{ opacity: 1, y: 0 }}
                viewport={{ once: true }}
                transition={{ duration: 0.4, delay: 0.1 * i }}
                className="flex gap-4"
              >
                <div className="flex items-center justify-center size-10 rounded-lg bg-indigo-500/10 text-indigo-600 dark:text-indigo-400 shrink-0 mt-0.5">
                  {feature.icon}
                </div>
                <div>
                  <h3 className="text-sm font-semibold text-fd-foreground">
                    {feature.title}
                  </h3>
                  <p className="text-sm text-fd-muted-foreground mt-1 leading-relaxed">
                    {feature.description}
                  </p>
                </div>
              </motion.div>
            ))}
          </motion.div>

          {/* Right: Pipeline diagram */}
          <motion.div
            initial={{ opacity: 0, x: 20 }}
            whileInView={{ opacity: 1, x: 0 }}
            viewport={{ once: true }}
            transition={{ duration: 0.5, delay: 0.2 }}
          >
            <AuthPipelineDiagram />
          </motion.div>
        </div>
      </div>
    </section>
  );
}
