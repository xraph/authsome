/// Auth card layout wrapper.
///
/// A centered Material 3 card with title, description, logo, and content area.
library;

import 'package:flutter/material.dart';

import '../theme/auth_theme.dart';

/// Layout wrapper for auth forms — centered card with title, description,
/// optional logo, and footer.
class AuthCard extends StatelessWidget {
  /// Card title text.
  final String title;

  /// Optional description text below the title.
  final String? description;

  /// Optional logo widget above the title.
  final Widget? logo;

  /// Optional footer widget below the content.
  final Widget? footer;

  /// Card content.
  final Widget child;

  /// Maximum width (overrides [AuthThemeData.cardMaxWidth]).
  final double? maxWidth;

  const AuthCard({
    required this.title,
    this.description,
    this.logo,
    this.footer,
    required this.child,
    this.maxWidth,
    super.key,
  });

  @override
  Widget build(BuildContext context) {
    final theme = AuthTheme.of(context);
    final textTheme = Theme.of(context).textTheme;
    final colorScheme = Theme.of(context).colorScheme;

    return Center(
      child: ConstrainedBox(
        constraints: BoxConstraints(maxWidth: maxWidth ?? theme.cardMaxWidth),
        child: Card(
          elevation: 0,
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(theme.borderRadius),
            side: BorderSide(color: colorScheme.outlineVariant),
          ),
          child: Padding(
            padding: EdgeInsets.all(theme.cardPadding),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                if (logo != null) ...[
                  Center(child: logo!),
                  SizedBox(height: theme.fieldSpacing),
                ],
                Text(
                  title,
                  style: textTheme.headlineSmall?.copyWith(
                    fontWeight: FontWeight.w600,
                  ),
                  textAlign: TextAlign.center,
                ),
                if (description != null) ...[
                  const SizedBox(height: 8),
                  Text(
                    description!,
                    style: textTheme.bodyMedium?.copyWith(
                      color: colorScheme.onSurfaceVariant,
                    ),
                    textAlign: TextAlign.center,
                  ),
                ],
                SizedBox(height: theme.sectionSpacing),
                child,
                if (footer != null) ...[
                  SizedBox(height: theme.sectionSpacing),
                  footer!,
                ],
              ],
            ),
          ),
        ),
      ),
    );
  }
}
