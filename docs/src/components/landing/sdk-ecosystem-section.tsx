"use client";

import { useState } from "react";
import { motion } from "framer-motion";
import { cn } from "@/lib/cn";
import { SectionHeader } from "./section-header";
import { CodeBlock } from "./code-block";

type SdkTab = "go" | "typescript" | "flutter";

interface TabConfig {
  key: SdkTab;
  label: string;
  code: string;
  filename: string;
  language: "go" | "tsx";
}

const tabs: TabConfig[] = [
  {
    key: "go",
    label: "Go",
    code: `engine, _ := authsome.NewEngine(
  authsome.WithStore(postgres.New(pool)),
  authsome.WithPlugins(
    password.New(),
    social.New(social.Google(cfg)),
  ),
)

// Sign in
session, _ := engine.SignIn(ctx, authsome.SignInInput{
  Strategy: "password",
  Email:    "user@example.com",
  Password: "secret",
})

// Verify session
user, _ := engine.VerifySession(ctx, session.AccessToken)`,
    filename: "main.go",
    language: "go",
  },
  {
    key: "typescript",
    label: "TypeScript",
    code: `import { AuthClient } from "@authsome/client"

const auth = new AuthClient({ baseUrl: "/api/auth" })

// Sign in
const session = await auth.signIn({
  strategy: "password",
  email: "user@example.com",
  password: "secret",
})

// Get current user
const user = await auth.getUser()

// Sign out
await auth.signOut()`,
    filename: "app.ts",
    language: "tsx",
  },
  {
    key: "flutter",
    label: "Flutter",
    code: `import 'package:authsome_flutter/authsome_flutter.dart';

final auth = AuthClient(baseUrl: 'https://api.example.com');

// Sign in
final session = await auth.signIn(
  strategy: 'password',
  email: 'user@example.com',
  password: 'secret',
);

// Get current user
final user = await auth.getUser();

// Listen to auth state
auth.onAuthStateChange.listen((state) {
  print('Auth state: \${state}');
});`,
    filename: "main.dart",
    language: "go",
  },
];

interface FeatureCard {
  title: string;
  description: string;
}

const featureCards: FeatureCard[] = [
  {
    title: "Type-Safe",
    description:
      "Full TypeScript types and Go interfaces. Catch auth errors at compile time.",
  },
  {
    title: "Isomorphic",
    description:
      "Works in Node.js, browsers, React Native, and Flutter. Same API everywhere.",
  },
  {
    title: "Documented",
    description:
      "Comprehensive docs, code examples, and Storybook playground for UI components.",
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

export function SdkEcosystemSection() {
  const [activeTab, setActiveTab] = useState<SdkTab>("go");
  const currentTab = tabs.find((t) => t.key === activeTab) ?? tabs[0];

  return (
    <section className="relative w-full py-20 sm:py-28">
      <div className="container max-w-(--fd-layout-width) mx-auto px-4 sm:px-6">
        <SectionHeader
          badge="SDKs"
          title="Every platform, one API"
          description="Official SDKs for Go, TypeScript, and Flutter. Type-safe clients with full feature coverage across server, web, and mobile."
        />

        <motion.div
          initial={{ opacity: 0, y: 20 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }}
          transition={{ duration: 0.5 }}
          className="mt-14"
        >
          {/* Tab bar */}
          <div className="flex gap-1 p-1 rounded-lg bg-fd-muted/30 border border-fd-border w-fit mb-6">
            {tabs.map((tab) => (
              <button
                key={tab.key}
                type="button"
                onClick={() => setActiveTab(tab.key)}
                className={cn(
                  "px-4 py-2 text-sm font-medium rounded-md transition-colors",
                  activeTab === tab.key
                    ? "bg-fd-background text-fd-foreground shadow-sm"
                    : "text-fd-muted-foreground hover:text-fd-foreground",
                )}
              >
                {tab.label}
              </button>
            ))}
          </div>

          {/* Code block */}
          <CodeBlock
            code={currentTab.code}
            filename={currentTab.filename}
            language={currentTab.language}
          />
        </motion.div>

        {/* Feature cards */}
        <motion.div
          variants={containerVariants}
          initial="hidden"
          whileInView="visible"
          viewport={{ once: true, margin: "-50px" }}
          className="grid grid-cols-1 md:grid-cols-3 gap-4 mt-6"
        >
          {featureCards.map((card) => (
            <motion.div
              key={card.title}
              variants={itemVariants}
              className="rounded-lg border border-fd-border bg-fd-card/30 p-4 text-center"
            >
              <h3 className="text-sm font-semibold text-fd-foreground">
                {card.title}
              </h3>
              <p className="text-xs text-fd-muted-foreground mt-1.5 leading-relaxed">
                {card.description}
              </p>
            </motion.div>
          ))}
        </motion.div>
      </div>
    </section>
  );
}
