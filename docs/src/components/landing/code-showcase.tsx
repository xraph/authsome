"use client";

import { motion } from "framer-motion";
import { CodeBlock } from "./code-block";
import { SectionHeader } from "./section-header";

const backendCode = `package main

import (
  "log/slog"
  "net/http"

  "github.com/xraph/authsome"
  "github.com/xraph/authsome/plugin/password"
  "github.com/xraph/authsome/plugin/social"
  "github.com/xraph/authsome/plugin/mfa"
  "github.com/xraph/authsome/store/postgres"
)

func main() {
  engine, _ := authsome.NewEngine(
    authsome.WithStore(postgres.New(pool)),
    authsome.WithPlugins(
      password.New(),
      social.New(social.Google(cfg)),
      mfa.New(mfa.WithTOTP()),
    ),
    authsome.WithLogger(slog.Default()),
  )

  mux := http.NewServeMux()
  engine.RegisterRoutes(mux)
  http.ListenAndServe(":8080", mux)
}`;

const frontendCode = `import { AuthProvider, useAuth } from "@authsome/ui-react"
import {
  SignInForm,
  UserButton,
  OrgSwitcher,
} from "@authsome/ui-components"

function App() {
  return (
    <AuthProvider apiUrl="/api/auth">
      <Layout />
    </AuthProvider>
  )
}

function Layout() {
  const { user, isLoaded } = useAuth()

  if (!isLoaded) return <Loading />

  return user ? (
    <Dashboard>
      <OrgSwitcher />
      <UserButton />
    </Dashboard>
  ) : (
    <SignInForm
      strategies={["password", "google", "passkey"]}
      onSuccess={() => router.push("/dashboard")}
    />
  )
}`;

export function CodeShowcase() {
  return (
    <section className="relative w-full py-20 sm:py-28">
      <div className="container max-w-(--fd-layout-width) mx-auto px-4 sm:px-6">
        <SectionHeader
          badge="Developer Experience"
          title="Backend + Frontend. Fully integrated."
          description="Set up your Go auth server and connect it to your React or Next.js frontend in minutes. Authsome handles both sides."
        />

        <div className="mt-14 grid grid-cols-1 lg:grid-cols-2 gap-6">
          {/* Backend side */}
          <motion.div
            initial={{ opacity: 0, x: -20 }}
            whileInView={{ opacity: 1, x: 0 }}
            viewport={{ once: true }}
            transition={{ duration: 0.5, delay: 0.1 }}
          >
            <div className="mb-3 flex items-center gap-2">
              <div className="size-2 rounded-full bg-indigo-500" />
              <span className="text-xs font-medium text-fd-muted-foreground uppercase tracking-wider">
                Go Backend
              </span>
            </div>
            <CodeBlock code={backendCode} filename="main.go" language="go" />
          </motion.div>

          {/* Frontend side */}
          <motion.div
            initial={{ opacity: 0, x: 20 }}
            whileInView={{ opacity: 1, x: 0 }}
            viewport={{ once: true }}
            transition={{ duration: 0.5, delay: 0.2 }}
          >
            <div className="mb-3 flex items-center gap-2">
              <div className="size-2 rounded-full bg-cyan-500" />
              <span className="text-xs font-medium text-fd-muted-foreground uppercase tracking-wider">
                React Frontend
              </span>
            </div>
            <CodeBlock
              code={frontendCode}
              filename="App.tsx"
              language="tsx"
            />
          </motion.div>
        </div>
      </div>
    </section>
  );
}
