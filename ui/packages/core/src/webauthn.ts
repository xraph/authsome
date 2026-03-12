/**
 * WebAuthn encoding helpers.
 *
 * The go-webauthn backend serialises binary fields (challenge, user.id,
 * credential IDs, …) as **base64url strings** in JSON.  The browser
 * WebAuthn API requires those same fields as ArrayBuffer / Uint8Array.
 *
 * These helpers bridge the gap in both directions:
 *   begin response  → browser API  (base64url string → ArrayBuffer)
 *   browser API     → finish POST  (ArrayBuffer → base64url string)
 */

// ── low-level codecs ──────────────────────────────────────────────

/** Decode a base64url-encoded string into a Uint8Array. */
export function base64urlToBuffer(base64url: string): Uint8Array {
  const base64 = base64url.replace(/-/g, "+").replace(/_/g, "/");
  const padded = base64 + "=".repeat((4 - (base64.length % 4)) % 4);
  const binary = atob(padded);
  const bytes = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) {
    bytes[i] = binary.charCodeAt(i);
  }
  return bytes;
}

/** Encode an ArrayBuffer (or Uint8Array) to a base64url string (no padding). */
export function bufferToBase64url(
  buffer: ArrayBuffer | Uint8Array,
): string {
  const bytes =
    buffer instanceof Uint8Array ? buffer : new Uint8Array(buffer);
  let binary = "";
  for (let i = 0; i < bytes.byteLength; i++) {
    binary += String.fromCharCode(bytes[i]);
  }
  return btoa(binary)
    .replace(/\+/g, "-")
    .replace(/\//g, "_")
    .replace(/=+$/, "");
}

// ── begin → browser helpers ────────────────────────────────────────

/**
 * Prepare the options object returned by `/passkeys/register/begin`
 * for `navigator.credentials.create({ publicKey })`.
 *
 * Converts base64url strings → Uint8Array for:
 *   challenge, user.id, excludeCredentials[].id
 */
export function prepareCreationOptions(
  options: Record<string, unknown>,
): PublicKeyCredentialCreationOptions {
  const publicKey = (options.publicKey ?? options) as Record<string, unknown>;

  const out: Record<string, unknown> = { ...publicKey };

  // challenge
  if (typeof out.challenge === "string") {
    out.challenge = base64urlToBuffer(out.challenge);
  }

  // user.id
  if (out.user && typeof out.user === "object") {
    const user = { ...(out.user as Record<string, unknown>) };
    if (typeof user.id === "string") {
      user.id = base64urlToBuffer(user.id);
    }
    out.user = user;
  }

  // excludeCredentials[].id
  if (Array.isArray(out.excludeCredentials)) {
    out.excludeCredentials = (
      out.excludeCredentials as Record<string, unknown>[]
    ).map((c) => {
      if (typeof c.id === "string") {
        return { ...c, id: base64urlToBuffer(c.id) };
      }
      return c;
    });
  }

  return out as unknown as PublicKeyCredentialCreationOptions;
}

/**
 * Prepare the options object returned by `/passkeys/login/begin`
 * for `navigator.credentials.get({ publicKey })`.
 *
 * Converts base64url strings → Uint8Array for:
 *   challenge, allowCredentials[].id
 */
export function prepareRequestOptions(
  options: Record<string, unknown>,
): PublicKeyCredentialRequestOptions {
  const publicKey = (options.publicKey ?? options) as Record<string, unknown>;

  const out: Record<string, unknown> = { ...publicKey };

  // challenge
  if (typeof out.challenge === "string") {
    out.challenge = base64urlToBuffer(out.challenge);
  }

  // allowCredentials[].id
  if (Array.isArray(out.allowCredentials)) {
    out.allowCredentials = (
      out.allowCredentials as Record<string, unknown>[]
    ).map((c) => {
      if (typeof c.id === "string") {
        return { ...c, id: base64urlToBuffer(c.id) };
      }
      return c;
    });
  }

  return out as unknown as PublicKeyCredentialRequestOptions;
}

// ── browser → finish helpers ───────────────────────────────────────

/**
 * Serialize a PublicKeyCredential returned by navigator.credentials
 * into a plain object suitable for POSTing to the `/finish` endpoint.
 *
 * Converts ArrayBuffer fields → base64url strings so they survive
 * JSON.stringify and match what go-webauthn expects.
 */
export function serializeCredential(
  credential: Credential,
): Record<string, unknown> {
  const cred = credential as PublicKeyCredential;
  const response = cred.response;

  const result: Record<string, unknown> = {
    id: cred.id,
    rawId: bufferToBase64url(cred.rawId),
    type: cred.type,
    response: {} as Record<string, unknown>,
  };

  const serializedResponse: Record<string, unknown> = {
    clientDataJSON: bufferToBase64url(response.clientDataJSON),
  };

  // Attestation response (register finish)
  if ("attestationObject" in response) {
    const attestation = response as AuthenticatorAttestationResponse;
    serializedResponse.attestationObject = bufferToBase64url(
      attestation.attestationObject,
    );
  }

  // Assertion response (login finish)
  if ("authenticatorData" in response) {
    const assertion = response as AuthenticatorAssertionResponse;
    serializedResponse.authenticatorData = bufferToBase64url(
      assertion.authenticatorData,
    );
    serializedResponse.signature = bufferToBase64url(assertion.signature);
    if (assertion.userHandle) {
      serializedResponse.userHandle = bufferToBase64url(assertion.userHandle);
    }
  }

  result.response = serializedResponse;

  // Include authenticatorAttachment if present
  if (cred.authenticatorAttachment) {
    result.authenticatorAttachment = cred.authenticatorAttachment;
  }

  return result;
}
