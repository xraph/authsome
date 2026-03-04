"use client";

import Link from "next/link";
import { ThemedLogo } from "@/components/ui/themed-logo";

const footerLinks = {
  "Getting Started": [
    { label: "Introduction", href: "/docs" },
    { label: "Quick Start", href: "/docs/getting-started" },
    { label: "Architecture", href: "/docs/architecture" },
    { label: "Configuration", href: "/docs/concepts/configuration" },
  ],
  Authentication: [
    { label: "Password", href: "/docs/authentication/password" },
    { label: "Social Login", href: "/docs/authentication/social-login" },
    { label: "Passkeys", href: "/docs/authentication/passkeys" },
    { label: "MFA", href: "/docs/authentication/mfa" },
    { label: "SSO", href: "/docs/authentication/sso" },
  ],
  "UI Packages": [
    { label: "Overview", href: "/docs/ui/overview" },
    { label: "React", href: "/docs/ui/react" },
    { label: "Next.js", href: "/docs/ui/nextjs" },
    { label: "Components", href: "/docs/ui/components" },
    { label: "Playground", href: "/docs/ui/playground" },
  ],
  Community: [
    {
      label: "GitHub",
      href: "https://github.com/xraph/authsome",
      external: true,
    },
    {
      label: "Issues",
      href: "https://github.com/xraph/authsome/issues",
      external: true,
    },
    {
      label: "Discussions",
      href: "https://github.com/xraph/authsome/discussions",
      external: true,
    },
    {
      label: "Contributing",
      href: "https://github.com/xraph/authsome/blob/main/CONTRIBUTING.md",
      external: true,
    },
  ],
};

export function Footer() {
  return (
    <footer className="w-full border-t border-fd-border bg-fd-card/50">
      <div className="container max-w-(--fd-layout-width) mx-auto px-4 sm:px-6">
        {/* Main footer grid */}
        <div className="grid grid-cols-2 gap-8 py-12 sm:py-16 md:grid-cols-5 lg:gap-12">
          {/* Brand column */}
          <div className="col-span-2 md:col-span-1">
            <Link href="/" className="inline-flex items-center gap-2 mb-4">
              <ThemedLogo />
              <span className="font-bold text-lg">Authsome</span>
            </Link>
            <p className="text-sm text-fd-muted-foreground leading-relaxed max-w-xs">
              Composable authentication engine for Go. 14 plugins, 4 store
              backends, 40+ UI components, and Forge-native.
            </p>
            {/* Social links */}
            <div className="flex items-center gap-3 mt-6">
              <a
                href="https://github.com/xraph/authsome"
                target="_blank"
                rel="noreferrer"
                className="text-fd-muted-foreground hover:text-fd-foreground transition-colors"
              >
                <span className="sr-only">GitHub</span>
                <svg
                  className="size-5"
                  fill="currentColor"
                  viewBox="0 0 24 24"
                  aria-hidden="true"
                >
                  <path d="M12 .297c-6.63 0-12 5.373-12 12 0 5.303 3.438 9.8 8.205 11.385.6.113.82-.258.82-.577 0-.285-.01-1.04-.015-2.04-3.338.724-4.042-1.61-4.042-1.61C4.422 18.07 3.633 17.7 3.633 17.7c-1.087-.744.084-.729.084-.729 1.205.084 1.838 1.236 1.838 1.236 1.07 1.835 2.809 1.305 3.495.998.108-.776.417-1.305.76-1.605-2.665-.3-5.466-1.332-5.466-5.93 0-1.31.465-2.38 1.235-3.286-.135-.303-.54-1.523.105-3.176 0 0 1.005-.322 3.3 1.23.96-.267 1.98-.399 3-.405 1.02.006 2.04.138 3 .405 2.28-1.552 3.285-1.23 3.285-1.23.645 1.653.24 2.873.12 3.176.765.84 1.23 1.91 1.23 3.22 0 4.61-2.805 5.625-5.475 5.92.42.36.81 1.096.81 2.22 0 1.606-.015 2.896-.015 3.286 0 .315.21.69.825.57C20.565 22.092 24 17.592 24 12.297c0-6.627-5.373-12-12-12" />
                </svg>
              </a>
              <a
                href="https://x.com/xraph"
                target="_blank"
                rel="noreferrer"
                className="text-fd-muted-foreground hover:text-fd-foreground transition-colors"
              >
                <span className="sr-only">X (Twitter)</span>
                <svg
                  className="size-5"
                  fill="currentColor"
                  viewBox="0 0 24 24"
                  aria-hidden="true"
                >
                  <path d="M18.244 2.25h3.308l-7.227 8.26 8.502 11.24H16.17l-5.214-6.817L4.99 21.75H1.68l7.73-8.835L1.254 2.25H8.08l4.713 6.231zm-1.161 17.52h1.833L7.084 4.126H5.117z" />
                </svg>
              </a>
            </div>
          </div>

          {/* Link columns */}
          {Object.entries(footerLinks).map(([category, links]) => (
            <div key={category}>
              <h3 className="text-sm font-semibold text-fd-foreground mb-4">
                {category}
              </h3>
              <ul className="space-y-2.5">
                {links.map((link) => (
                  <li key={link.label}>
                    {"external" in link && link.external ? (
                      <a
                        href={link.href}
                        target="_blank"
                        rel="noreferrer"
                        className="text-sm text-fd-muted-foreground hover:text-fd-foreground transition-colors"
                      >
                        {link.label}
                      </a>
                    ) : (
                      <Link
                        href={link.href}
                        className="text-sm text-fd-muted-foreground hover:text-fd-foreground transition-colors"
                      >
                        {link.label}
                      </Link>
                    )}
                  </li>
                ))}
              </ul>
            </div>
          ))}
        </div>

        {/* Bottom bar */}
        <div className="border-t border-fd-border py-6 flex flex-col sm:flex-row items-center justify-between gap-4">
          <p className="text-xs text-fd-muted-foreground">
            &copy; {new Date().getFullYear()} Xraph. All rights reserved.
          </p>
          <div className="flex items-center gap-1 text-xs text-fd-muted-foreground">
            <span>Built with</span>
            <span className="inline-block text-rose-400 mx-0.5">
              <svg
                className="size-3.5 inline-block"
                viewBox="0 0 24 24"
                fill="currentColor"
                aria-hidden="true"
              >
                <path d="M11.645 20.91l-.007-.003-.022-.012a15.247 15.247 0 01-.383-.218 25.18 25.18 0 01-4.244-3.17C4.688 15.36 2.25 12.174 2.25 8.25 2.25 5.322 4.714 3 7.688 3A5.5 5.5 0 0112 5.052 5.5 5.5 0 0116.313 3c2.973 0 5.437 2.322 5.437 5.25 0 3.925-2.438 7.111-4.739 9.256a25.175 25.175 0 01-4.244 3.17 15.247 15.247 0 01-.383.219l-.022.012-.007.004-.003.001a.752.752 0 01-.704 0l-.003-.001z" />
              </svg>
            </span>
            <span>and Go</span>
          </div>
        </div>
      </div>
    </footer>
  );
}
