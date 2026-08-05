/** @authsome/ui-core — framework-agnostic authentication primitives. */

export * from "./types";
export * from "./client";
export { AuthManager, createLocalStorage, SESSION_STORAGE_KEY } from "./auth";
export {
  isSafeRedirect,
  safeRedirectTarget,
  type SafeRedirectOptions,
} from "./redirect";
export {
  base64urlToBuffer,
  bufferToBase64url,
  prepareCreationOptions,
  prepareRequestOptions,
  serializeCredential,
} from "./webauthn";
