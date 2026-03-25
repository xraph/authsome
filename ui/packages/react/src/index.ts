/** @authsome/ui-react — React hooks and components for AuthSome. */

// Re-export core types for convenience.
export type {
  AuthState,
  AuthConfig,
  ClientConfig,
  SocialProviderConfig,
  SSOConnectionConfig,
  SignupFieldConfig,
  SignupFieldValidation,
  SignupFieldOption,
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
  SignedIn,
  SignedOut,
  Protect,
  type SignInFormProps,
  type SignUpFormProps,
  type MFAChallengeFormProps,
  type AuthGuardProps,
  type SignedInProps,
  type SignedOutProps,
  type ProtectProps,
} from "./components";
