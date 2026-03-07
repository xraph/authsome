/** @authsome/ui-react — React hooks and components for AuthSome. */

// Re-export core types for convenience.
export type {
  AuthState,
  AuthConfig,
  ClientConfig,
  SocialProviderConfig,
  SSOConnectionConfig,
  User,
  Session,
  Organization,
  Member,
  MFAEnrollment,
  APIError,
  TokenStorage,
} from "@authsome/ui-core";

// Context + hooks
export { AuthProvider, useAuth, AuthContext, type AuthProviderProps, type AuthContextValue } from "./context";
export { useUser, useOrganizations, useSessionToken, useClientConfig } from "./hooks";

// Headless components
export {
  SignInForm,
  SignUpForm,
  MFAChallengeForm,
  AuthGuard,
  type SignInFormProps,
  type SignUpFormProps,
  type MFAChallengeFormProps,
  type AuthGuardProps,
} from "./components";
