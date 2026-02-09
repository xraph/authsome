import Link from "next/link";
import {
  ArrowRight,
  Zap,
  Shield,
  Key,
  Fingerprint,
  Smartphone,
  Mail,
  Lock,
  Building2,
  Globe,
  Scan,
  KeyRound,
  BadgeCheck,
  ShieldCheck,
  GitBranch,
  Server,
} from "lucide-react";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";
import { ReactNode } from "react";
import { LineShadowText } from "@/components/ui/line-shadow-text";
import FeatureHighlightSection from "@/components/feature-highlight-section";
import SpecialHighlightSection from "@/components/special-highlight-section";
import FooterSection from "@/components/footer-section";

/**
 * Hero Section Component
 * Modern hero with gradient background, tagline, and code preview
 */
function HeroSection() {
  return (
    <section className="relative container overflow-hidden py-24 sm:py-32">
      <div className="absolute inset-0 bg-[radial-gradient(ellipse_60%_50%_at_50%_-20%,rgba(120,119,198,0.15),transparent)]" />
      <div className="relative mx-auto max-w-7xl px-6 lg:px-8 grid grid-cols-1 lg:grid-cols-2 gap-12 items-center">
        <div className="max-w-2xl text-left">
          <Badge variant="outline" className="mb-4">
            <Zap className="mr-1 h-3 w-3" />
            Enterprise-Grade Authentication for Go
          </Badge>
          <h1 className="text-4xl font-bold tracking-tight text-foreground sm:text-6xl">
            Auth
            <LineShadowText className="italic" shadowColor="var(--color-foreground)">
              some
            </LineShadowText>
          </h1>
          <p className="mt-6 text-lg leading-8 text-muted-foreground">
            A comprehensive, pluggable authentication framework for Go. 
            Multi-tenancy, RBAC, 30+ plugins, and enterprise security -- built 
            on the Forge framework so you can ship auth in minutes, not months.
          </p>
          <div className="mt-10 flex items-center justify-start gap-x-6">
            <Link
              href="/docs/go/getting-started"
              className="rounded-md bg-brand px-3.5 py-2.5 text-sm font-semibold text-brand-foreground shadow-sm hover:bg-brand/90 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-brand"
            >
              Get Started
              <ArrowRight className="ml-2 h-4 w-4 inline" />
            </Link>
            <Link
              href="/docs/go/examples"
              className="text-sm font-semibold leading-6 text-foreground hover:text-brand"
            >
              View Examples <span aria-hidden="true">&rarr;</span>
            </Link>
          </div>
        </div>

        {/* Code preview */}
        <div className="hidden lg:block">
          <div className="rounded-xl border border-border bg-muted/50 backdrop-blur-sm shadow-lg overflow-hidden">
            <div className="flex items-center gap-1.5 px-4 py-3 border-b border-border bg-muted/80">
              <div className="h-2.5 w-2.5 rounded-full bg-red-400/60" />
              <div className="h-2.5 w-2.5 rounded-full bg-yellow-400/60" />
              <div className="h-2.5 w-2.5 rounded-full bg-green-400/60" />
              <span className="ml-2 text-xs text-muted-foreground font-mono">main.go</span>
            </div>
            <pre className="p-5 text-sm leading-relaxed font-mono overflow-x-auto">
              <code>
                <span className="text-muted-foreground">{"// Initialize AuthSome with plugins"}</span>{"\n"}
                <span className="text-blue-500 dark:text-blue-400">auth</span>{" := authsome."}<span className="text-purple-600 dark:text-purple-400">New</span>{"(\n"}
                {"  authsome."}<span className="text-purple-600 dark:text-purple-400">WithPlugins</span>{"(\n"}
                {"    social."}<span className="text-purple-600 dark:text-purple-400">New</span>{"(),\n"}
                {"    passkey."}<span className="text-purple-600 dark:text-purple-400">New</span>{"(),\n"}
                {"    mfa."}<span className="text-purple-600 dark:text-purple-400">New</span>{"(),\n"}
                {"    organization."}<span className="text-purple-600 dark:text-purple-400">New</span>{"(),\n"}
                {"  ),\n"}
                {")\n\n"}
                <span className="text-muted-foreground">{"// Mount on your Forge app"}</span>{"\n"}
                {"app."}<span className="text-purple-600 dark:text-purple-400">Mount</span>{"("}<span className="text-green-600 dark:text-green-400">{'"'}/auth{'"'}</span>{", auth)"}
              </code>
            </pre>
          </div>
        </div>
      </div>
    </section>
  );
}

/**
 * Quick Start Section
 * Shows installation and basic usage
 */
function QuickStartSection() {
  return (
    <section className="relative container overflow-hidden py-24 sm:py-32">
      <div className="absolute inset-0 bg-[radial-gradient(ellipse_80%_50%_at_50%_100%,rgba(120,119,198,0.08),transparent)]" />
      <div className="relative mx-auto max-w-7xl px-6 lg:px-8">
        <div className="mx-auto max-w-2xl text-center">
          <h2 className="text-3xl font-bold tracking-tight text-foreground sm:text-4xl">
            Get started in minutes
          </h2>
          <p className="mt-4 text-lg text-muted-foreground">
            Add enterprise-grade authentication to your Go application with just
            a few lines of code.
          </p>
        </div>
        <div className="mx-auto mt-16 max-w-4xl">
          <div className="grid gap-8 lg:grid-cols-2">
            <Card className="shadow-md rounded-sm bg-background/30 backdrop-blur-sm">
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <span className="flex h-6 w-6 items-center justify-center rounded-full bg-brand text-xs font-bold text-brand-foreground">
                    1
                  </span>
                  Install AuthSome
                </CardTitle>
              </CardHeader>
              <CardContent>
                <div className="rounded-lg bg-muted p-4">
                  <code className="text-sm font-mono">
                    go get github.com/xraph/authsome
                  </code>
                </div>
              </CardContent>
            </Card>
            <Card className="shadow-md rounded-sm bg-background/30 backdrop-blur-sm">
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <span className="flex h-6 w-6 items-center justify-center rounded-full bg-brand text-xs font-bold text-brand-foreground">
                    2
                  </span>
                  Configure and Mount
                </CardTitle>
              </CardHeader>
              <CardContent>
                <div className="rounded-lg bg-muted p-4">
                  <code className="text-sm font-mono whitespace-pre">
                    {"auth := authsome.New(\n  authsome.WithPlugins(...),\n)\napp.Mount(\"/auth\", auth)"}
                  </code>
                </div>
              </CardContent>
            </Card>
          </div>
          <div className="mt-8 text-center">
            <Link
              href="/docs/go/getting-started"
              className="inline-flex items-center rounded-md bg-brand px-4 py-2 text-sm font-semibold text-brand-foreground shadow-sm hover:bg-brand/90"
            >
              View Full Tutorial
              <ArrowRight className="ml-2 h-4 w-4" />
            </Link>
          </div>
        </div>
      </div>
    </section>
  );
}

/**
 * Auth Methods Showcase Section
 * Displays the breadth of supported authentication methods
 */
function AuthMethodsSection() {
  const categories = [
    {
      title: "Password-Based",
      methods: [
        { label: "Email / Password", icon: Mail },
        { label: "Username / Password", icon: Key },
      ],
    },
    {
      title: "Passwordless",
      methods: [
        { label: "Magic Link", icon: Mail },
        { label: "Passkey / WebAuthn", icon: Fingerprint },
        { label: "Email OTP", icon: KeyRound },
        { label: "Phone / SMS", icon: Smartphone },
      ],
    },
    {
      title: "Social OAuth",
      methods: [
        { label: "Google", icon: Globe },
        { label: "GitHub", icon: GitBranch },
        { label: "Apple", icon: Globe },
        { label: "Microsoft", icon: Building2 },
        { label: "Discord", icon: Globe },
        { label: "10+ more", icon: Globe },
      ],
    },
    {
      title: "Multi-Factor",
      methods: [
        { label: "TOTP", icon: Smartphone },
        { label: "SMS Codes", icon: Smartphone },
        { label: "Email Codes", icon: Mail },
        { label: "Backup Codes", icon: Key },
        { label: "WebAuthn MFA", icon: Fingerprint },
      ],
    },
    {
      title: "Enterprise",
      methods: [
        { label: "SAML SSO", icon: Building2 },
        { label: "OIDC Provider", icon: Server },
        { label: "Mutual TLS", icon: Lock },
        { label: "API Keys", icon: Key },
        { label: "JWT / Bearer", icon: BadgeCheck },
        { label: "SCIM 2.0", icon: Scan },
      ],
    },
    {
      title: "Advanced Security",
      methods: [
        { label: "Step-Up Auth", icon: ShieldCheck },
        { label: "Geofencing", icon: Globe },
        { label: "ID Verification", icon: Scan },
        { label: "Device Tracking", icon: Smartphone },
        { label: "Risk-Based MFA", icon: Shield },
      ],
    },
  ];

  return (
    <section className="py-24 sm:py-32">
      <div className="mx-auto max-w-7xl px-6 lg:px-8">
        <div className="mx-auto max-w-2xl text-center">
          <h2 className="text-3xl font-bold tracking-tight text-foreground sm:text-4xl">
            Every authentication method you need
          </h2>
          <p className="mt-4 text-lg text-muted-foreground">
            From simple passwords to enterprise SSO, AuthSome covers the full
            spectrum of authentication -- all via a unified plugin system.
          </p>
        </div>
        <div className="mx-auto mt-16 grid max-w-6xl grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3">
          {categories.map((category) => (
            <div
              key={category.title}
              className="rounded-xl border border-border bg-card/50 p-6 transition-colors hover:border-brand/30"
            >
              <h3 className="text-sm font-semibold uppercase tracking-wider text-brand mb-4">
                {category.title}
              </h3>
              <div className="space-y-3">
                {category.methods.map((method) => (
                  <div
                    key={method.label}
                    className="flex items-center gap-3 text-sm text-muted-foreground"
                  >
                    <method.icon className="h-4 w-4 shrink-0 text-foreground/60" />
                    <span>{method.label}</span>
                  </div>
                ))}
              </div>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}

/**
 * Community Section
 * Links to community resources and contribution
 */
function CommunitySection() {
  return (
    <section className="py-24 sm:py-32 bg-muted/30">
      <div className="mx-auto max-w-7xl px-6 lg:px-8">
        <div className="mx-auto max-w-2xl text-center">
          <h2 className="text-3xl font-bold tracking-tight text-foreground sm:text-4xl">
            Open source, community driven
          </h2>
          <p className="mt-4 text-lg text-muted-foreground">
            AuthSome is built in the open. Contribute, report issues, or join
            the conversation.
          </p>
        </div>
        <div className="mx-auto mt-16 grid max-w-2xl grid-cols-1 gap-8 sm:mt-20 lg:mx-0 lg:max-w-none lg:grid-cols-2">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <GitBranch className="h-5 w-5 text-brand" />
                GitHub Repository
              </CardTitle>
            </CardHeader>
            <CardContent>
              <CardDescription className="text-base mb-4">
                Star the project, report issues, and contribute to the codebase.
              </CardDescription>
              <Link
                href="https://github.com/xraph/authsome"
                className="inline-flex items-center text-sm font-semibold text-brand hover:text-brand/80"
              >
                View on GitHub
                <ArrowRight className="ml-1 h-4 w-4" />
              </Link>
            </CardContent>
          </Card>
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Shield className="h-5 w-5 text-brand" />
                Documentation
              </CardTitle>
            </CardHeader>
            <CardContent>
              <CardDescription className="text-base mb-4">
                Explore guides, API references, and examples to get the most out
                of AuthSome.
              </CardDescription>
              <Link
                href="/docs"
                className="inline-flex items-center text-sm font-semibold text-brand hover:text-brand/80"
              >
                Browse Documentation
                <ArrowRight className="ml-1 h-4 w-4" />
              </Link>
            </CardContent>
          </Card>
        </div>
      </div>
    </section>
  );
}

/**
 * Container wrapper for consistent page-level styling
 */
function ContainerSection({ children }: { children: ReactNode }) {
  return (
    <section className="container mx-auto px-6 lg:px-8">
      {children}
    </section>
  );
}

/**
 * Main Home Page Component
 * Combines all sections into a comprehensive landing page
 */
export default function HomePage() {
  return (
    <main className="min-h-screen">
      <ContainerSection>
        <div className="bg-fd-secondary/50 p-3 empty:hidden"></div>
        <div className="bg-background/10 absolute inset-0 z-[-2] backdrop-blur-[85px] will-change-transform md:backdrop-blur-[170px]"></div>
        <div className="absolute inset-0 z-[-1] size-full opacity-70 mix-blend-overlay dark:md:opacity-100" style={{ background: 'url(/images/noise.webp) lightgray 0% 0% / 83.69069695472717px 83.69069695472717px repeat' }}></div>
        <div className="h-full border-l border-r border-border">
          <HeroSection />
        </div>
      </ContainerSection>
      <Separator />

      <ContainerSection>
        <div className="h-full border-l border-r border-border">
          <FeatureHighlightSection />
          <SpecialHighlightSection />
          <Separator />
          <QuickStartSection />
          <Separator />
          <AuthMethodsSection />
          <Separator />
          <CommunitySection />
          <FooterSection />
        </div>
      </ContainerSection>
    </main>
  );
}
