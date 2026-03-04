"use client";

import { motion } from "framer-motion";
import { cn } from "@/lib/cn";
import { CodeBlock } from "./code-block";
import { SectionHeader } from "./section-header";

interface FeatureCard {
  title: string;
  description: string;
  icon: React.ReactNode;
  code: string;
  filename: string;
  colSpan?: number;
}

const features: FeatureCard[] = [
  {
    title: "Plugin Architecture",
    description:
      "14 built-in plugins — password, magic link, social OAuth, SSO, passkeys, MFA, phone, API keys, and more. Each plugin registers strategies, migrations, and hooks automatically.",
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
    code: `engine, _ := authsome.NewEngine(
  authsome.WithStore(postgres.New(pool)),
  authsome.WithPlugins(
    password.New(),
    magiclink.New(mailer),
    social.New(social.Google(cfg)),
    mfa.New(mfa.WithTOTP()),
    passkey.New(rpID, rpOrigins),
  ),
)`,
    filename: "main.go",
  },
  {
    title: "Session & Token Management",
    description:
      "Opaque tokens or JWTs with OIDC claims. Configurable expiry, refresh rotation, max sessions per user, device binding, and JWKS endpoint.",
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
        <circle cx="12" cy="16" r="1" />
      </svg>
    ),
    code: `session, _ := engine.SignIn(ctx,
  authsome.SignInInput{
    Strategy: "password",
    Email:    "user@example.com",
    Password: "secure-pass",
  })
// session.AccessToken  (JWT or opaque)
// session.RefreshToken (rotation enabled)
// session.ExpiresAt`,
    filename: "signin.go",
  },
  {
    title: "Multi-Tenant Isolation",
    description:
      "Every user, session, and org is scoped to an App via context. Cross-tenant queries are structurally impossible. Per-app configuration overrides.",
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
    code: `ctx = authsome.WithAppID(ctx, appID)

// All operations automatically scoped
user, _ := engine.SignUp(ctx, input)
// user.AppID == appID (guaranteed)

// Per-app config overrides
engine.SetAppConfig(ctx, appID, config)`,
    filename: "tenant.go",
  },
  {
    title: "Pluggable Store Backends",
    description:
      "Start with in-memory for testing, swap to PostgreSQL, SQLite, or MongoDB for production. Every subsystem is a Go interface — bring your own backend.",
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
        <ellipse cx="12" cy="5" rx="9" ry="3" />
        <path d="M21 12c0 1.66-4.03 3-9 3s-9-1.34-9-3" />
        <path d="M3 5v14c0 1.66 4.03 3 9 3s9-1.34 9-3V5" />
      </svg>
    ),
    code: `// PostgreSQL (production)
engine, _ := authsome.NewEngine(
  authsome.WithStore(postgres.New(pool)),
)
// SQLite, MongoDB, or Memory
// authsome.WithStore(sqlite.New(db))
// authsome.WithStore(mongodb.New(client))
// authsome.WithStore(memory.New())`,
    filename: "store.go",
  },
  {
    title: "Security & RBAC",
    description:
      "Account lockout, rate limiting, IP control, and role-based access control via Warden integration. 31 webhook events for audit trails.",
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
    code: `engine, _ := authsome.NewEngine(
  authsome.WithLockout(authsome.LockoutConfig{
    MaxAttempts: 5,
    Window:      15 * time.Minute,
  }),
  authsome.WithRateLimit(limiter),
  authsome.WithWebhooks(relay.Bridge()),
)`,
    filename: "security.go",
  },
  {
    title: "React & Next.js UI Components",
    description:
      "40+ pre-built, styled authentication components — sign-in forms, MFA challenges, session management, org switchers, and more. Headless primitives for full control. Server-side session with Next.js middleware.",
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
        <rect x="3" y="3" width="18" height="18" rx="2" />
        <path d="M3 9h18" />
        <path d="M9 21V9" />
      </svg>
    ),
    code: `import { AuthProvider } from "@authsome/ui-react"
import { SignInForm, MFAChallenge } from "@authsome/ui-components"

function App() {
  return (
    <AuthProvider apiUrl="/api/auth">
      <SignInForm
        strategies={["password", "google", "passkey"]}
        onSuccess={() => router.push("/dashboard")}
      />
    </AuthProvider>
  )
}`,
    filename: "App.tsx",
    colSpan: 2,
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

export function FeatureBento() {
  return (
    <section className="relative w-full py-20 sm:py-28">
      <div className="container max-w-(--fd-layout-width) mx-auto px-4 sm:px-6">
        <SectionHeader
          badge="Features"
          title="Everything you need for authentication"
          description="Authsome handles the hard parts — identity, sessions, MFA, social login, RBAC, and multi-tenancy — so you can focus on your application."
        />

        <motion.div
          variants={containerVariants}
          initial="hidden"
          whileInView="visible"
          viewport={{ once: true, margin: "-50px" }}
          className="mt-14 grid grid-cols-1 md:grid-cols-2 gap-4"
        >
          {features.map((feature) => (
            <motion.div
              key={feature.title}
              variants={itemVariants}
              className={cn(
                "group relative rounded-xl border border-fd-border bg-fd-card/50 backdrop-blur-sm p-6 hover:border-indigo-500/20 hover:bg-fd-card/80 transition-all duration-300",
                feature.colSpan === 2 && "md:col-span-2",
              )}
            >
              {/* Header */}
              <div className="flex items-start gap-3 mb-4">
                <div className="flex items-center justify-center size-9 rounded-lg bg-indigo-500/10 text-indigo-600 dark:text-indigo-400 shrink-0">
                  {feature.icon}
                </div>
                <div>
                  <h3 className="text-sm font-semibold text-fd-foreground">
                    {feature.title}
                  </h3>
                  <p className="text-xs text-fd-muted-foreground mt-1 leading-relaxed">
                    {feature.description}
                  </p>
                </div>
              </div>

              {/* Code snippet */}
              <CodeBlock
                code={feature.code}
                filename={feature.filename}
                showLineNumbers={false}
                className="text-xs"
                language={feature.filename.endsWith(".tsx") ? "tsx" : "go"}
              />
            </motion.div>
          ))}
        </motion.div>
      </div>
    </section>
  );
}
