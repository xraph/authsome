/// User avatar widget — displays user image or initials in a circle.
///
/// Shows the user's profile image as a [CircleAvatar] when available,
/// falling back to initials derived from the user's name or email.
/// Supports three sizes: small (32), medium (40), and large (48).
///
/// ```dart
/// // Uses current authenticated user from context
/// const UserAvatar(size: UserAvatarSize.lg)
///
/// // Uses a custom user object
/// UserAvatar(user: someUser, size: UserAvatarSize.sm)
/// ```
library;

import 'package:flutter/material.dart';
import 'package:authsome_flutter/authsome_flutter.dart';

/// Size presets for [UserAvatar].
enum UserAvatarSize {
  /// 32x32 avatar.
  sm,

  /// 40x40 avatar.
  md,

  /// 48x48 avatar.
  lg,
}

/// A circular avatar that displays the user's profile image or initials.
///
/// If [user] is provided it is used directly; otherwise the authenticated
/// user is read from [AuthProvider] via `context.auth.user`.
///
/// The avatar shows a [NetworkImage] when the user has an `image` field,
/// and falls back to uppercase initials rendered over
/// [ColorScheme.primaryContainer].
class UserAvatar extends StatelessWidget {
  /// Avatar size preset.
  final UserAvatarSize size;

  /// Optional user override. When null, `context.auth.user` is used.
  final dynamic user;

  /// Creates a [UserAvatar].
  const UserAvatar({
    this.size = UserAvatarSize.md,
    this.user,
    super.key,
  });

  double get _radius {
    return switch (size) {
      UserAvatarSize.sm => 16.0,
      UserAvatarSize.md => 20.0,
      UserAvatarSize.lg => 24.0,
    };
  }

  double get _fontSize {
    return switch (size) {
      UserAvatarSize.sm => 12.0,
      UserAvatarSize.md => 14.0,
      UserAvatarSize.lg => 16.0,
    };
  }

  /// Safely reads a string field from a dynamic user object.
  static String? _field(dynamic user, String key) {
    try {
      if (user is Map) {
        final value = user[key];
        return value is String ? value : null;
      }
      // Try dynamic property access for typed objects.
      final dynamic value = switch (key) {
        'image' => user.image,
        'name' => user.name,
        'email' => user.email,
        _ => null,
      };
      return value is String ? value : null;
    } catch (_) {
      return null;
    }
  }

  /// Derives up to two initials from a name or email string.
  static String _initials(String? name, String? email) {
    if (name != null && name.trim().isNotEmpty) {
      final parts = name.trim().split(RegExp(r'\s+'));
      if (parts.length >= 2) {
        return '${parts.first[0]}${parts[1][0]}'.toUpperCase();
      }
      return parts.first[0].toUpperCase();
    }
    if (email != null && email.isNotEmpty) {
      return email[0].toUpperCase();
    }
    return '?';
  }

  @override
  Widget build(BuildContext context) {
    final resolvedUser = user ?? context.auth.user;
    final colorScheme = Theme.of(context).colorScheme;

    final image = _field(resolvedUser, 'image');
    final name = _field(resolvedUser, 'name');
    final email = _field(resolvedUser, 'email');

    if (image != null && image.isNotEmpty) {
      return CircleAvatar(
        radius: _radius,
        backgroundImage: NetworkImage(image),
        backgroundColor: colorScheme.primaryContainer,
        onBackgroundImageError: (_, __) {
          // Silently fall back — the child initials will show.
        },
        child: Text(
          _initials(name, email),
          style: TextStyle(
            fontSize: _fontSize,
            fontWeight: FontWeight.w600,
            color: colorScheme.onPrimaryContainer,
          ),
        ),
      );
    }

    return CircleAvatar(
      radius: _radius,
      backgroundColor: colorScheme.primaryContainer,
      child: Text(
        _initials(name, email),
        style: TextStyle(
          fontSize: _fontSize,
          fontWeight: FontWeight.w600,
          color: colorScheme.onPrimaryContainer,
        ),
      ),
    );
  }
}
