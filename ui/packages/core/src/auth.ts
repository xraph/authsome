/**
 * Framework-agnostic authentication state machine.
 *
 * Manages sign-in/sign-up flows, token persistence, automatic refresh,
 * and MFA challenges. Framework adapters (React, Vue, etc.) wrap this
 * class and expose reactive state.
 */

import { AuthClient, type SignInRequest, type SignUpRequest, AuthClientError } from "./client";
import type { AuthConfig, AuthState, ClientConfig, Session, TokenStorage, User } from "./types";

const SESSION_KEY = "authsome:session";
const CONFIG_KEY = "authsome:client_config";
const REFRESH_BEFORE_MS = 60_000; // Refresh 60 s before expiry.
const CONFIG_TTL_MS = 5 * 60_000; // Cache client config for 5 minutes.

/** Default in-memory storage (lost on page reload). */
const memoryStorage: TokenStorage = (() => {
  const store = new Map<string, string>();
  return {
    getItem: (key: string) => store.get(key) ?? null,
    setItem: (key: string, value: string) => {
      store.set(key, value);
    },
    removeItem: (key: string) => {
      store.delete(key);
    },
  };
})();

/** Try to use localStorage, fall back to memory. */
function defaultStorage(): TokenStorage {
  try {
    if (typeof window !== "undefined" && window.localStorage) {
      return window.localStorage;
    }
  } catch {
    // SSR or restricted environments.
  }
  return memoryStorage;
}

/**
 * AuthManager is the core state machine that drives authentication.
 *
 * Usage:
 * ```ts
 * const auth = new AuthManager({ baseURL: "https://api.example.com" });
 * auth.subscribe((state) => console.log(state));
 * await auth.initialize(); // hydrate from storage
 * await auth.signIn({ email, password });
 * ```
 */
export class AuthManager {
  private client: AuthClient;
  private storage: TokenStorage;
  private state: AuthState = { status: "idle" };
  private listeners = new Set<(state: AuthState) => void>();
  private refreshTimer: ReturnType<typeof setTimeout> | null = null;
  private onError?: (error: { error: string; code?: number }) => void;

  private publishableKey?: string;
  private clientConfig: ClientConfig | null = null;
  private configListeners = new Set<(config: ClientConfig | null) => void>();
  private configFetchPromise: Promise<ClientConfig> | null = null;

  constructor(config: AuthConfig) {
    this.client = new AuthClient(config);
    this.storage = config.storage ?? defaultStorage();
    this.onError = config.onError;
    this.publishableKey = config.publishableKey;

    if (config.initialClientConfig) {
      this.clientConfig = config.initialClientConfig;
    }

    if (config.onStateChange) {
      this.listeners.add(config.onStateChange);
    }
  }

  // ── Public API ────────────────────────────────────

  /** Current auth state (snapshot). */
  getState(): AuthState {
    return this.state;
  }

  /** Subscribe to state changes. Returns an unsubscribe function. */
  subscribe(listener: (state: AuthState) => void): () => void {
    this.listeners.add(listener);
    return () => {
      this.listeners.delete(listener);
    };
  }

  /** Access the underlying HTTP client. */
  getClient(): AuthClient {
    return this.client;
  }

  /**
   * Initialize by hydrating the session from storage.
   * When a publishableKey is set, also fetches client config in parallel.
   * Call this once on app start.
   */
  async initialize(): Promise<void> {
    // Kick off config fetch in parallel (non-blocking).
    if (this.publishableKey && !this.clientConfig) {
      void this.fetchClientConfig();
    }

    try {
      const raw = await this.storage.getItem(SESSION_KEY);
      if (!raw) {
        this.setState({ status: "unauthenticated" });
        return;
      }

      const session: Session = JSON.parse(raw);
      const expiresAt = new Date(session.expires_at).getTime();

      if (Date.now() >= expiresAt) {
        // Token expired — try refresh.
        await this.refreshSession(session.refresh_token);
        return;
      }

      // Token still valid — fetch user profile.
      this.setState({ status: "loading" });
      const user = await this.client.getMe(session.session_token);
      this.setState({ status: "authenticated", user, session });
      this.scheduleRefresh(session);
    } catch {
      await this.clearSession();
      this.setState({ status: "unauthenticated" });
    }
  }

  /** Sign in with email & password. */
  async signIn(credentials: SignInRequest): Promise<void> {
    this.setState({ status: "loading" });
    try {
      const res = await this.client.signIn(credentials);
      const session: Session = {
        session_token: res.session_token,
        refresh_token: res.refresh_token,
        expires_at: new Date(Date.now() + 3600_000).toISOString(), // fallback
      };
      await this.handleAuthResponse(res.user, session);
    } catch (err) {
      this.handleError(err);
    }
  }

  /** Sign up with email & password. */
  async signUp(data: SignUpRequest): Promise<void> {
    this.setState({ status: "loading" });
    try {
      const res = await this.client.signUp(data);
      const session: Session = {
        session_token: res.session_token,
        refresh_token: res.refresh_token,
        expires_at: new Date(Date.now() + 3600_000).toISOString(),
      };
      await this.handleAuthResponse(res.user, session);
    } catch (err) {
      this.handleError(err);
    }
  }

  /** Sign out and clear the session. */
  async signOut(): Promise<void> {
    const token = this.getSessionToken();
    if (token) {
      try {
        await this.client.signOut(token);
      } catch {
        // Best-effort server sign-out.
      }
    }
    this.clearRefreshTimer();
    await this.clearSession();
    this.setState({ status: "unauthenticated" });
  }

  /** Submit an MFA challenge code. */
  async submitMFACode(enrollmentId: string, code: string): Promise<void> {
    this.setState({ status: "loading" });
    try {
      const res = await this.client.mfaChallenge({ enrollment_id: enrollmentId, code });
      const session: Session = {
        session_token: res.session_token,
        refresh_token: res.refresh_token,
        expires_at: new Date(Date.now() + 3600_000).toISOString(),
      };
      await this.handleAuthResponse(res.user, session);
    } catch (err) {
      this.handleError(err);
    }
  }

  /** Submit an MFA recovery code. */
  async submitRecoveryCode(code: string): Promise<void> {
    this.setState({ status: "loading" });
    try {
      const res = await this.client.verifyRecoveryCode(code);
      const session: Session = {
        session_token: res.session_token,
        refresh_token: res.refresh_token,
        expires_at: new Date(Date.now() + 3600_000).toISOString(),
      };
      await this.handleAuthResponse(res.user, session);
    } catch (err) {
      this.handleError(err);
    }
  }

  /** Send an SMS code for MFA verification. Returns masked phone + expiry info. */
  async sendSMSCode(): Promise<{ sent: boolean; phone_masked: string; expires_in_seconds: number }> {
    const token = this.getSessionToken();
    if (!token) throw new Error("No session token available");
    return this.client.sendSMSCodeForMFA(token);
  }

  /** Submit an SMS verification code during MFA challenge. */
  async submitSMSCode(code: string): Promise<void> {
    this.setState({ status: "loading" });
    try {
      const token = this.getSessionToken();
      if (!token) throw new Error("No session token available");
      const res = await this.client.verifySMSCodeForMFA(code, token);
      const session: Session = {
        session_token: res.session_token,
        refresh_token: res.refresh_token,
        expires_at: new Date(Date.now() + 3600_000).toISOString(),
      };
      await this.handleAuthResponse(res.user, session);
    } catch (err) {
      this.handleError(err);
    }
  }

  /** Refresh the current session manually. */
  async refreshNow(): Promise<void> {
    const state = this.state;
    if (state.status !== "authenticated") return;
    await this.refreshSession(state.session.refresh_token);
  }

  /** Get the current session token (if authenticated). */
  getSessionToken(): string | null {
    if (this.state.status === "authenticated") {
      return this.state.session.session_token;
    }
    if (this.state.status === "mfa_required") {
      return this.state.session.session_token;
    }
    return null;
  }

  /** Get the current user (if authenticated). */
  getUser(): User | null {
    if (this.state.status === "authenticated") {
      return this.state.user;
    }
    return null;
  }

  // ── Client Config API ────────────────────────────

  /** Get the cached client config (or null if not yet fetched). */
  getClientConfig(): ClientConfig | null {
    return this.clientConfig;
  }

  /** Subscribe to client config changes. Returns an unsubscribe function. */
  subscribeConfig(listener: (config: ClientConfig | null) => void): () => void {
    this.configListeners.add(listener);
    return () => {
      this.configListeners.delete(listener);
    };
  }

  /**
   * Fetch client config from the backend.
   * Deduplicates concurrent calls and caches the result with a TTL.
   */
  async fetchClientConfig(): Promise<ClientConfig> {
    // Deduplicate concurrent fetches.
    if (this.configFetchPromise) {
      return this.configFetchPromise;
    }

    // Check storage cache.
    try {
      const cached = await this.storage.getItem(CONFIG_KEY);
      if (cached) {
        const { config, fetchedAt } = JSON.parse(cached) as { config: ClientConfig; fetchedAt: number };
        if (Date.now() - fetchedAt < CONFIG_TTL_MS) {
          this.setClientConfig(config);
          return config;
        }
      }
    } catch {
      // Cache miss or parse error — fetch fresh.
    }

    this.configFetchPromise = this.client
      .fetchClientConfig(this.publishableKey)
      .then(async (config) => {
        this.setClientConfig(config);
        // Cache with timestamp.
        try {
          await this.storage.setItem(
            CONFIG_KEY,
            JSON.stringify({ config, fetchedAt: Date.now() }),
          );
        } catch {
          // Storage write failure is non-fatal.
        }
        return config;
      })
      .finally(() => {
        this.configFetchPromise = null;
      });

    return this.configFetchPromise;
  }

  /** Tear down: clear timers and listeners. */
  destroy(): void {
    this.clearRefreshTimer();
    this.listeners.clear();
    this.configListeners.clear();
  }

  // ── Internals ─────────────────────────────────────

  private async handleAuthResponse(user: User, session: Session): Promise<void> {
    await this.persistSession(session);
    this.setState({ status: "authenticated", user, session });
    this.scheduleRefresh(session);
  }

  private async refreshSession(refreshToken: string): Promise<void> {
    try {
      const newSession = await this.client.refresh(refreshToken);
      const user = await this.client.getMe(newSession.session_token);
      await this.persistSession(newSession);
      this.setState({ status: "authenticated", user, session: newSession });
      this.scheduleRefresh(newSession);
    } catch {
      await this.clearSession();
      this.setState({ status: "unauthenticated" });
    }
  }

  private scheduleRefresh(session: Session): void {
    this.clearRefreshTimer();
    const expiresAt = new Date(session.expires_at).getTime();
    const delay = expiresAt - Date.now() - REFRESH_BEFORE_MS;

    if (delay <= 0) {
      // Already near expiry — refresh immediately.
      void this.refreshSession(session.refresh_token);
      return;
    }

    this.refreshTimer = setTimeout(() => {
      void this.refreshSession(session.refresh_token);
    }, delay);
  }

  private clearRefreshTimer(): void {
    if (this.refreshTimer !== null) {
      clearTimeout(this.refreshTimer);
      this.refreshTimer = null;
    }
  }

  private async persistSession(session: Session): Promise<void> {
    await this.storage.setItem(SESSION_KEY, JSON.stringify(session));
  }

  private async clearSession(): Promise<void> {
    await this.storage.removeItem(SESSION_KEY);
  }

  private setClientConfig(config: ClientConfig): void {
    this.clientConfig = config;
    for (const listener of this.configListeners) {
      try {
        listener(config);
      } catch {
        // Listener errors should not break the config flow.
      }
    }
  }

  private setState(newState: AuthState): void {
    this.state = newState;
    for (const listener of this.listeners) {
      try {
        listener(newState);
      } catch {
        // Listener errors should not break the state machine.
      }
    }
  }

  private handleError(err: unknown): void {
    const message = err instanceof AuthClientError ? err.message : "An unexpected error occurred";
    const code = err instanceof AuthClientError ? err.code : undefined;

    // MFA required is returned as a specific error code.
    if (code === 403 && message.toLowerCase().includes("mfa")) {
      const token = this.getSessionToken();
      if (token) {
        this.setState({
          status: "mfa_required",
          session: {
            session_token: token,
            refresh_token: "",
            expires_at: new Date(Date.now() + 300_000).toISOString(),
          },
        });
        return;
      }
    }

    this.setState({ status: "error", error: message });
    this.onError?.({ error: message, code });
  }
}
