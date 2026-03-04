"use client";

import { motion } from "framer-motion";
import { cn } from "@/lib/cn";

// ─── Flow Node ───────────────────────────────────────────────
interface FlowNodeProps {
  label: string;
  sublabel?: string;
  color?:
    | "teal"
    | "amber"
    | "green"
    | "red"
    | "blue"
    | "gray"
    | "purple"
    | "violet";
  size?: "sm" | "md" | "lg";
  icon?: React.ReactNode;
  pulse?: boolean;
  className?: string;
  delay?: number;
}

const colorMap = {
  teal: "border-teal-500/30 bg-teal-500/10 text-teal-700 dark:text-teal-300",
  amber:
    "border-amber-500/30 bg-amber-500/10 text-amber-700 dark:text-amber-300",
  green:
    "border-green-500/30 bg-green-500/10 text-green-700 dark:text-green-300",
  red: "border-red-500/30 bg-red-500/10 text-red-700 dark:text-red-300",
  blue: "border-blue-500/30 bg-blue-500/10 text-blue-700 dark:text-blue-300",
  gray: "border-fd-border bg-fd-muted/50 text-fd-muted-foreground",
  purple:
    "border-purple-500/30 bg-purple-500/10 text-purple-700 dark:text-purple-300",
  violet:
    "border-violet-500/30 bg-violet-500/10 text-violet-700 dark:text-violet-300",
};

const pulseColorMap = {
  teal: "shadow-teal-500/20",
  amber: "shadow-amber-500/20",
  green: "shadow-green-500/20",
  red: "shadow-red-500/20",
  blue: "shadow-blue-500/20",
  gray: "shadow-fd-border/20",
  purple: "shadow-purple-500/20",
  violet: "shadow-violet-500/20",
};

const sizeMap = {
  sm: "px-2.5 py-1.5 text-[10px]",
  md: "px-3 py-2 text-xs",
  lg: "px-4 py-2.5 text-sm",
};

export function FlowNode({
  label,
  sublabel,
  color = "gray",
  size = "md",
  icon,
  pulse = false,
  className,
  delay = 0,
}: FlowNodeProps) {
  return (
    <motion.div
      initial={{ opacity: 0, scale: 0.8 }}
      animate={{ opacity: 1, scale: 1 }}
      transition={{ duration: 0.4, delay }}
      className={cn(
        "relative rounded-lg border font-mono font-medium flex items-center gap-1.5",
        colorMap[color],
        sizeMap[size],
        pulse && `shadow-lg ${pulseColorMap[color]}`,
        className,
      )}
    >
      {pulse && (
        <motion.div
          className={cn("absolute inset-0 rounded-lg border", colorMap[color])}
          animate={{ opacity: [0.5, 0], scale: [1, 1.15] }}
          transition={{ duration: 2, repeat: Infinity, ease: "easeOut" }}
        />
      )}
      {icon}
      <span>{label}</span>
      {sublabel && <span className="opacity-60 text-[0.85em]">{sublabel}</span>}
    </motion.div>
  );
}

// ─── Flow Connection Line ────────────────────────────────────
interface FlowLineProps {
  direction?: "horizontal" | "vertical";
  length?: number;
  color?: "teal" | "amber" | "green" | "red" | "blue" | "gray" | "violet" | "purple";
  animated?: boolean;
  className?: string;
  delay?: number;
}

const lineColorMap = {
  teal: "bg-teal-500/40",
  amber: "bg-amber-500/40",
  green: "bg-green-500/40",
  red: "bg-red-500/40",
  blue: "bg-blue-500/40",
  gray: "bg-fd-border",
  violet: "bg-violet-500/40",
  purple: "bg-purple-500/40",
};

const particleColorMap = {
  teal: "bg-teal-400",
  amber: "bg-amber-400",
  green: "bg-green-400",
  red: "bg-red-400",
  blue: "bg-blue-400",
  gray: "bg-fd-muted-foreground",
  violet: "bg-violet-400",
  purple: "bg-purple-400",
};

export function FlowLine({
  direction = "horizontal",
  length = 40,
  color = "violet",
  animated = true,
  className,
  delay = 0,
}: FlowLineProps) {
  const isH = direction === "horizontal";

  return (
    <motion.div
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      transition={{ duration: 0.3, delay }}
      className={cn(
        "relative flex items-center justify-center shrink-0",
        isH ? "flex-row" : "flex-col",
        className,
      )}
      style={isH ? { width: length } : { height: length }}
    >
      {/* Line track */}
      <div
        className={cn(
          "absolute rounded-full",
          lineColorMap[color],
          isH ? "h-[1.5px] w-full" : "w-[1.5px] h-full",
        )}
      />

      {/* Animated particle */}
      {animated && (
        <motion.div
          className={cn(
            "absolute rounded-full size-1.5",
            particleColorMap[color],
          )}
          animate={
            isH
              ? { x: [-length / 2, length / 2], opacity: [0, 1, 1, 0] }
              : { y: [-length / 2, length / 2], opacity: [0, 1, 1, 0] }
          }
          transition={{
            duration: 1.5,
            repeat: Infinity,
            ease: "linear",
            delay: delay * 0.3,
          }}
        />
      )}

      {/* Arrowhead */}
      <div className={cn("absolute", isH ? "right-0" : "bottom-0")}>
        <div
          className={cn(
            "border-solid",
            isH
              ? "border-l-[5px] border-y-[3px] border-y-transparent"
              : "border-t-[5px] border-x-[3px] border-x-transparent",
            color === "teal" &&
              (isH ? "border-l-teal-500/50" : "border-t-teal-500/50"),
            color === "amber" &&
              (isH ? "border-l-amber-500/50" : "border-t-amber-500/50"),
            color === "green" &&
              (isH ? "border-l-green-500/50" : "border-t-green-500/50"),
            color === "red" &&
              (isH ? "border-l-red-500/50" : "border-t-red-500/50"),
            color === "blue" &&
              (isH ? "border-l-blue-500/50" : "border-t-blue-500/50"),
            color === "gray" &&
              (isH ? "border-l-fd-border" : "border-t-fd-border"),
            color === "violet" &&
              (isH ? "border-l-violet-500/50" : "border-t-violet-500/50"),
            color === "purple" &&
              (isH ? "border-l-purple-500/50" : "border-t-purple-500/50"),
          )}
        />
      </div>
    </motion.div>
  );
}

// ─── Flow Particle Stream ────────────────────────────────────
// Multiple particles traveling along a line for richer visual
interface FlowParticleStreamProps {
  direction?: "horizontal" | "vertical";
  length?: number;
  color?: "teal" | "amber" | "green" | "red" | "blue" | "violet";
  count?: number;
  className?: string;
}

export function FlowParticleStream({
  direction = "horizontal",
  length = 60,
  color = "violet",
  count = 3,
  className,
}: FlowParticleStreamProps) {
  const isH = direction === "horizontal";

  return (
    <div
      className={cn(
        "relative flex items-center justify-center shrink-0",
        isH ? "flex-row" : "flex-col",
        className,
      )}
      style={isH ? { width: length } : { height: length }}
    >
      <div
        className={cn(
          "absolute rounded-full",
          lineColorMap[color],
          isH ? "h-[1px] w-full" : "w-[1px] h-full",
        )}
      />
      {Array.from({ length: count }).map((_, i) => (
        <motion.div
          // biome-ignore lint/suspicious/noArrayIndexKey: static particle count
          key={i}
          className={cn(
            "absolute rounded-full size-1",
            particleColorMap[color],
          )}
          animate={
            isH
              ? { x: [-length / 2, length / 2], opacity: [0, 1, 1, 0] }
              : { y: [-length / 2, length / 2], opacity: [0, 1, 1, 0] }
          }
          transition={{
            duration: 2,
            repeat: Infinity,
            ease: "linear",
            delay: (i * 2) / count,
          }}
        />
      ))}
    </div>
  );
}

// ─── Status Badge ────────────────────────────────────────────
interface StatusBadgeProps {
  status: "delivered" | "retry" | "dlq" | "disabled" | "pending";
  label?: string;
  className?: string;
}

const statusConfig = {
  delivered: {
    color:
      "text-green-600 dark:text-green-400 bg-green-500/10 border-green-500/20",
    defaultLabel: "200 Delivered",
    icon: "check",
  },
  retry: {
    color:
      "text-violet-600 dark:text-violet-400 bg-violet-500/10 border-violet-500/20",
    defaultLabel: "503 Retry",
    icon: "retry",
  },
  dlq: {
    color: "text-red-600 dark:text-red-400 bg-red-500/10 border-red-500/20",
    defaultLabel: "422 DLQ",
    icon: "archive",
  },
  disabled: {
    color: "text-gray-500 bg-gray-500/10 border-gray-500/20",
    defaultLabel: "410 Disabled",
    icon: "x",
  },
  pending: {
    color: "text-fd-muted-foreground bg-fd-muted/50 border-fd-border",
    defaultLabel: "Pending",
    icon: "dots",
  },
};

export function StatusBadge({ status, label, className }: StatusBadgeProps) {
  const config = statusConfig[status];

  return (
    <span
      className={cn(
        "inline-flex items-center gap-1 rounded-md border px-2 py-0.5 text-[10px] font-mono font-medium",
        config.color,
        className,
      )}
    >
      {config.icon === "check" && (
        <svg
          className="size-2.5"
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
      {config.icon === "retry" && (
        <svg
          className="size-2.5"
          viewBox="0 0 12 12"
          fill="none"
          aria-hidden="true"
        >
          <path
            d="M2 6a4 4 0 016.5-3M10 6a4 4 0 01-6.5 3"
            stroke="currentColor"
            strokeWidth="1.5"
            strokeLinecap="round"
          />
        </svg>
      )}
      {config.icon === "archive" && (
        <svg
          className="size-2.5"
          viewBox="0 0 12 12"
          fill="none"
          aria-hidden="true"
        >
          <rect
            x="1"
            y="2"
            width="10"
            height="3"
            rx="0.5"
            stroke="currentColor"
            strokeWidth="1"
          />
          <path
            d="M2 5v4.5a.5.5 0 00.5.5h7a.5.5 0 00.5-.5V5"
            stroke="currentColor"
            strokeWidth="1"
          />
          <path
            d="M5 7h2"
            stroke="currentColor"
            strokeWidth="1"
            strokeLinecap="round"
          />
        </svg>
      )}
      {label || config.defaultLabel}
    </span>
  );
}

// ─── Floating Badge ──────────────────────────────────────────
interface FloatingBadgeProps {
  label: string;
  className?: string;
  delay?: number;
}

export function FloatingBadge({
  label,
  className,
  delay = 0,
}: FloatingBadgeProps) {
  return (
    <motion.div
      initial={{ opacity: 0, y: 8 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.5, delay }}
      className={cn(
        "inline-flex items-center gap-1.5 rounded-full border border-fd-border bg-fd-card/80 backdrop-blur-sm px-3 py-1.5 text-[11px] font-medium text-fd-muted-foreground shadow-sm",
        className,
      )}
    >
      <motion.span
        animate={{ y: [0, -3, 0] }}
        transition={{ duration: 3, repeat: Infinity, delay: delay * 0.5 }}
      >
        {label}
      </motion.span>
    </motion.div>
  );
}
