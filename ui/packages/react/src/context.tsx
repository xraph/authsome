/** React context and provider for AuthSome authentication. */

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from "react";
import {
  AuthManager,
  type AuthConfig,
  type AuthState,
  type AuthClient,
  type ClientConfig,
  type User,
  type Session,
} from "@authsome/ui-core";

/** The value exposed by AuthContext. */
export interface AuthContextValue {
  /** Current authentication state. */
  state: AuthState;
  /** The underlying AuthManager instance. */
  manager: AuthManager;
  /** The HTTP API client. */
  client: AuthClient;
  /** Convenience: current user or null. */
  user: User | null;
  /** Convenience: current session or null. */
  session: Session | null;
  /** Convenience: whether the user is authenticated. */
  isAuthenticated: boolean;
  /** Convenience: whether auth is loading. */
  isLoading: boolean;
  /** Auto-discovered client configuration (null until loaded). */
  clientConfig: ClientConfig | null;
  /** Whether the client config has been loaded. */
  isConfigLoaded: boolean;

  /** Sign in with email & password. */
  signIn: (email: string, password: string) => Promise<void>;
  /** Sign up with email & password and optional extra fields. */
  signUp: (email: string, password: string, fields?: Record<string, string>) => Promise<void>;
  /** Sign out. */
  signOut: () => Promise<void>;
  /** Submit MFA code. */
  submitMFACode: (enrollmentId: string, code: string) => Promise<void>;
  /** Submit MFA recovery code. */
  submitRecoveryCode: (code: string) => Promise<void>;
  /** Send an SMS code for MFA verification. */
  sendSMSCode: () => Promise<{ sent: boolean; phone_masked: string; expires_in_seconds: number }>;
  /** Submit an SMS verification code for MFA. */
  submitSMSCode: (code: string) => Promise<void>;
}

/** The internal auth context. Exported for testing/mocking purposes. */
export const AuthContext = createContext<AuthContextValue | null>(null);

/** Props for `AuthProvider`. */
export interface AuthProviderProps extends AuthConfig {
  children: ReactNode;
}

/**
 * AuthProvider wraps your app and provides authentication state.
 *
 * ```tsx
 * <AuthProvider baseURL="https://api.example.com">
 *   <App />
 * </AuthProvider>
 * ```
 */
export function AuthProvider({ children, ...config }: AuthProviderProps) {
  const managerRef = useRef<AuthManager | null>(null);
  if (managerRef.current === null) {
    managerRef.current = new AuthManager(config);
  }
  const manager = managerRef.current;

  const [state, setState] = useState<AuthState>(manager.getState());
  const [clientConfig, setClientConfig] = useState<ClientConfig | null>(
    manager.getClientConfig(),
  );

  useEffect(() => {
    const unsubscribe = manager.subscribe(setState);
    const unsubscribeConfig = manager.subscribeConfig(setClientConfig);
    // Hydrate from storage on mount.
    void manager.initialize();
    return () => {
      unsubscribe();
      unsubscribeConfig();
      manager.destroy();
    };
  }, [manager]);

  const signIn = useCallback(
    (email: string, password: string) => manager.signIn({ email, password }),
    [manager],
  );

  const signUp = useCallback(
    (email: string, password: string, fields?: Record<string, string>) => {
      const { first_name, last_name, username, ...rest } = fields ?? {};
      const metadata = Object.keys(rest).length > 0 ? rest : undefined;
      return manager.signUp({ email, password, first_name, last_name, username, metadata });
    },
    [manager],
  );

  const signOut = useCallback(() => manager.signOut(), [manager]);

  const submitMFACode = useCallback(
    (enrollmentId: string, code: string) => manager.submitMFACode(enrollmentId, code),
    [manager],
  );

  const submitRecoveryCode = useCallback(
    (code: string) => manager.submitRecoveryCode(code),
    [manager],
  );

  const sendSMSCode = useCallback(
    () => manager.sendSMSCode(),
    [manager],
  );

  const submitSMSCode = useCallback(
    (code: string) => manager.submitSMSCode(code),
    [manager],
  );

  const value = useMemo<AuthContextValue>(() => {
    const user = state.status === "authenticated" ? state.user : null;
    const session = state.status === "authenticated" ? state.session : null;

    return {
      state,
      manager,
      client: manager.getClient(),
      user,
      session,
      isAuthenticated: state.status === "authenticated",
      isLoading: state.status === "loading",
      clientConfig,
      isConfigLoaded: clientConfig !== null,
      signIn,
      signUp,
      signOut,
      submitMFACode,
      submitRecoveryCode,
      sendSMSCode,
      submitSMSCode,
    };
  }, [state, manager, clientConfig, signIn, signUp, signOut, submitMFACode, submitRecoveryCode, sendSMSCode, submitSMSCode]);

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

/**
 * useAuth returns the full authentication context.
 * Must be used inside an `<AuthProvider>`.
 */
export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) {
    throw new Error("useAuth must be used within an <AuthProvider>");
  }
  return ctx;
}
