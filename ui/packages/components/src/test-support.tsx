/**
 * Shared scaffolding for this package's component tests.
 *
 * Two levers, deliberately: `withProvider` drives the real AuthProvider ->
 * AuthManager -> AuthClient stack over an injected fetch and storage, which is
 * how a test proves a component works against the shipping code path.
 * `stubAuth` hands back an AuthContextValue the test owns outright, for the
 * cases that need the session to change while the component stays mounted —
 * AuthContext is exported from ui-react for exactly that.
 */

import { AuthClient, AuthManager } from "@authsome/ui-core";
import type {
  AuthState,
  ClientConfig,
  Session,
  TokenStorage,
  User,
} from "@authsome/ui-core";
import {
  AuthContext,
  AuthProvider,
  type AuthContextValue,
} from "@authsome/ui-react";
import type { ReactElement, ReactNode } from "react";

export const BASE = "https://api.example.test";

/** A session far enough from expiry that the manager will not try to refresh. */
export function makeSession(token = "tok"): Session {
  return {
    session_token: token,
    refresh_token: `${token}-refresh`,
    expires_at: new Date(Date.now() + 3_600_000).toISOString(),
  };
}

export function makeUser(overrides: Partial<User> = {}): User {
  return {
    id: "usr_1",
    app_id: "app_1",
    env_id: "env_1",
    email: "ada@test",
    email_verified: true,
    first_name: "Ada",
    last_name: "Lovelace",
    banned: false,
    phone_verified: false,
    created_at: new Date(0).toISOString(),
    updated_at: new Date(0).toISOString(),
    ...overrides,
  } as User;
}

export function json(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

/** Handler for one "METHOD /path" route. Return a Response or a plain body. */
export type Route = (req: {
  path: string;
  method: string;
  token: string | null;
}) => unknown | Promise<unknown>;

export interface RoutedFetch {
  fetchFn: typeof globalThis.fetch;
  /** Every "METHOD /path" the component actually requested, in order. */
  calls: string[];
  /** The same requests with their query strings, for asserting on params. */
  urls: string[];
}

/**
 * A fetch that dispatches on "METHOD /path". Anything unrouted 404s loudly
 * rather than hanging, so a missing route shows up as a failed assertion
 * instead of a timeout.
 */
export function routedFetch(routes: Record<string, Route>): RoutedFetch {
  const calls: string[] = [];
  const urls: string[] = [];

  // AuthManager.initialize() fetches the profile before it will report a
  // session, so every component test needs this route. Defaulting it here
  // keeps it out of the tests, which care about their own endpoint. A test
  // that wants a different profile, or a failing one, just overrides it.
  const table: Record<string, Route> = { "GET /v1/me": () => makeUser(), ...routes };

  const fetchFn = (async (
    input: string | URL | Request,
    init?: RequestInit,
  ): Promise<Response> => {
    const url = new URL(
      String(input instanceof Request ? input.url : input),
      BASE,
    );
    const method = (init?.method ?? "GET").toUpperCase();
    const key = `${method} ${url.pathname}`;
    calls.push(key);
    urls.push(`${method} ${url.pathname}${url.search}`);

    const handler = table[key];
    if (!handler) return json({ error: `no route for ${key}` }, 404);

    const auth = new Headers(init?.headers).get("Authorization");
    // Await it: a handler that holds a request open (to make the in-flight
    // state observable) returns a promise, and stringifying that yields "{}".
    const result = await handler({
      path: url.pathname,
      method,
      token: auth?.replace(/^Bearer /, "") ?? null,
    });
    return result instanceof Response ? result : json(result);
  }) as typeof globalThis.fetch;

  return { fetchFn, calls, urls };
}

export function memoryStorage(seed?: Session): TokenStorage {
  const store = new Map<string, string>();
  if (seed) store.set("authsome:session", JSON.stringify(seed));
  return {
    getItem: (k) => store.get(k) ?? null,
    setItem: (k, v) => void store.set(k, v),
    removeItem: (k) => void store.delete(k),
  };
}

/** Wrap children in a real AuthProvider over an injected fetch and storage. */
export function withProvider(
  children: ReactNode,
  opts: {
    fetch: typeof globalThis.fetch;
    session?: Session | null;
    /** Seeds the manager's client config, so useClientConfig has something
     * to report without a discovery round trip. */
    clientConfig?: ClientConfig;
  },
): ReactElement {
  const session = opts.session === undefined ? makeSession() : opts.session;
  return (
    <AuthProvider
      baseURL={BASE}
      fetch={opts.fetch}
      storage={memoryStorage(session ?? undefined)}
      initialClientConfig={opts.clientConfig}
    >
      {children}
    </AuthProvider>
  );
}

function unsupported(name: string): () => never {
  return () => {
    throw new Error(`${name} is not wired in this test`);
  };
}

/**
 * A complete AuthContextValue the test controls. Only `client` and `session`
 * matter to the components under test here; the rest are present because the
 * type demands them and throw if something reaches for them unexpectedly.
 */
export function stubAuth(opts: {
  fetch: typeof globalThis.fetch;
  session: Session | null;
  user?: User | null;
  /**
   * Reuse a client across values. A real token change comes from the same
   * AuthManager and therefore the same client, so a test that swaps the
   * session must hold the client identity steady or it is also changing a
   * dependency the component keys its fetch on.
   */
  client?: AuthClient;
}): AuthContextValue {
  const client =
    opts.client ?? new AuthClient({ baseURL: BASE, fetch: opts.fetch });
  const user = opts.user === undefined ? makeUser() : opts.user;
  const state: AuthState =
    opts.session && user
      ? { status: "authenticated", user, session: opts.session }
      : { status: "unauthenticated" };

  return {
    state,
    manager: new AuthManager({ baseURL: BASE, fetch: opts.fetch }),
    client,
    user: opts.session ? user : null,
    session: opts.session,
    isAuthenticated: Boolean(opts.session),
    isLoading: false,
    clientConfig: null,
    isConfigLoaded: true,
    signIn: unsupported("signIn"),
    signUp: unsupported("signUp"),
    signOut: unsupported("signOut"),
    resendVerification: unsupported("resendVerification"),
    submitMFAChallenge: unsupported("submitMFAChallenge"),
    submitMFACode: unsupported("submitMFACode"),
    submitRecoveryCode: unsupported("submitRecoveryCode"),
    sendSMSCode: unsupported("sendSMSCode"),
    submitSMSCode: unsupported("submitSMSCode"),
  };
}

/** Wrap children in a caller-owned context value. */
export function withAuth(
  children: ReactNode,
  value: AuthContextValue,
): ReactElement {
  return (
    <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
  );
}
