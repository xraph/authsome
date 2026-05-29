import 'package:authsome_core/authsome_core.dart';
import 'package:test/test.dart';

void main() {
  group('Session', () {
    test('roundtrips through fromJson / toJson', () {
      const json = {
        'session_token': 'sess_abc',
        'refresh_token': 'ref_xyz',
        'expires_at': '2026-06-01T00:00:00Z',
      };
      final session = Session.fromJson(json);
      expect(session.sessionToken, 'sess_abc');
      expect(session.refreshToken, 'ref_xyz');
      expect(session.expiresAt, '2026-06-01T00:00:00Z');
      expect(session.toJson(), json);
    });
  });
}
