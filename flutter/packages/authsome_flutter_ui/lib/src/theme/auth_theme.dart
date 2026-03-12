/// Theme configuration for AuthSome UI widgets.
///
/// Provides overridable spacing, border radius, and sizing defaults.
/// Falls back to Material 3 values from [Theme.of(context)].
library;

import 'package:flutter/material.dart';

/// Theme data for AuthSome UI widgets.
class AuthThemeData {
  /// Maximum width of auth cards (default: 400).
  final double cardMaxWidth;

  /// Default border radius for cards and inputs (default: 12).
  final double borderRadius;

  /// Default input border radius (default: 8).
  final double inputBorderRadius;

  /// Vertical spacing between form fields (default: 16).
  final double fieldSpacing;

  /// Vertical spacing between sections (default: 24).
  final double sectionSpacing;

  /// Horizontal padding inside cards (default: 24).
  final double cardPadding;

  const AuthThemeData({
    this.cardMaxWidth = 400,
    this.borderRadius = 12,
    this.inputBorderRadius = 8,
    this.fieldSpacing = 16,
    this.sectionSpacing = 24,
    this.cardPadding = 24,
  });
}

/// Provides [AuthThemeData] to descendant widgets.
///
/// Wrap your app or a subtree with [AuthTheme] to customize
/// AuthSome widget appearance:
///
/// ```dart
/// AuthTheme(
///   data: AuthThemeData(cardMaxWidth: 360, borderRadius: 16),
///   child: SignInForm(),
/// )
/// ```
class AuthTheme extends InheritedWidget {
  /// The theme data to provide to descendant widgets.
  final AuthThemeData data;

  const AuthTheme({
    required this.data,
    required super.child,
    super.key,
  });

  /// Get the [AuthThemeData] from the nearest ancestor, or defaults.
  static AuthThemeData of(BuildContext context) {
    final widget = context.dependOnInheritedWidgetOfExactType<AuthTheme>();
    return widget?.data ?? const AuthThemeData();
  }

  @override
  bool updateShouldNotify(AuthTheme oldWidget) => data != oldWidget.data;
}
