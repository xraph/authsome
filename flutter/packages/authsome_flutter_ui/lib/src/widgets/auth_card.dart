/// Auth card layout wrapper.
///
/// A centered Material 3 card with title, description, logo, and content area.
library;

import 'package:flutter/material.dart';
import 'package:authsome_flutter/authsome_flutter.dart';

import '../theme/auth_theme.dart';

/// Horizontal alignment of the title + description text inside an [AuthCard].
///
/// Mirrors the React `align` prop on the shared web sign-in components: the
/// form fields themselves are always full-width, but the heading copy can sit
/// flush-left for a denser product feel or stay centered for the default
/// marketing-style layout.
enum AuthCardAlign {
  /// Title and description are centered (default).
  center,

  /// Title and description are flush-left.
  left,
}

/// Layout wrapper for auth forms — a centered Material 3 card with title,
/// description, optional logo, and footer.
class AuthCard extends StatelessWidget {
  /// Card title text.
  final String title;

  /// Optional description text below the title.
  final String? description;

  /// Optional logo widget above the title.
  ///
  /// When null, the card shows the app's branding logo
  /// (`clientConfig.branding.logoUrl`) if one is configured. Set
  /// [showBrandingLogo] to false to keep the card logo-free.
  final Widget? logo;

  /// Branding to fall back on when [logo] is null. When omitted, the card
  /// reads it from the surrounding [AuthProvider]'s client config.
  final BrandingConfig? branding;

  /// Whether to fall back to the branding logo when [logo] is null.
  final bool showBrandingLogo;

  /// Optional footer widget below the content.
  final Widget? footer;

  /// Card content.
  final Widget child;

  /// Maximum width (overrides [AuthThemeData.cardMaxWidth]).
  final double? maxWidth;

  /// Horizontal alignment of the title + description. Defaults to
  /// [AuthCardAlign.center]. Use [AuthCardAlign.left] to match a flush-left
  /// product layout.
  final AuthCardAlign align;

  const AuthCard({
    required this.title,
    this.description,
    this.logo,
    this.branding,
    this.showBrandingLogo = true,
    this.footer,
    required this.child,
    this.maxWidth,
    this.align = AuthCardAlign.center,
    super.key,
  });

  Widget? _resolveLogo(BuildContext context) {
    if (logo != null || !showBrandingLogo) return logo;
    final brand =
        branding ?? AuthProvider.maybeOf(context)?.clientConfig?.branding;
    final url = brand?.logoUrl;
    if (url == null || url.isEmpty) return null;
    return Image.network(
      url,
      height: 32,
      semanticLabel: brand?.appName,
      // A broken logo URL must not break the sign-in screen.
      errorBuilder: (_, __, ___) => const SizedBox.shrink(),
    );
  }

  @override
  Widget build(BuildContext context) {
    final theme = AuthTheme.of(context);
    final textTheme = Theme.of(context).textTheme;
    final colorScheme = Theme.of(context).colorScheme;

    final textAlign =
        align == AuthCardAlign.left ? TextAlign.start : TextAlign.center;
    // Logo centring is unchanged for left-aligned layouts — the React
    // equivalent keeps the brand mark in the top-left of the card, so a
    // left-aligned heading + a flush-left logo read as one visual block.
    final logoAlignment = align == AuthCardAlign.left
        ? Alignment.centerLeft
        : Alignment.center;
    final logo = _resolveLogo(context);

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
                  Align(alignment: logoAlignment, child: logo),
                  SizedBox(height: theme.fieldSpacing),
                ],
                Text(
                  title,
                  style: textTheme.headlineSmall?.copyWith(
                    fontWeight: FontWeight.w600,
                  ),
                  textAlign: textAlign,
                ),
                if (description != null) ...[
                  const SizedBox(height: 8),
                  Text(
                    description!,
                    style: textTheme.bodyMedium?.copyWith(
                      color: colorScheme.onSurfaceVariant,
                    ),
                    textAlign: textAlign,
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
