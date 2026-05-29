/// WebAuthn encoding helpers.
///
/// The go-webauthn backend serialises binary fields (challenge, user.id,
/// credential IDs, …) as **base64url strings** in JSON. The browser
/// WebAuthn API (and `passkeys` corbado package on native) require those
/// same fields as raw bytes (`Uint8List` / `ArrayBuffer`).
///
/// These helpers bridge the gap in both directions:
///   begin response  → platform API  (base64url string → Uint8List)
///   platform API    → finish POST   (Uint8List → base64url string)
///
/// Ports `ui/packages/core/src/webauthn.ts` 1:1.
library;

import 'dart:convert';
import 'dart:typed_data';

// ── low-level codecs ──────────────────────────────────────────────

/// Decode a base64url-encoded string into a [Uint8List].
Uint8List base64urlToBytes(String base64url) {
  // Restore the standard alphabet and re-pad before calling `base64.decode`.
  var s = base64url.replaceAll('-', '+').replaceAll('_', '/');
  final pad = (4 - (s.length % 4)) % 4;
  s = s + ('=' * pad);
  return base64.decode(s);
}

/// Encode bytes to a base64url string (no padding).
String bytesToBase64url(Uint8List bytes) {
  return base64
      .encode(bytes)
      .replaceAll('+', '-')
      .replaceAll('/', '_')
      .replaceAll('=', '');
}

// ── begin → platform helpers ───────────────────────────────────────

/// Prepare the options object returned by `/v1/passkeys/login/begin`
/// for `navigator.credentials.get({ publicKey })` (Web) or its native
/// equivalent.
///
/// Converts base64url strings → [Uint8List] for:
///   challenge, allowCredentials[].id
///
/// The server may return options nested under `publicKey` or flat —
/// both shapes are accepted to match the React behaviour.
Map<String, dynamic> prepareRequestOptions(Map<String, dynamic> options) {
  final publicKey = (options['publicKey'] is Map)
      ? Map<String, dynamic>.from(options['publicKey'] as Map)
      : Map<String, dynamic>.from(options);

  final out = <String, dynamic>{...publicKey};

  final challenge = out['challenge'];
  if (challenge is String) {
    out['challenge'] = base64urlToBytes(challenge);
  }

  final allow = out['allowCredentials'];
  if (allow is List) {
    out['allowCredentials'] = allow.map((entry) {
      if (entry is Map) {
        final m = Map<String, dynamic>.from(entry);
        final id = m['id'];
        if (id is String) m['id'] = base64urlToBytes(id);
        return m;
      }
      return entry;
    }).toList();
  }

  return out;
}

/// Prepare the options object returned by `/v1/passkeys/register/begin`
/// for `navigator.credentials.create({ publicKey })`.
///
/// Converts base64url strings → [Uint8List] for:
///   challenge, user.id, excludeCredentials[].id
Map<String, dynamic> prepareCreationOptions(Map<String, dynamic> options) {
  final publicKey = (options['publicKey'] is Map)
      ? Map<String, dynamic>.from(options['publicKey'] as Map)
      : Map<String, dynamic>.from(options);

  final out = <String, dynamic>{...publicKey};

  final challenge = out['challenge'];
  if (challenge is String) {
    out['challenge'] = base64urlToBytes(challenge);
  }

  final user = out['user'];
  if (user is Map) {
    final u = Map<String, dynamic>.from(user);
    final id = u['id'];
    if (id is String) u['id'] = base64urlToBytes(id);
    out['user'] = u;
  }

  final exclude = out['excludeCredentials'];
  if (exclude is List) {
    out['excludeCredentials'] = exclude.map((entry) {
      if (entry is Map) {
        final m = Map<String, dynamic>.from(entry);
        final id = m['id'];
        if (id is String) m['id'] = base64urlToBytes(id);
        return m;
      }
      return entry;
    }).toList();
  }

  return out;
}

// ── platform → finish helpers ──────────────────────────────────────

/// Build the credential JSON payload expected by the `/finish` endpoints.
///
/// Mirrors `serializeCredential` from `webauthn.ts`: takes raw byte fields
/// returned by the platform's WebAuthn API and re-encodes them as
/// base64url strings so they survive `jsonEncode` and match what
/// go-webauthn expects on the wire.
///
/// Inputs are deliberately untyped (`dynamic`) because we accept either
/// [Uint8List] or already-encoded [String] values — handy when bridging
/// from packages that pre-encode (e.g. corbado `passkeys`).
Map<String, dynamic> serializeAssertion({
  required String id,
  required dynamic rawId,
  required String type,
  required dynamic clientDataJson,
  required dynamic authenticatorData,
  required dynamic signature,
  dynamic userHandle,
  String? authenticatorAttachment,
}) {
  String enc(dynamic v) {
    if (v is String) return v;
    if (v is Uint8List) return bytesToBase64url(v);
    if (v is List<int>) return bytesToBase64url(Uint8List.fromList(v));
    throw ArgumentError(
      'Expected Uint8List, List<int>, or base64url String, got '
      '${v.runtimeType}',
    );
  }

  final response = <String, dynamic>{
    'clientDataJSON': enc(clientDataJson),
    'authenticatorData': enc(authenticatorData),
    'signature': enc(signature),
  };
  if (userHandle != null) {
    response['userHandle'] = enc(userHandle);
  }

  final out = <String, dynamic>{
    'id': id,
    'rawId': enc(rawId),
    'type': type,
    'response': response,
  };
  if (authenticatorAttachment != null) {
    out['authenticatorAttachment'] = authenticatorAttachment;
  }
  return out;
}
