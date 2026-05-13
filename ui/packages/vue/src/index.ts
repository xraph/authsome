/** @authsome/ui-vue — Vue 3 composables for AuthSome. */

// Re-export core types.
export type {
  AuthState,
  AuthConfig,
  ClientConfig,
  User,
  Session,
  Organization,
  Member,
  MFAEnrollment,
  APIError,
  TokenStorage,
} from "@authsome/ui-core";

// Composables.
export {
  createAuthPlugin,
  useAuth,
  useClientConfig,
  useUser,
  useOrganizations,
  useSessionToken,
  type AuthSubmitOptions,
} from "./composables";
