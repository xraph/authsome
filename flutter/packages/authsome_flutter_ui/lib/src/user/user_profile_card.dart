/// User profile card widget — displays and edits user information.
///
/// Shows the authenticated user's avatar, name, email, username, phone,
/// and account creation date in a Material 3 card. An edit button toggles
/// between read-only and edit mode, where the user can update their name
/// and username via the API.
///
/// ```dart
/// UserProfileCard(
///   onUpdate: () => ScaffoldMessenger.of(context).showSnackBar(
///     const SnackBar(content: Text('Profile updated')),
///   ),
/// )
/// ```
library;

import 'package:flutter/material.dart';
import 'package:authsome_flutter/authsome_flutter.dart';

import 'user_avatar.dart';

/// A card that displays the current user's profile and allows inline editing.
///
/// In read mode, the card shows the user's avatar (large), name, email
/// (with a verified badge if applicable), username, phone, and member-since
/// date. Tapping the edit button switches to edit mode with [TextField]s
/// for name and username, plus Save / Cancel buttons.
///
/// On save, calls `auth.client.updateMe` with the new values and invokes
/// [onUpdate] on success.
class UserProfileCard extends StatefulWidget {
  /// Called after a successful profile update.
  final VoidCallback? onUpdate;

  /// Creates a [UserProfileCard].
  const UserProfileCard({this.onUpdate, super.key});

  @override
  State<UserProfileCard> createState() => _UserProfileCardState();
}

class _UserProfileCardState extends State<UserProfileCard> {
  bool _editing = false;
  bool _saving = false;

  late TextEditingController _nameController;
  late TextEditingController _usernameController;

  @override
  void initState() {
    super.initState();
    _nameController = TextEditingController();
    _usernameController = TextEditingController();
  }

  @override
  void dispose() {
    _nameController.dispose();
    _usernameController.dispose();
    super.dispose();
  }

  /// Safely reads a field from the dynamic user object.
  String _field(dynamic user, String key) {
    try {
      if (user is Map) {
        final value = user[key];
        return value?.toString() ?? '';
      }
      final dynamic value = switch (key) {
        'name' => user.name,
        'email' => user.email,
        'username' => user.username,
        'phone' => user.phone,
        'image' => user.image,
        'id' => user.id,
        _ => null,
      };
      return value?.toString() ?? '';
    } catch (_) {
      return '';
    }
  }

  /// Safely reads a bool field from the dynamic user object.
  bool _boolField(dynamic user, String key) {
    try {
      if (user is Map) {
        final value = user[key];
        return value == true;
      }
      if (key == 'email_verified') {
        return user.email_verified == true;
      }
      return false;
    } catch (_) {
      return false;
    }
  }

  /// Safely reads a date string field from the dynamic user object.
  String _dateField(dynamic user, String key) {
    try {
      String? raw;
      if (user is Map) {
        raw = user[key]?.toString();
      } else if (key == 'created_at') {
        raw = user.created_at?.toString();
      }
      if (raw == null || raw.isEmpty) return '';
      final date = DateTime.tryParse(raw);
      if (date == null) return raw;
      return '${_monthName(date.month)} ${date.day}, ${date.year}';
    } catch (_) {
      return '';
    }
  }

  String _monthName(int month) {
    const months = [
      'Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun',
      'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec',
    ];
    return months[(month - 1).clamp(0, 11)];
  }

  void _startEditing(dynamic user) {
    setState(() {
      _editing = true;
      _nameController.text = _field(user, 'name');
      _usernameController.text = _field(user, 'username');
    });
  }

  void _cancelEditing() {
    setState(() {
      _editing = false;
    });
  }

  Future<void> _save() async {
    final auth = context.auth;
    final token = auth.session?.sessionToken ?? '';

    setState(() => _saving = true);

    try {
      await auth.client.updateMe(
        body: {
          'name': _nameController.text.trim(),
          'username': _usernameController.text.trim(),
        },
        token: token,
      );

      if (mounted) {
        setState(() {
          _editing = false;
          _saving = false;
        });
        widget.onUpdate?.call();
      }
    } catch (e) {
      if (mounted) {
        setState(() => _saving = false);
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: Text('Failed to update profile: $e'),
            backgroundColor: Theme.of(context).colorScheme.error,
          ),
        );
      }
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
    final username = _field(user, 'username');
    final phone = _field(user, 'phone');
    final emailVerified = _boolField(user, 'email_verified');
    final memberSince = _dateField(user, 'created_at');

    return Card(
      elevation: 0,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(12),
        side: BorderSide(color: colorScheme.outlineVariant),
      ),
      child: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          mainAxisSize: MainAxisSize.min,
          children: [
            // Header row: avatar + edit button.
            Row(
              children: [
                const UserAvatar(size: UserAvatarSize.lg),
                const Spacer(),
                if (!_editing)
                  IconButton(
                    icon: const Icon(Icons.edit_outlined),
                    tooltip: 'Edit profile',
                    onPressed: () => _startEditing(user),
                  ),
              ],
            ),

            const SizedBox(height: 16),

            if (_editing) ...[
              // Edit mode.
              TextField(
                controller: _nameController,
                decoration: InputDecoration(
                  labelText: 'Name',
                  border: OutlineInputBorder(
                    borderRadius: BorderRadius.circular(8),
                  ),
                ),
                enabled: !_saving,
              ),
              const SizedBox(height: 12),
              TextField(
                controller: _usernameController,
                decoration: InputDecoration(
                  labelText: 'Username',
                  border: OutlineInputBorder(
                    borderRadius: BorderRadius.circular(8),
                  ),
                ),
                enabled: !_saving,
              ),
              const SizedBox(height: 16),
              Row(
                mainAxisAlignment: MainAxisAlignment.end,
                children: [
                  TextButton(
                    onPressed: _saving ? null : _cancelEditing,
                    child: const Text('Cancel'),
                  ),
                  const SizedBox(width: 8),
                  FilledButton(
                    onPressed: _saving ? null : _save,
                    child: _saving
                        ? const SizedBox(
                            width: 16,
                            height: 16,
                            child: CircularProgressIndicator(
                              strokeWidth: 2,
                            ),
                          )
                        : const Text('Save'),
                  ),
                ],
              ),
            ] else ...[
              // Read mode.
              if (name.isNotEmpty)
                Text(
                  name,
                  style: textTheme.titleLarge?.copyWith(
                    fontWeight: FontWeight.w600,
                  ),
                ),

              if (email.isNotEmpty) ...[
                const SizedBox(height: 4),
                Row(
                  children: [
                    Text(
                      email,
                      style: textTheme.bodyMedium?.copyWith(
                        color: colorScheme.onSurfaceVariant,
                      ),
                    ),
                    if (emailVerified) ...[
                      const SizedBox(width: 4),
                      Icon(
                        Icons.verified,
                        size: 16,
                        color: colorScheme.primary,
                      ),
                    ],
                  ],
                ),
              ],

              const SizedBox(height: 16),
              const Divider(height: 1),
              const SizedBox(height: 16),

              _InfoRow(label: 'Username', value: username),
              _InfoRow(label: 'Phone', value: phone),
              _InfoRow(label: 'Member since', value: memberSince),
            ],
          ],
        ),
      ),
    );
  }
}

/// A label–value row used inside the profile card's read mode.
class _InfoRow extends StatelessWidget {
  final String label;
  final String value;

  const _InfoRow({required this.label, required this.value});

  @override
  Widget build(BuildContext context) {
    if (value.isEmpty) return const SizedBox.shrink();

    final textTheme = Theme.of(context).textTheme;
    final colorScheme = Theme.of(context).colorScheme;

    return Padding(
      padding: const EdgeInsets.only(bottom: 8),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          SizedBox(
            width: 120,
            child: Text(
              label,
              style: textTheme.bodySmall?.copyWith(
                color: colorScheme.onSurfaceVariant,
              ),
            ),
          ),
          Expanded(
            child: Text(
              value,
              style: textTheme.bodyMedium,
            ),
          ),
        ],
      ),
    );
  }
}
