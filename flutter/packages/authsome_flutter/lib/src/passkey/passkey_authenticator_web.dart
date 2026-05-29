/// Flutter Web implementation of [PasskeyAuthenticator] — drives
/// `navigator.credentials.get` via the `package:web` typed bindings and
/// `dart:js_interop`. Mirrors the React `passkey-login-button.tsx` flow.
library;

import 'dart:async';
import 'dart:js_interop';
import 'dart:js_interop_unsafe';
import 'dart:typed_data';

import 'package:authsome_core/authsome_core.dart';
import 'package:web/web.dart' as web;

class PlatformPasskeyAuthenticator implements PasskeyAuthenticator {
  const PlatformPasskeyAuthenticator();

  @override
  bool get isAvailable {
    // PublicKeyCredential is the canonical WebAuthn feature-detect;
    // missing implies no support (very old browsers / non-secure
    // contexts / browsers that disable WebAuthn). globalContext lookup
    // is the reliable cross-renderer (HTML + Wasm) way to read globals.
    try {
      return globalContext.has('PublicKeyCredential');
    } catch (_) {
      return false;
    }
  }

  @override
  Future<Map<String, dynamic>> authenticate(
    Map<String, dynamic> options,
  ) async {
    if (!isAvailable) {
      throw const AuthClientException(
        'WebAuthn is not available in this browser',
        code: 400,
      );
    }

    // `options` arrives from `prepareRequestOptions`, so binary fields
    // are already `Uint8List`. JS interop expects `JSAny` — wrap into a
    // plain object with `BufferSource` (ArrayBuffer/TypedArray) values.
    final publicKey = _toJSObject(options);

    final request = web.CredentialRequestOptions(
      publicKey: publicKey as web.PublicKeyCredentialRequestOptions,
    );

    final result = await web.window.navigator.credentials
        .get(request)
        .toDart;
    if (result == null) {
      throw const AuthClientException(
        'No credential returned from authenticator',
        code: 400,
      );
    }

    final cred = result as web.PublicKeyCredential;
    final response = cred.response as web.AuthenticatorAssertionResponse;

    return serializeAssertion(
      id: cred.id,
      rawId: _bufferToBytes(cred.rawId),
      type: cred.type,
      clientDataJson: _bufferToBytes(response.clientDataJSON),
      authenticatorData: _bufferToBytes(response.authenticatorData),
      signature: _bufferToBytes(response.signature),
      userHandle: response.userHandle == null
          ? null
          : _bufferToBytes(response.userHandle!),
      authenticatorAttachment: cred.authenticatorAttachment,
    );
  }
}

// ── JS interop helpers ─────────────────────────────────────────────

/// Recursively convert a Dart [Map] / [List] (with [Uint8List] binary
/// fields) into a JS-side plain object. Binary fields cross the boundary
/// as `Uint8Array` because that satisfies `BufferSource` parameters on
/// the WebAuthn API.
JSAny _toJSObject(Object? value) {
  if (value == null) return _nullJS();
  if (value is bool) return value.toJS;
  if (value is num) return value.toJS;
  if (value is String) return value.toJS;
  if (value is Uint8List) return value.toJS;
  if (value is List) {
    final arr = JSArray<JSAny?>();
    for (var i = 0; i < value.length; i++) {
      arr.setProperty(i.toJS, _toJSObject(value[i]));
    }
    return arr;
  }
  if (value is Map) {
    final obj = JSObject();
    value.forEach((k, v) {
      obj.setProperty(k.toString().toJS, _toJSObject(v));
    });
    return obj;
  }
  throw ArgumentError('Cannot convert ${value.runtimeType} to JS value');
}

JSAny _nullJS() => 'null'.toJS;

Uint8List _bufferToBytes(JSArrayBuffer buffer) {
  return buffer.toDart.asUint8List();
}
