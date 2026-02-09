import Link from "next/link";
import {
  ArrowRight,
  Zap,
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
import { SvgGradientBackground } from "@/components/misc-components";
// import { useTheme } from "next-themes";

/**
 * Hero Section Component
 * Modern hero with gradient background and call-to-action
 */
function HeroSection() {
  // const theme = useTheme()
  // const shadowColor = theme.resolvedTheme === "dark" ? "white" : "black"
  return (
    <section className="relative container overflow-hidden  py-24 sm:py-32">
      <SvgGradientBackground />
      <div className="absolute inset-0 bg-grid-white/[0.02] bg-[size:60px_60px]" />
      <div className="relative mx-auto max-w-7xl px-6 lg:px-8 grid grid-cols-1 md:grid-cols-2 gap-8">
        <div className="mx-auto max-w-2xl text-left">
          <Badge variant="outline" className="mb-4">
            <Zap className="mr-1 h-3 w-3" />
            Enterprise-Grade Authentication
          </Badge>
          <h1 className="text-4xl font-bold tracking-tight text-foreground sm:text-6xl">
            AuthSome
            <LineShadowText className="italic" shadowColor='black'>
              UI
            </LineShadowText>
          </h1>
          <p className="mt-6 text-lg leading-8 text-muted-foreground">
            Universal authentication UI components for React and Next.js.
            Adapter-based architecture, 13+ hooks, headless components, and
            full server-side support.
          </p>
          <div className="mt-10 flex items-center justify-start gap-x-6">
            <Link
              href="/docs/ui"
              className="rounded-md bg-brand px-3.5 py-2.5 text-sm font-semibold text-brand-foreground shadow-sm hover:bg-brand/90 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-brand"
            >
              Get Started
              <ArrowRight className="ml-2 h-4 w-4 inline" />
            </Link>
            <Link
              href="/docs/ui/quick-start"
              className="text-sm font-semibold leading-6 text-foreground hover:text-brand"
            >
              Quick Start <span aria-hidden="true">→</span>
            </Link>
          </div>
        </div>

        <div className="mx-auto max-w-2xl text-left">
          <div className="rounded-lg bg-muted p-4">
            <code className="text-sm whitespace-pre text-foreground">
              {`npm install @authsome/ui-core \\
  @authsome/ui-react \\
  @authsome/adapter-authsome`}
            </code>
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
    <section className="relative container overflow-hidden  py-24 sm:py-32">
      <SvgGradientBackground />
      <div className="mx-auto max-w-7xl px-6 lg:px-8">
        <div className="mx-auto max-w-2xl text-center">
          <h2 className="text-3xl font-bold tracking-tight text-foreground sm:text-4xl">
            Get started in minutes
          </h2>
          <p className="mt-4 text-lg text-muted-foreground">
            Add authentication to your React or Next.js application with just
            a few lines of code.
          </p>
        </div>
        <div className="mx-auto mt-16 max-w-4xl">
          <div className="grid gap-8 lg:grid-cols-2">
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <span className="flex h-6 w-6 items-center justify-center rounded-full bg-brand text-xs font-bold text-brand-foreground">
                    1
                  </span>
                  Install Packages
                </CardTitle>
              </CardHeader>
              <CardContent>
                <div className="rounded-lg bg-muted p-4">
                  <code className="text-sm">
                    npm install @authsome/ui-core @authsome/ui-react
                  </code>
                </div>
              </CardContent>
            </Card>
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <span className="flex h-6 w-6 items-center justify-center rounded-full bg-brand text-xs font-bold text-brand-foreground">
                    2
                  </span>
                  Wrap with Provider
                </CardTitle>
              </CardHeader>
              <CardContent>
                <div className="rounded-lg bg-muted p-4">
                  <code className="text-sm whitespace-pre">
                    {`<AuthProvider client={client}>`}
                    <br />
                    {"  "}{`<App />`}
                    <br />
                    {`</AuthProvider>`}
                  </code>
                </div>
              </CardContent>
            </Card>
          </div>
          <div className="mt-8 text-center">
            <Link
              href="/docs/ui/quick-start"
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
 * Navigation Cards Section
 * Main documentation sections with visual cards
 */
function NavigationSection() {
  const sections = [
    {
      title: "Quick Start",
      description:
        "Install packages, set up the provider, and build your first auth flow.",
      href: "/docs/ui/quick-start",
      icon: "🚀",
    },
    {
      title: "Packages",
      description:
        "Core, React, Headless, shadcn CLI, and Next.js packages explained.",
      href: "/docs/ui/packages/core",
      icon: "📦",
    },
    {
      title: "Adapters",
      description:
        "Connect to AuthSome, Clerk, Supabase, or any custom backend.",
      href: "/docs/ui/adapters",
      icon: "🔌",
    },
    {
      title: "Hooks Reference",
      description: "13+ React hooks for auth, OAuth, MFA, passkeys, and more.",
      href: "/docs/ui/api/hooks",
      icon: "🪝",
    },
    {
      title: "Guides",
      description:
        "Next.js integration, auth flows, headless components, and theming.",
      href: "/docs/ui/guides/nextjs-app-router",
      icon: "📖",
    },
    {
      title: "API Reference",
      description: "Complete reference for hooks, components, types, and server utilities.",
      href: "/docs/ui/api/types",
      icon: "📚",
    },
  ];

  return (
    <section className="py-24 sm:py-32">
      <div className="mx-auto max-w-7xl px-6 lg:px-8">
        <div className="mx-auto max-w-2xl text-center">
          <h2 className="text-3xl font-bold tracking-tight text-foreground sm:text-4xl">
            Explore the documentation
          </h2>
          <p className="mt-4 text-lg text-muted-foreground">
            Everything you need to build authentication UIs for React and
            Next.js.
          </p>
        </div>
        <div className="mx-auto mt-16 grid max-w-2xl grid-cols-1 gap-6 sm:mt-20 lg:mx-0 lg:max-w-none lg:grid-cols-3">
          {sections.map((section, index) => (
            <Link key={index} href={section.href} className="group">
              <Card className="h-full transition-all duration-200 hover:shadow-lg hover:border-brand/50 group-hover:scale-[1.02]">
                <CardHeader>
                  <div className="flex items-center gap-3">
                    <span className="text-2xl">{section.icon}</span>
                    <CardTitle className="text-xl group-hover:text-brand transition-colors">
                      {section.title}
                    </CardTitle>
                  </div>
                </CardHeader>
                <CardContent>
                  <CardDescription className="text-base">
                    {section.description}
                  </CardDescription>
                </CardContent>
              </Card>
            </Link>
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
            Join the community
          </h2>
          <p className="mt-4 text-lg text-muted-foreground">
            AuthSome is open source and built by developers, for developers.
          </p>
        </div>
        <div className="mx-auto mt-16 grid max-w-2xl grid-cols-1 gap-8 sm:mt-20 lg:mx-0 lg:max-w-none lg:grid-cols-2">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <span className="text-xl">🐙</span>
                GitHub Repository
              </CardTitle>
            </CardHeader>
            <CardContent>
              <CardDescription className="text-base mb-4">
                Star the project, report issues, and contribute to the codebase.
              </CardDescription>
              <Link
                href="https://github.com/xraph/authsome-ui"
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
                <span className="text-xl">💬</span>
                Community Support
              </CardTitle>
            </CardHeader>
            <CardContent>
              <CardDescription className="text-base mb-4">
                Get help, share ideas, and connect with other developers.
              </CardDescription>
              <Link
                href="/docs/ui/guides/nextjs-app-router"
                className="inline-flex items-center text-sm font-semibold text-brand hover:text-brand/80"
              >
                Join Discussions
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
 * Community Section
 * Links to community resources and contribution
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
export default function UIPage() {
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
          <NavigationSection />
          <Separator />
          <CommunitySection />
          <FooterSection />
        </div>
      </ContainerSection>
    </main>
  );
}
