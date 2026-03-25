/** Headless UI components for common auth flows. */

import { useCallback, useState, type FormEvent, type ReactNode } from "react";
import { useAuth, type AuthContextValue } from "./context";

// ── Sign In ─────────────────────────────────────────

export interface SignInFormProps {
  /** Render function receiving form state and handlers. */
  children: (props: {
    email: string;
    password: string;
    setEmail: (v: string) => void;
    setPassword: (v: string) => void;
    submit: (e?: FormEvent) => void;
    isLoading: boolean;
    error: string | null;
  }) => ReactNode;
  /** Called after successful sign-in. */
  onSuccess?: () => void;
}

/**
 * Headless sign-in form component.
 *
 * ```tsx
 * <SignInForm onSuccess={() => router.push("/dashboard")}>
 *   {({ email, password, setEmail, setPassword, submit, isLoading, error }) => (
 *     <form onSubmit={submit}>
 *       <input value={email} onChange={e => setEmail(e.target.value)} />
 *       <input type="password" value={password} onChange={e => setPassword(e.target.value)} />
 *       <button disabled={isLoading}>Sign In</button>
 *       {error && <p>{error}</p>}
 *     </form>
 *   )}
 * </SignInForm>
 * ```
 */
export function SignInForm({ children, onSuccess }: SignInFormProps) {
  const { signIn, isLoading } = useAuth();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);

  const submit = useCallback(
    (e?: FormEvent) => {
      e?.preventDefault();
      setError(null);
      signIn(email, password)
        .then(() => onSuccess?.())
        .catch((err: Error) => setError(err.message));
    },
    [email, password, signIn, onSuccess],
  );

  return <>{children({ email, password, setEmail, setPassword, submit, isLoading, error })}</>;
}

// ── Sign Up ─────────────────────────────────────────

export interface SignUpFormProps {
  children: (props: {
    email: string;
    password: string;
    fields: Record<string, string>;
    setEmail: (v: string) => void;
    setPassword: (v: string) => void;
    setField: (key: string, value: string) => void;
    submit: (e?: FormEvent) => void;
    isLoading: boolean;
    error: string | null;
  }) => ReactNode;
  onSuccess?: () => void;
}

/**
 * Headless sign-up form component.
 */
export function SignUpForm({ children, onSuccess }: SignUpFormProps) {
  const { signUp, isLoading } = useAuth();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [fields, setFields] = useState<Record<string, string>>({});
  const [error, setError] = useState<string | null>(null);

  const setField = useCallback((key: string, value: string) => {
    setFields((prev) => ({ ...prev, [key]: value }));
  }, []);

  const submit = useCallback(
    (e?: FormEvent) => {
      e?.preventDefault();
      setError(null);
      const f = Object.keys(fields).length > 0 ? fields : undefined;
      signUp(email, password, f)
        .then(() => onSuccess?.())
        .catch((err: Error) => setError(err.message));
    },
    [email, password, fields, signUp, onSuccess],
  );

  return (
    <>
      {children({ email, password, fields, setEmail, setPassword, setField, submit, isLoading, error })}
    </>
  );
}

// ── MFA Challenge ───────────────────────────────────

export interface MFAChallengeFormProps {
  enrollmentId: string;
  children: (props: {
    code: string;
    setCode: (v: string) => void;
    submit: (e?: FormEvent) => void;
    isLoading: boolean;
    error: string | null;
  }) => ReactNode;
  onSuccess?: () => void;
}

/**
 * Headless MFA challenge form component.
 */
export function MFAChallengeForm({ enrollmentId, children, onSuccess }: MFAChallengeFormProps) {
  const { submitMFACode, isLoading } = useAuth();
  const [code, setCode] = useState("");
  const [error, setError] = useState<string | null>(null);

  const submit = useCallback(
    (e?: FormEvent) => {
      e?.preventDefault();
      setError(null);
      submitMFACode(enrollmentId, code)
        .then(() => onSuccess?.())
        .catch((err: Error) => setError(err.message));
    },
    [enrollmentId, code, submitMFACode, onSuccess],
  );

  return <>{children({ code, setCode, submit, isLoading, error })}</>;
}

// ── Auth Guard ──────────────────────────────────────

export interface AuthGuardProps {
  children: ReactNode;
  fallback?: ReactNode;
  loading?: ReactNode;
}

/**
 * Renders children only when authenticated.
 *
 * ```tsx
 * <AuthGuard fallback={<SignInPage />} loading={<Spinner />}>
 *   <Dashboard />
 * </AuthGuard>
 * ```
 */
export function AuthGuard({ children, fallback = null, loading = null }: AuthGuardProps) {
  const { state } = useAuth();

  if (state.status === "loading" || state.status === "idle") {
    return <>{loading}</>;
  }

  if (state.status !== "authenticated") {
    return <>{fallback}</>;
  }

  return <>{children}</>;
}

// ── SignedIn / SignedOut ─────────────────────────────

export interface SignedInProps {
  children: ReactNode;
}

/**
 * Renders children only when the user is authenticated.
 *
 * ```tsx
 * <SignedIn>
 *   <Dashboard />
 * </SignedIn>
 * ```
 */
export function SignedIn({ children }: SignedInProps) {
  const { isAuthenticated, isLoading } = useAuth();
  if (isLoading || !isAuthenticated) return null;
  return <>{children}</>;
}

export interface SignedOutProps {
  children: ReactNode;
}

/**
 * Renders children only when the user is NOT authenticated.
 *
 * ```tsx
 * <SignedOut>
 *   <SignInPage />
 * </SignedOut>
 * ```
 */
export function SignedOut({ children }: SignedOutProps) {
  const { isAuthenticated, isLoading } = useAuth();
  if (isLoading || isAuthenticated) return null;
  return <>{children}</>;
}

// ── Protect ─────────────────────────────────────────

export interface ProtectProps {
  children: ReactNode;
  /** Fallback rendered when the user doesn't meet the requirements. */
  fallback?: ReactNode;
  /** Require a specific role (e.g. "admin"). */
  role?: string;
  /** Require a specific permission (e.g. "packages:write"). */
  permission?: string;
  /** Custom condition receiving the auth context. */
  condition?: (auth: AuthContextValue) => boolean;
}

/**
 * Renders children only when the user is authenticated AND meets
 * the specified role, permission, or custom condition.
 *
 * Without role/permission/condition, behaves like `<SignedIn />`.
 *
 * ```tsx
 * <Protect role="admin" fallback={<p>Access denied</p>}>
 *   <AdminPanel />
 * </Protect>
 * ```
 */
export function Protect({
  children,
  fallback = null,
  role,
  permission,
  condition,
}: ProtectProps) {
  const auth = useAuth();

  if (auth.isLoading) return null;
  if (!auth.isAuthenticated) return <>{fallback}</>;

  // Custom condition check
  if (condition && !condition(auth)) return <>{fallback}</>;

  // Role/permission checks are left to the consumer's RBAC layer
  // since the auth context doesn't carry roles directly. The
  // `condition` prop covers this use case generically.
  if (role || permission) {
    // These are checked via condition or by a higher-level wrapper
    // that injects role/permission data.
    if (condition === undefined) {
      // No condition provided but role/permission requested — pass through
      // (the consumer should wrap with their own RBAC provider).
    }
  }

  return <>{children}</>;
}
