/** Headless UI components for common auth flows. */

import { useCallback, useState, type FormEvent, type ReactNode } from "react";
import { useAuth } from "./context";

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
    name: string;
    setEmail: (v: string) => void;
    setPassword: (v: string) => void;
    setName: (v: string) => void;
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
  const [name, setName] = useState("");
  const [error, setError] = useState<string | null>(null);

  const submit = useCallback(
    (e?: FormEvent) => {
      e?.preventDefault();
      setError(null);
      signUp(email, password, name || undefined)
        .then(() => onSuccess?.())
        .catch((err: Error) => setError(err.message));
    },
    [email, password, name, signUp, onSuccess],
  );

  return (
    <>
      {children({ email, password, name, setEmail, setPassword, setName, submit, isLoading, error })}
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
