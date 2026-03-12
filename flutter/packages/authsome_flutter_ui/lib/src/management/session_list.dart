/// Session list widget — displays and manages active user sessions.
///
/// Fetches the authenticated user's sessions and displays them in a
/// [ListView]. Each session shows device info, IP address, and relative
/// last-active time. The current session is badged and cannot be revoked.
/// Individual sessions can be revoked, and a footer action revokes all
/// other sessions at once.
///
/// ```dart
/// SessionList(
///   currentSessionToken: auth.session?.sessionToken,
///   onRevoke: (sessionId) => print('Revoked $sessionId'),
/// )
/// ```
library;

import 'package:flutter/material.dart';
import 'package:authsome_flutter/authsome_flutter.dart';

/// A list view of the user's active sessions with revoke actions.
///
/// On mount, fetches sessions via `auth.client.listSessions`. Each
/// session is rendered as a [ListTile] with a device icon, name/browser,
/// IP address, relative last-active time, and a revoke button.
///
/// The session matching [currentSessionToken] is badged as "Current"
/// and its revoke button is disabled.
class SessionList extends StatefulWidget {
  /// The token of the current session — used to identify and badge it.
  final String? currentSessionToken;

  /// Called after a session is successfully revoked, with the session ID.
  final ValueChanged<String>? onRevoke;

  /// Creates a [SessionList].
  const SessionList({
    this.currentSessionToken,
    this.onRevoke,
    super.key,
  });

  @override
  State<SessionList> createState() => _SessionListState();
}

class _SessionListState extends State<SessionList> {
  List<Map<String, dynamic>> _sessions = [];
  bool _loading = true;
  String? _error;
  final Set<String> _revokingIds = {};

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    _fetchSessions();
  }

  Future<void> _fetchSessions() async {
    final auth = context.auth;
    final token = auth.session?.sessionToken ?? '';

    setState(() {
      _loading = true;
      _error = null;
    });

    try {
      final response = await auth.client.listSessions(token: token);
      final sessions = response.sessions;

      if (mounted) {
        setState(() {
          _sessions = sessions;
          _loading = false;
        });
      }
    } catch (e) {
      if (mounted) {
        setState(() {
          _loading = false;
          _error = e.toString();
        });
      }
    }
  }

  /// Reads a field from a session map.
  String _sessionField(Map<String, dynamic> session, String key) {
    return session[key]?.toString() ?? '';
  }

  /// Returns an appropriate device icon based on the device type / name.
  IconData _deviceIcon(Map<String, dynamic> session) {
    final type = _sessionField(session, 'device_type').toLowerCase();
    final name = _sessionField(session, 'device_name').toLowerCase();
    final browser = _sessionField(session, 'browser').toLowerCase();

    if (type.contains('phone') ||
        type.contains('mobile') ||
        name.contains('phone') ||
        name.contains('android') ||
        name.contains('iphone')) {
      return Icons.phone_android;
    }
    if (type.contains('tablet') ||
        name.contains('tablet') ||
        name.contains('ipad')) {
      return Icons.tablet;
    }
    if (type.contains('desktop') ||
        name.contains('desktop') ||
        name.contains('mac') ||
        name.contains('windows') ||
        name.contains('linux')) {
      return Icons.computer;
    }
    if (browser.isNotEmpty) {
      return Icons.language;
    }
    return Icons.devices;
  }

  /// Checks whether a session is the current one.
  bool _isCurrent(Map<String, dynamic> session) {
    if (widget.currentSessionToken == null) return false;
    final sessionToken = _sessionField(session, 'session_token');
    final token = _sessionField(session, 'token');
    return sessionToken == widget.currentSessionToken ||
        token == widget.currentSessionToken;
  }

  Future<void> _revokeSession(Map<String, dynamic> session) async {
    final id = _sessionField(session, 'id');
    if (id.isEmpty) return;

    final confirmed = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Revoke Session'),
        content: const Text(
          'Are you sure you want to revoke this session? '
          'The device will be signed out immediately.',
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(ctx).pop(false),
            child: const Text('Cancel'),
          ),
          FilledButton(
            onPressed: () => Navigator.of(ctx).pop(true),
            style: FilledButton.styleFrom(
              backgroundColor: Theme.of(ctx).colorScheme.error,
            ),
            child: const Text('Revoke'),
          ),
        ],
      ),
    );

    if (confirmed != true || !mounted) return;

    setState(() => _revokingIds.add(id));

    try {
      final auth = context.auth;
      final token = auth.session?.sessionToken ?? '';
      await auth.client.revokeSession(
        sessionId: id,
        body: RevokeSessionRequest(sessionID: id),
        token: token,
      );

      if (mounted) {
        setState(() {
          _sessions.removeWhere((s) => _sessionField(s, 'id') == id);
          _revokingIds.remove(id);
        });
        widget.onRevoke?.call(id);
      }
    } catch (e) {
      if (mounted) {
        setState(() => _revokingIds.remove(id));
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: Text('Failed to revoke session: $e'),
            backgroundColor: Theme.of(context).colorScheme.error,
          ),
        );
      }
    }
  }

  Future<void> _revokeAllOther() async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Revoke All Other Sessions'),
        content: const Text(
          'This will sign out all other devices. '
          'Only your current session will remain active.',
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(ctx).pop(false),
            child: const Text('Cancel'),
          ),
          FilledButton(
            onPressed: () => Navigator.of(ctx).pop(true),
            style: FilledButton.styleFrom(
              backgroundColor: Theme.of(ctx).colorScheme.error,
            ),
            child: const Text('Revoke All'),
          ),
        ],
      ),
    );

    if (confirmed != true || !mounted) return;

    try {
      final auth = context.auth;
      final token = auth.session?.sessionToken ?? '';
      final otherSessions = _sessions.where((s) => !_isCurrent(s)).toList();
      for (final session in otherSessions) {
        final id = _sessionField(session, 'id');
        if (id.isNotEmpty) {
          await auth.client.revokeSession(
            sessionId: id,
            body: RevokeSessionRequest(sessionID: id),
            token: token,
          );
        }
      }

      if (mounted) {
        _fetchSessions();
      }
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: Text('Failed to revoke sessions: $e'),
            backgroundColor: Theme.of(context).colorScheme.error,
          ),
        );
      }
    }
  }

  /// Formats an ISO 8601 date string into a relative time label.
  String _formatRelativeTime(String isoDate) {
    if (isoDate.isEmpty) return '';
    final date = DateTime.tryParse(isoDate);
    if (date == null) return isoDate;

    final now = DateTime.now();
    final diff = now.difference(date);

    if (diff.isNegative) return 'just now';
    if (diff.inSeconds < 60) return 'just now';
    if (diff.inMinutes < 60) return '${diff.inMinutes}m ago';
    if (diff.inHours < 24) return '${diff.inHours}h ago';
    if (diff.inDays < 30) return '${diff.inDays}d ago';
    if (diff.inDays < 365) return '${(diff.inDays / 30).floor()}mo ago';
    return '${(diff.inDays / 365).floor()}y ago';
  }

  @override
  Widget build(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;
    final textTheme = Theme.of(context).textTheme;

    if (_loading) {
      return const Center(child: CircularProgressIndicator());
    }

    if (_error != null) {
      return Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(Icons.error_outline, color: colorScheme.error, size: 32),
            const SizedBox(height: 8),
            Text(
              'Failed to load sessions',
              style: textTheme.bodyMedium?.copyWith(color: colorScheme.error),
            ),
            const SizedBox(height: 8),
            TextButton(
              onPressed: _fetchSessions,
              child: const Text('Retry'),
            ),
          ],
        ),
      );
    }

    if (_sessions.isEmpty) {
      return Center(
        child: Text(
          'No active sessions',
          style: textTheme.bodyMedium?.copyWith(
            color: colorScheme.onSurfaceVariant,
          ),
        ),
      );
    }

    return Column(
      mainAxisSize: MainAxisSize.min,
      children: [
        ListView.separated(
          shrinkWrap: true,
          physics: const NeverScrollableScrollPhysics(),
          itemCount: _sessions.length,
          separatorBuilder: (_, __) => const Divider(height: 1),
          itemBuilder: (context, index) {
            final session = _sessions[index];
            final id = _sessionField(session, 'id');
            final deviceName = _sessionField(session, 'device_name');
            final browser = _sessionField(session, 'browser');
            final ipAddress = _sessionField(session, 'ip_address');
            final lastActive = _sessionField(session, 'last_active');
            final isCurrent = _isCurrent(session);
            final isRevoking = _revokingIds.contains(id);

            final title = deviceName.isNotEmpty ? deviceName : browser;

            return ListTile(
              leading: Icon(
                _deviceIcon(session),
                color: colorScheme.onSurfaceVariant,
              ),
              title: Row(
                children: [
                  Flexible(
                    child: Text(
                      title.isNotEmpty ? title : 'Unknown device',
                      overflow: TextOverflow.ellipsis,
                    ),
                  ),
                  if (isCurrent) ...[
                    const SizedBox(width: 8),
                    Chip(
                      label: Text(
                        'Current',
                        style: textTheme.labelSmall?.copyWith(
                          color: colorScheme.onPrimary,
                        ),
                      ),
                      backgroundColor: colorScheme.primary,
                      padding: EdgeInsets.zero,
                      materialTapTargetSize: MaterialTapTargetSize.shrinkWrap,
                      visualDensity: VisualDensity.compact,
                    ),
                  ],
                ],
              ),
              subtitle: Text(
                [
                  if (ipAddress.isNotEmpty) ipAddress,
                  if (lastActive.isNotEmpty) _formatRelativeTime(lastActive),
                ].join(' \u2022 '),
                style: textTheme.bodySmall?.copyWith(
                  color: colorScheme.onSurfaceVariant,
                ),
              ),
              trailing: isRevoking
                  ? const SizedBox(
                      width: 20,
                      height: 20,
                      child: CircularProgressIndicator(strokeWidth: 2),
                    )
                  : IconButton(
                      icon: Icon(
                        Icons.close,
                        color: isCurrent
                            ? colorScheme.onSurface.withValues(alpha: 0.3)
                            : colorScheme.error,
                      ),
                      tooltip: isCurrent ? 'Current session' : 'Revoke',
                      onPressed: isCurrent ? null : () => _revokeSession(session),
                    ),
            );
          },
        ),
        if (_sessions.length > 1) ...[
          const SizedBox(height: 8),
          Center(
            child: TextButton.icon(
              onPressed: _revokeAllOther,
              icon: const Icon(Icons.logout),
              label: const Text('Revoke all other sessions'),
              style: TextButton.styleFrom(
                foregroundColor: colorScheme.error,
              ),
            ),
          ),
        ],
      ],
    );
  }
}
