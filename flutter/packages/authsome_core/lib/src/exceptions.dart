/// Typed convenience for [AuthClientException].
///
/// Mirrors React `AuthClientError` from `ui/packages/core/src/auth.ts`:
/// callers branch on the structured `type` field instead of string-matching
/// the human-readable message.
library;

import 'generated/api_client.dart';

/// Standard error-category strings emitted by the AuthSome backend.
/// Keep in sync with the TypeScript `AuthClientError` constants.
class AuthErrorType {
  static const String mfaRequired = 'mfa_required';
  static const String emailNotVerified = 'email_not_verified';
  static const String captchaRequired = 'captcha_required';
  static const String invalidCredentials = 'invalid_credentials';
  static const String accountLocked = 'account_locked';
  static const String passwordExpired = 'password_expired';
  static const String userSuspended = 'user_suspended';

  const AuthErrorType._();
}

extension AuthClientExceptionTyped on AuthClientException {
  bool get isMfaRequired => type == AuthErrorType.mfaRequired;
  bool get isEmailNotVerified => type == AuthErrorType.emailNotVerified;
  bool get isCaptchaRequired => type == AuthErrorType.captchaRequired;
  bool get isInvalidCredentials => type == AuthErrorType.invalidCredentials;
  bool get isAccountLocked => type == AuthErrorType.accountLocked;
  bool get isPasswordExpired => type == AuthErrorType.passwordExpired;
  bool get isUserSuspended => type == AuthErrorType.userSuspended;

  /// MFA ticket from the error envelope (when [isMfaRequired]).
  String? get mfaTicket => details?['mfa_ticket'] as String?;

  /// Available MFA methods from the error envelope (when [isMfaRequired]).
  List<String> get availableMfaMethods {
    final raw = details?['available_methods'];
    if (raw is List) return raw.cast<String>();
    return const [];
  }

  /// Email associated with the error envelope (when [isEmailNotVerified]).
  String? get errorEmail => details?['email'] as String?;
}
