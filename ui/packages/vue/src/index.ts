/** @authsome/ui-vue — Vue 3 composables for AuthSome. */

// Re-export core types.
export type {
  AuthState,
  AuthConfig,
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
  useUser,
  useOrganizations,
  useSessionToken,
} from "./composables";
