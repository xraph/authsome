/// User button widget — avatar with dropdown menu for account actions.
///
/// Displays a [UserAvatar] that opens a [PopupMenuButton] with
/// profile, settings, custom menu items, and sign-out options.
///
/// ```dart
/// UserButton(
///   onProfileTap: () => Navigator.pushNamed(context, '/profile'),
///   onSettingsTap: () => Navigator.pushNamed(context, '/settings'),
///   menuItems: [
///     UserButtonMenuItem(
///       label: 'Billing',
///       icon: Icons.payment,
///       onTap: () => Navigator.pushNamed(context, '/billing'),
///     ),
///   ],
/// )
/// ```
library;

import 'package:flutter/material.dart';
import 'package:authsome_flutter/authsome_flutter.dart';

import 'user_avatar.dart';

/// A custom menu item for [UserButton].
class UserButtonMenuItem {
  /// Display label.
  final String label;

  /// Optional leading icon.
  final IconData? icon;

  /// Callback when the item is tapped.
  final VoidCallback onTap;

  /// Creates a [UserButtonMenuItem].
  const UserButtonMenuItem({
    required this.label,
    this.icon,
    required this.onTap,
  });
}

/// A user avatar button that opens a popup menu with account actions.
///
/// The menu shows the user's name and email as a header, followed by
/// optional profile and settings entries, any custom [menuItems],
/// a divider, and a sign-out action.
class UserButton extends StatelessWidget {
  /// Called when "Profile" is tapped. If null, the item is hidden.
  final VoidCallback? onProfileTap;

  /// Called when "Settings" is tapped. If null, the item is hidden.
  final VoidCallback? onSettingsTap;

  /// Called when "Sign out" is tapped.
  /// Defaults to `context.auth.signOut()` if not provided.
  final VoidCallback? onSignOut;

  /// Additional menu items inserted between built-in items and sign-out.
  final List<UserButtonMenuItem>? menuItems;

  /// Informational redirect URL after sign-out (not used for navigation
  /// in Flutter, but available for app-level routing decisions).
  final String? afterSignOutUrl;

  /// Creates a [UserButton].
  const UserButton({
    this.onProfileTap,
    this.onSettingsTap,
    this.onSignOut,
    this.menuItems,
    this.afterSignOutUrl,
    super.key,
  });

  /// Safely reads a string field from a dynamic user object.
  static String? _field(dynamic user, String key) {
    try {
      if (user is Map) {
        final value = user[key];
        return value is String ? value : null;
      }
      final dynamic value = switch (key) {
        'name' => user.name,
        'email' => user.email,
        _ => null,
      };
      return value is String ? value : null;
    } catch (_) {
      return null;
    }
  }

  @override
  Widget build(BuildContext context) {
    final auth = context.auth;
    final user = auth.user;
    final colorScheme = Theme.of(context).colorScheme;
    final textTheme = Theme.of(context).textTheme;

    final name = _field(user, 'name');
    final email = _field(user, 'email');

    return PopupMenuButton<String>(
      offset: const Offset(0, 48),
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
      itemBuilder: (context) {
        final items = <PopupMenuEntry<String>>[];

        // Header — user info (not selectable).
        items.add(
          PopupMenuItem<String>(
            enabled: false,
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              mainAxisSize: MainAxisSize.min,
              children: [
                if (name != null && name.isNotEmpty)
                  Text(
                    name,
                    style: textTheme.titleSmall?.copyWith(
                      fontWeight: FontWeight.w600,
                    ),
                  ),
                if (email != null && email.isNotEmpty)
                  Text(
                    email,
                    style: textTheme.bodySmall?.copyWith(
                      color: colorScheme.onSurfaceVariant,
                    ),
                  ),
              ],
            ),
          ),
        );

        items.add(const PopupMenuDivider());

        // Profile.
        if (onProfileTap != null) {
          items.add(
            const PopupMenuItem<String>(
              value: 'profile',
              child: _MenuRow(icon: Icons.person_outline, label: 'Profile'),
            ),
          );
        }

        // Settings.
        if (onSettingsTap != null) {
          items.add(
            const PopupMenuItem<String>(
              value: 'settings',
              child: _MenuRow(icon: Icons.settings_outlined, label: 'Settings'),
            ),
          );
        }

        // Custom items.
        if (menuItems != null) {
          for (var i = 0; i < menuItems!.length; i++) {
            final item = menuItems![i];
            items.add(
              PopupMenuItem<String>(
                value: 'custom_$i',
                child: _MenuRow(
                  icon: item.icon ?? Icons.circle_outlined,
                  label: item.label,
                ),
              ),
            );
          }
        }

        items.add(const PopupMenuDivider());

        // Sign out.
        items.add(
          PopupMenuItem<String>(
            value: 'signout',
            child: _MenuRow(
              icon: Icons.logout,
              label: 'Sign out',
              color: colorScheme.error,
            ),
          ),
        );

        return items;
      },
      onSelected: (value) {
        if (value == 'profile') {
          onProfileTap?.call();
        } else if (value == 'settings') {
          onSettingsTap?.call();
        } else if (value == 'signout') {
          if (onSignOut != null) {
            onSignOut!();
          } else {
            auth.signOut();
          }
        } else if (value.startsWith('custom_')) {
          final index = int.tryParse(value.replaceFirst('custom_', ''));
          if (index != null && menuItems != null && index < menuItems!.length) {
            menuItems![index].onTap();
          }
        }
      },
      child: const UserAvatar(size: UserAvatarSize.md),
    );
  }
}

/// Internal helper for rendering a menu row with icon and label.
class _MenuRow extends StatelessWidget {
  final IconData icon;
  final String label;
  final Color? color;

  const _MenuRow({required this.icon, required this.label, this.color});

  @override
  Widget build(BuildContext context) {
    final effectiveColor = color ?? Theme.of(context).colorScheme.onSurface;
    return Row(
      children: [
        Icon(icon, size: 20, color: effectiveColor),
        const SizedBox(width: 12),
        Text(label, style: TextStyle(color: effectiveColor)),
      ],
    );
  }
}
