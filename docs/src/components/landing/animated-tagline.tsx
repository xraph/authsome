"use client";

import { motion } from "framer-motion";
import { cn } from "@/lib/cn";

const words = ["Authsome", " your", " apps"];

const charVariants = {
  hidden: {
    opacity: 0,
    filter: "blur(8px)",
    y: 8,
  },
  visible: (i: number) => ({
    opacity: 1,
    filter: "blur(0px)",
    y: 0,
    transition: {
      delay: i * 0.03,
      duration: 0.4,
      ease: "easeOut" as const,
    },
  }),
};

export function AnimatedTagline({ className }: { className?: string }) {
  let charIndex = 0;

  return (
    <h1
      className={cn(
        "text-4xl sm:text-5xl md:text-6xl lg:text-7xl font-bold tracking-tight leading-[1.1]",
        className,
      )}
    >
      {words.map((word, wordIdx) => (
        // biome-ignore lint/suspicious/noArrayIndexKey: static word list never reorders
        <span key={wordIdx} className="inline-block">
          {word.split("").map((char) => {
            const currentIndex = charIndex++;
            return (
              <motion.span
                key={currentIndex}
                custom={currentIndex}
                variants={charVariants}
                initial="hidden"
                animate="visible"
                className={cn(
                  "inline-block",
                  wordIdx === 0 &&
                    "bg-gradient-to-r from-indigo-400 via-blue-500 to-cyan-500 bg-clip-text text-transparent",
                  wordIdx !== 0 && "text-fd-foreground",
                  char === " " && "w-[0.25em]",
                )}
              >
                {char === " " ? "\u00A0" : char}
              </motion.span>
            );
          })}
        </span>
      ))}
    </h1>
  );
}
