/** @authsome/ui-nextjs — Next.js integration for AuthSome. */

// Re-export React package for convenience.
export {
  AuthProvider,
  useAuth,
  useUser,
  useOrganizations,
  useSessionToken,
  useClientConfig,
  SignInForm,
  SignUpForm,
  MFAChallengeForm,
  AuthGuard,
} from "@authsome/ui-react";

// Re-export core types.
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
  TokenStorage,
} from "@authsome/ui-core";

// Next.js-specific exports.
export {
  getServerSession,
  getClientConfig,
  type ServerSession,
  type GetServerSessionOptions,
  type GetClientConfigOptions,
} from "./server";
export { createCookieStorage, type CookieStorageOptions } from "./cookie-storage";
export { createProxyHandler, type ProxyHandlerConfig } from "./proxy";
