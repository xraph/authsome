import 'dart:typed_data';

import 'package:authsome_core/src/webauthn.dart';
import 'package:test/test.dart';

void main() {
  group('webauthn codecs', () {
    test('base64urlToBytes / bytesToBase64url round-trip', () {
      final original = Uint8List.fromList([1, 2, 3, 250, 251, 252, 253, 254, 255]);
      final encoded = bytesToBase64url(original);
      expect(encoded.contains('+'), isFalse);
      expect(encoded.contains('/'), isFalse);
      expect(encoded.contains('='), isFalse);
      final decoded = base64urlToBytes(encoded);
      expect(decoded, original);
    });

    test('bytesToBase64url handles empty input', () {
      expect(bytesToBase64url(Uint8List(0)), '');
    });

    test('base64urlToBytes tolerates input that needs re-padding', () {
      // "Hello" -> base64 "SGVsbG8=" -> base64url "SGVsbG8"
      final bytes = base64urlToBytes('SGVsbG8');
      expect(String.fromCharCodes(bytes), 'Hello');
    });
  });

  group('prepareRequestOptions', () {
    test('decodes challenge and allowCredentials[].id from base64url', () {
      final out = prepareRequestOptions({
        'challenge': 'AQID',
        'allowCredentials': [
          {'id': 'AQID', 'type': 'public-key'},
        ],
      });
      expect(out['challenge'], isA<Uint8List>());
      expect(out['challenge'], Uint8List.fromList([1, 2, 3]));
      final allow = out['allowCredentials'] as List;
      expect(allow.first['id'], Uint8List.fromList([1, 2, 3]));
      expect(allow.first['type'], 'public-key');
    });

    test('unwraps a server response nested under `publicKey`', () {
      final out = prepareRequestOptions({
        'publicKey': {'challenge': 'AQID'},
      });
      expect(out['challenge'], Uint8List.fromList([1, 2, 3]));
    });
  });

  group('serializeAssertion', () {
    test('encodes Uint8List fields as base64url strings', () {
      final result = serializeAssertion(
        id: 'cred-id',
        rawId: Uint8List.fromList([1, 2, 3]),
        type: 'public-key',
        clientDataJson: Uint8List.fromList([4, 5, 6]),
        authenticatorData: Uint8List.fromList([7, 8, 9]),
        signature: Uint8List.fromList([10, 11, 12]),
      );
      expect(result['id'], 'cred-id');
      expect(result['type'], 'public-key');
      expect(result['rawId'], 'AQID');
      final response = result['response'] as Map<String, dynamic>;
      expect(response['clientDataJSON'], 'BAUG');
      expect(response['authenticatorData'], 'BwgJ');
      expect(response['signature'], 'CgsM');
      expect(response.containsKey('userHandle'), isFalse);
    });

    test('passes through pre-encoded base64url strings unchanged', () {
      final result = serializeAssertion(
        id: 'x',
        rawId: 'AAEC',
        type: 'public-key',
        clientDataJson: 'AAEC',
        authenticatorData: 'AAEC',
        signature: 'AAEC',
        userHandle: 'AAEC',
        authenticatorAttachment: 'platform',
      );
      expect(result['rawId'], 'AAEC');
      final response = result['response'] as Map<String, dynamic>;
      expect(response['userHandle'], 'AAEC');
      expect(result['authenticatorAttachment'], 'platform');
    });
  });
}
