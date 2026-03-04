"use client";

import { motion } from "framer-motion";
import { cn } from "@/lib/cn";

interface SectionHeaderProps {
  badge?: string;
  title: string;
  description?: string;
  className?: string;
  align?: "left" | "center";
}

export function SectionHeader({
  badge,
  title,
  description,
  className,
  align = "center",
}: SectionHeaderProps) {
  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      whileInView={{ opacity: 1, y: 0 }}
      viewport={{ once: true }}
      transition={{ duration: 0.5 }}
      className={cn(
        "max-w-2xl",
        align === "center" && "mx-auto text-center",
        className,
      )}
    >
      {badge && (
        <div
          className={cn(
            "inline-flex items-center rounded-full border border-teal-500/20 bg-teal-500/10 px-3 py-1 text-xs font-medium text-teal-600 dark:text-teal-400 mb-4",
          )}
        >
          {badge}
        </div>
      )}
      <h2 className="text-3xl font-bold tracking-tight text-fd-foreground sm:text-4xl">
        {title}
      </h2>
      {description && (
        <p className="mt-4 text-lg text-fd-muted-foreground leading-relaxed">
          {description}
        </p>
      )}
    </motion.div>
  );
}
