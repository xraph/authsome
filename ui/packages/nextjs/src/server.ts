/**
 * Server-side utilities for Next.js (App Router / RSC).
 *
 * Usage in a Server Component:
 * ```ts
 * import { getServerSession } from "@authsome/ui-nextjs";
 *
 * export default async function Page() {
 *   const session = await getServerSession({ baseURL: process.env.AUTHSOME_API_URL! });
 *   if (!session) redirect("/sign-in");
 *   return <Dashboard user={session.user} />;
 * }
 * ```
 */

import { cookies } from "next/headers";
import { AuthClient, type ClientConfig, type User } from "@authsome/ui-core";

const SESSION_COOKIE = "authsome_session_token";

/** Server session containing the validated user. */
export interface ServerSession {
  user: User;
  sessionToken: string;
}

/** Options for getServerSession. */
export interface GetServerSessionOptions {
  /** Base URL of the AuthSome API. */
  baseURL: string;
  /** Cookie name (default: "authsome_session_token"). */
  cookieName?: string;
}

/**
 * Reads the session cookie and validates it server-side.
 * Returns `null` if there is no valid session.
 */
export async function getServerSession(
  opts: GetServerSessionOptions,
): Promise<ServerSession | null> {
  const cookieName = opts.cookieName ?? SESSION_COOKIE;
  const cookieStore = await cookies();
  const sessionToken = cookieStore.get(cookieName)?.value;

  if (!sessionToken) {
    return null;
  }

  try {
    const client = new AuthClient({ baseURL: opts.baseURL });
    const user = await client.getMe(sessionToken);
    return { user, sessionToken };
  } catch {
    return null;
  }
}

// ── Client Config ──────────────────────────────────

/** Options for getClientConfig. */
export interface GetClientConfigOptions {
  /** Base URL of the AuthSome API. */
  baseURL: string;
  /** Publishable key to identify the app. */
  publishableKey?: string;
}

/**
 * Fetches the client config server-side with ISR caching (5 min revalidation).
 *
 * Use this to pre-fetch config in a Server Component and pass it as
 * `initialClientConfig` to `AuthProvider`, avoiding a client-side fetch.
 *
 * ```ts
 * import { getClientConfig } from "@authsome/ui-nextjs";
 *
 * export default async function Layout({ children }) {
 *   const config = await getClientConfig({
 *     baseURL: process.env.AUTHSOME_API_URL!,
 *     publishableKey: process.env.NEXT_PUBLIC_AUTHSOME_KEY!,
 *   });
 *   return (
 *     <AuthProvider
 *       baseURL={process.env.AUTHSOME_API_URL!}
 *       publishableKey={process.env.NEXT_PUBLIC_AUTHSOME_KEY!}
 *       initialClientConfig={config ?? undefined}
 *     >
 *       {children}
 *     </AuthProvider>
 *   );
 * }
 * ```
 */
export async function getClientConfig(
  opts: GetClientConfigOptions,
): Promise<ClientConfig | null> {
  const url = new URL("/v1/auth/client-config", opts.baseURL);
  if (opts.publishableKey) {
    url.searchParams.set("key", opts.publishableKey);
  }

  try {
    const fetchOpts: RequestInit & Record<string, unknown> = {
      headers: { "Content-Type": "application/json" },
      next: { revalidate: 300 },
    };
    const res = await fetch(url.toString(), fetchOpts as RequestInit);
    if (!res.ok) return null;
    return (await res.json()) as ClientConfig;
  } catch {
    return null;
  }
}
