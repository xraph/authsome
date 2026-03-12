/** @authsome/ui-core — framework-agnostic authentication primitives. */

export * from "./types";
export * from "./client";
export { AuthManager } from "./auth";
export {
  base64urlToBuffer,
  bufferToBase64url,
  prepareCreationOptions,
  prepareRequestOptions,
  serializeCredential,
} from "./webauthn";
