import {
  Code,
  Library,
  Terminal,
  Smartphone,
  ArrowRight,
  BookOpen,
  Layers,
  Plug,
  Server,
  Shield,
  Rocket,
} from 'lucide-react';
import Link, { type LinkProps } from 'next/link';

export default function DocsPage() {
  return (
    <main className="container flex flex-col flex-1 py-16 z-2">
      {/* Header */}
      <div className="mx-auto max-w-3xl text-center mb-12">
        <h1 className="mb-4 text-4xl font-bold md:text-5xl">
          Documentation
        </h1>
        <p className="text-fd-muted-foreground text-lg">
          Comprehensive guides, API references, and examples to help you
          integrate AuthSome into your applications.
        </p>
      </div>

      {/* Quick Links */}
      <div className="mx-auto w-full max-w-4xl mb-12">
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-3">
          {[
            {
              label: 'Getting Started',
              href: '/docs/go/getting-started',
              icon: <Rocket className="size-4" />,
            },
            {
              label: 'API Reference',
              href: '/docs/go/api',
              icon: <Server className="size-4" />,
            },
            {
              label: 'Examples',
              href: '/docs/go/examples',
              icon: <BookOpen className="size-4" />,
            },
          ].map((link) => (
            <Link
              key={link.label}
              href={link.href}
              className="group flex items-center gap-3 rounded-lg border border-fd-border bg-fd-card px-4 py-3 text-sm font-medium transition-colors hover:border-fd-primary/40 hover:bg-fd-accent"
            >
              <span className="text-fd-muted-foreground group-hover:text-fd-primary transition-colors">
                {link.icon}
              </span>
              {link.label}
              <ArrowRight className="ml-auto size-3.5 text-fd-muted-foreground opacity-0 transition-all group-hover:opacity-100 group-hover:translate-x-0.5" />
            </Link>
          ))}
        </div>
      </div>

      {/* SDK / Platform Cards */}
      <div className="mx-auto w-full max-w-4xl">
        <h2 className="mb-6 text-sm font-semibold uppercase tracking-wider text-fd-muted-foreground">
          Platforms
        </h2>
        <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
          <ItemLink href="/docs/go">
            <Icon color="blue">
              <Code className="size-full" />
            </Icon>
            <div>
              <h2 className="mb-1 text-base font-semibold">AuthSome for Go</h2>
              <p className="text-sm text-fd-muted-foreground">
                The core Go SDK. Multi-tenancy, 30+ auth plugins, RBAC, audit
                logging, and enterprise security -- all built on the Forge
                framework.
              </p>
            </div>
            <div className="mt-3 flex flex-wrap gap-2">
              <DocBadge>Multi-Tenancy</DocBadge>
              <DocBadge>30+ Plugins</DocBadge>
              <DocBadge>RBAC</DocBadge>
              <DocBadge>Enterprise</DocBadge>
            </div>
          </ItemLink>

          <ItemDisabled>
            <Icon color="orange">
              <Code className="size-full" />
            </Icon>
            <div>
              <h2 className="mb-1 text-base font-semibold flex items-center gap-2">
                AuthSome for Rust
                <span className="rounded-full bg-fd-muted px-2 py-0.5 text-[10px] font-medium uppercase text-fd-muted-foreground">
                  Coming Soon
                </span>
              </h2>
              <p className="text-sm text-fd-muted-foreground">
                A Rust port of AuthSome with the same plugin architecture and
                enterprise features. Under active development.
              </p>
            </div>
          </ItemDisabled>

          <ItemLink href="/docs/ui">
            <Icon color="purple">
              <Smartphone className="size-full" />
            </Icon>
            <div>
              <h2 className="mb-1 text-base font-semibold">AuthSome UI</h2>
              <p className="text-sm text-fd-muted-foreground">
                Pre-built React and Vue components for login forms, MFA flows,
                organization management, and user profile screens.
              </p>
            </div>
            <div className="mt-3 flex flex-wrap gap-2">
              <DocBadge>React</DocBadge>
              <DocBadge>Vue</DocBadge>
              <DocBadge>Customizable</DocBadge>
            </div>
          </ItemLink>

          <ItemLink href="/docs/cli">
            <Icon color="green">
              <Terminal className="size-full" />
            </Icon>
            <div>
              <h2 className="mb-1 text-base font-semibold">AuthSome CLI</h2>
              <p className="text-sm text-fd-muted-foreground">
                Command-line tools for scaffolding projects, managing
                migrations, and administering users and organizations.
              </p>
            </div>
            <div className="mt-3 flex flex-wrap gap-2">
              <DocBadge>Scaffold</DocBadge>
              <DocBadge>Migrate</DocBadge>
              <DocBadge>Admin</DocBadge>
            </div>
          </ItemLink>
        </div>
      </div>

      {/* Documentation Sections */}
      <div className="mx-auto w-full max-w-4xl mt-12">
        <h2 className="mb-6 text-sm font-semibold uppercase tracking-wider text-fd-muted-foreground">
          Learn More
        </h2>
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
          {[
            {
              title: 'Concepts',
              description: 'Users, sessions, organizations, and multi-tenancy.',
              href: '/docs/go/concepts',
              icon: <Layers className="size-full" />,
              color: 'blue' as const,
            },
            {
              title: 'Plugins',
              description: 'Browse 30+ authentication plugins and build your own.',
              href: '/docs/go/plugins',
              icon: <Plug className="size-full" />,
              color: 'purple' as const,
            },
            {
              title: 'Guides',
              description: 'Step-by-step tutorials for common scenarios.',
              href: '/docs/go/guides',
              icon: <BookOpen className="size-full" />,
              color: 'green' as const,
            },
            {
              title: 'Security',
              description: 'Audit logging, device tracking, and compliance.',
              href: '/docs/go/concepts/security',
              icon: <Shield className="size-full" />,
              color: 'orange' as const,
            },
            {
              title: 'API Reference',
              description: 'Complete API docs for all services and handlers.',
              href: '/docs/go/api',
              icon: <Server className="size-full" />,
              color: 'blue' as const,
            },
            {
              title: 'Examples',
              description: 'Real-world sample applications and integrations.',
              href: '/docs/go/examples',
              icon: <Library className="size-full" />,
              color: 'purple' as const,
            },
          ].map((item) => (
            <Link key={item.title} href={item.href} className="group">
              <div className="flex items-start gap-3 rounded-xl border border-fd-border bg-fd-card p-4 transition-all hover:border-fd-primary/30 hover:shadow-sm group-hover:bg-fd-accent/50">
                <Icon color={item.color} size="sm">
                  {item.icon}
                </Icon>
                <div>
                  <h3 className="text-sm font-semibold group-hover:text-fd-primary transition-colors">
                    {item.title}
                  </h3>
                  <p className="mt-0.5 text-xs text-fd-muted-foreground">
                    {item.description}
                  </p>
                </div>
              </div>
            </Link>
          ))}
        </div>
      </div>
    </main>
  );
}

function DocBadge({ children }: { children: React.ReactNode }) {
  return (
    <span className="rounded-md bg-fd-muted px-2 py-0.5 text-[11px] font-medium text-fd-muted-foreground">
      {children}
    </span>
  );
}

const iconColorMap = {
  blue: 'border-blue-200 bg-blue-50 text-blue-600 dark:border-blue-800 dark:bg-blue-950 dark:text-blue-400',
  purple: 'border-purple-200 bg-purple-50 text-purple-600 dark:border-purple-800 dark:bg-purple-950 dark:text-purple-400',
  green: 'border-green-200 bg-green-50 text-green-600 dark:border-green-800 dark:bg-green-950 dark:text-green-400',
  orange: 'border-orange-200 bg-orange-50 text-orange-600 dark:border-orange-800 dark:bg-orange-950 dark:text-orange-400',
} as const;

function Icon({
  children,
  color = 'blue',
  size = 'md',
}: {
  children: React.ReactNode;
  color?: keyof typeof iconColorMap;
  size?: 'sm' | 'md';
}) {
  const sizeClass = size === 'sm' ? 'size-6 p-1' : 'size-9 p-1.5';
  return (
    <div
      className={`shrink-0 rounded-lg border ${sizeClass} ${iconColorMap[color]}`}
    >
      {children}
    </div>
  );
}

function ItemLink(props: LinkProps & { children: React.ReactNode }) {
  return (
    <Link
      {...props}
      className="group flex flex-col gap-3 rounded-2xl border border-fd-border bg-fd-card p-5 shadow-sm transition-all hover:border-fd-primary/40 hover:shadow-md"
    >
      {props.children}
    </Link>
  );
}

function ItemDisabled({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex flex-col gap-3 rounded-2xl border border-dashed border-fd-border bg-fd-card/50 p-5 opacity-70">
      {children}
    </div>
  );
}
