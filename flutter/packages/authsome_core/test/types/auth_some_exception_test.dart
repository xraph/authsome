import 'package:authsome_core/authsome_core.dart';
import 'package:test/test.dart';

void main() {
  group('AuthClientException typed envelope', () {
    test('carries type and details fields', () {
      const exc = AuthClientException(
        'MFA required',
        code: 403,
        type: 'mfa_required',
        details: {
          'mfa_ticket': 'tk_abc',
          'available_methods': ['totp', 'sms'],
        },
      );

      expect(exc.code, 403);
      expect(exc.type, 'mfa_required');
      expect(exc.details, isNotNull);
    });

    test('isMfaRequired surfaces the typed category', () {
      const exc = AuthClientException('msg', type: 'mfa_required');
      expect(exc.isMfaRequired, isTrue);
      expect(exc.isEmailNotVerified, isFalse);
    });

    test('isEmailNotVerified detects the email-not-verified envelope', () {
      const exc = AuthClientException(
        'Please verify',
        code: 403,
        type: 'email_not_verified',
        details: {'email': 'user@example.com'},
      );
      expect(exc.isEmailNotVerified, isTrue);
      expect(exc.errorEmail, 'user@example.com');
    });

    test('mfaTicket and availableMfaMethods extract from details', () {
      const exc = AuthClientException(
        'MFA',
        code: 403,
        type: 'mfa_required',
        details: {
          'mfa_ticket': 'tk_xyz',
          'available_methods': ['totp', 'recovery'],
        },
      );
      expect(exc.mfaTicket, 'tk_xyz');
      expect(exc.availableMfaMethods, ['totp', 'recovery']);
    });

    test('all typed getters default to false / empty for an untyped exception',
        () {
      const exc = AuthClientException('Generic failure', code: 500);
      expect(exc.isMfaRequired, isFalse);
      expect(exc.isEmailNotVerified, isFalse);
      expect(exc.isCaptchaRequired, isFalse);
      expect(exc.mfaTicket, isNull);
      expect(exc.availableMfaMethods, isEmpty);
      expect(exc.errorEmail, isNull);
    });

    test('toString includes the type when present', () {
      const exc = AuthClientException('boom', code: 403, type: 'mfa_required');
      expect(exc.toString(), contains('mfa_required'));
      expect(exc.toString(), contains('boom'));
    });
  });
}
