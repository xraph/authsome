/// Pre-built Material Design 3 authentication UI for AuthSome.
///
/// Provides styled screens, headless builder widgets, and user management
/// components that integrate with [AuthProvider] from `authsome_flutter`.
///
/// Usage:
/// ```dart
/// import 'package:authsome_flutter_ui/authsome_flutter_ui.dart';
///
/// // Wrap your app with AuthProvider, then use pre-built screens:
/// SignInForm(
///   onSuccess: () => Navigator.pushReplacementNamed(context, '/home'),
///   onSignUpTap: () => Navigator.pushNamed(context, '/sign-up'),
/// )
/// ```
library authsome_flutter_ui;

// Re-export authsome_flutter for convenience.
export 'package:authsome_flutter/authsome_flutter.dart';

// Theme
export 'src/theme/auth_theme.dart';
export 'src/theme/social_icons.dart' show buildSocialIcon;

// Shared widgets
export 'src/widgets/auth_card.dart';
export 'src/widgets/error_display.dart';
export 'src/widgets/loading_indicator.dart';
export 'src/widgets/or_divider.dart';
export 'src/widgets/otp_input.dart';
export 'src/widgets/password_input.dart';
export 'src/widgets/social_buttons.dart';

// Headless builders
export 'src/builders/auth_guard.dart';
export 'src/builders/sign_in_form_builder.dart';
export 'src/builders/sign_up_form_builder.dart';
export 'src/builders/mfa_challenge_form_builder.dart';

// Styled auth form screens
export 'src/screens/sign_in_form.dart';
export 'src/screens/sign_up_form.dart';
export 'src/screens/forgot_password_form.dart';
export 'src/screens/reset_password_form.dart';
export 'src/screens/mfa_challenge_form.dart';
export 'src/screens/magic_link_form.dart';
export 'src/screens/change_password_form.dart';
export 'src/screens/email_verification_form.dart';

// User management
export 'src/user/user_avatar.dart';
export 'src/user/user_button.dart';
export 'src/user/user_profile_card.dart';
export 'src/user/org_switcher.dart';

// Session & device management
export 'src/management/session_list.dart';
export 'src/management/device_list.dart';
