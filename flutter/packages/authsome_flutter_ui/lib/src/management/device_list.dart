/// Device list widget — displays and manages trusted/untrusted devices.
///
/// Fetches the authenticated user's devices and renders them in a
/// [ListView]. Each device shows its type icon, name, browser, OS,
/// IP, last-seen time, and a trust status badge. Provides actions
/// to trust/untrust and delete individual devices.
///
/// ```dart
/// DeviceList(
///   onTrust: (deviceId) => print('Trusted $deviceId'),
///   onDelete: (deviceId) => print('Deleted $deviceId'),
/// )
/// ```
library;

import 'package:flutter/material.dart';
import 'package:authsome_flutter/authsome_flutter.dart';

/// A list view of the user's registered devices with trust and delete actions.
///
/// On mount, fetches devices via `auth.client.listDevices`. Each device
/// is rendered as a [ListTile] with a type icon, name, subtitle details,
/// a trust/untrust [Chip], and trailing action buttons for toggling trust
/// status and deleting the device.
class DeviceList extends StatefulWidget {
  /// Called after a device is trusted, with the device ID.
  final ValueChanged<String>? onTrust;

  /// Called after a device is deleted, with the device ID.
  final ValueChanged<String>? onDelete;

  /// Creates a [DeviceList].
  const DeviceList({
    this.onTrust,
    this.onDelete,
    super.key,
  });

  @override
  State<DeviceList> createState() => _DeviceListState();
}

class _DeviceListState extends State<DeviceList> {
  List<Device> _devices = [];
  bool _loading = true;
  String? _error;
  final Set<String> _actionIds = {};

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    _fetchDevices();
  }

  Future<void> _fetchDevices() async {
    final auth = context.auth;
    final token = auth.session?.sessionToken ?? '';

    setState(() {
      _loading = true;
      _error = null;
    });

    try {
      final response = await auth.client.listDevices(token: token);
      final devices = response.devices;

      if (mounted) {
        setState(() {
          _devices = devices;
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

  /// Returns an appropriate icon for the device type.
  IconData _deviceIcon(Device device) {
    final deviceType = (device.type ?? '').toLowerCase();
    final name = (device.name ?? '').toLowerCase();

    final combined = '$deviceType $name';

    if (combined.contains('phone') ||
        combined.contains('mobile') ||
        combined.contains('android') ||
        combined.contains('iphone')) {
      return Icons.phone_android;
    }
    if (combined.contains('tablet') || combined.contains('ipad')) {
      return Icons.tablet;
    }
    if (combined.contains('desktop') ||
        combined.contains('mac') ||
        combined.contains('pc') ||
        combined.contains('windows') ||
        combined.contains('linux')) {
      return Icons.computer;
    }
    if (combined.contains('tv') || combined.contains('television')) {
      return Icons.tv;
    }
    if (combined.contains('watch') || combined.contains('wearable')) {
      return Icons.watch;
    }
    return Icons.devices;
  }

  Future<void> _toggleTrust(Device device) async {
    final id = device.id;

    setState(() => _actionIds.add(id));

    try {
      final auth = context.auth;
      final token = auth.session?.sessionToken ?? '';

      await auth.client.trustDevice(
        deviceId: id,
        body: TrustDeviceRequest(deviceID: id),
        token: token,
      );

      if (mounted) {
        setState(() => _actionIds.remove(id));
        widget.onTrust?.call(id);
        _fetchDevices();
      }
    } catch (e) {
      if (mounted) {
        setState(() => _actionIds.remove(id));
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: Text(
              'Failed to update device trust: $e',
            ),
            backgroundColor: Theme.of(context).colorScheme.error,
          ),
        );
      }
    }
  }

  Future<void> _deleteDevice(Device device) async {
    final id = device.id;
    final name = device.name ?? '';

    final confirmed = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Delete Device'),
        content: Text(
          'Are you sure you want to delete '
          '"${name.isNotEmpty ? name : 'this device'}"? '
          'This action cannot be undone.',
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
            child: const Text('Delete'),
          ),
        ],
      ),
    );

    if (confirmed != true || !mounted) return;

    setState(() => _actionIds.add(id));

    try {
      final auth = context.auth;
      final token = auth.session?.sessionToken ?? '';
      await auth.client.deleteDevice(
        deviceId: id,
        body: DeleteDeviceRequest(deviceID: id),
        token: token,
      );

      if (mounted) {
        setState(() {
          _devices.removeWhere((d) => d.id == id);
          _actionIds.remove(id);
        });
        widget.onDelete?.call(id);
      }
    } catch (e) {
      if (mounted) {
        setState(() => _actionIds.remove(id));
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: Text('Failed to delete device: $e'),
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
              'Failed to load devices',
              style: textTheme.bodyMedium?.copyWith(color: colorScheme.error),
            ),
            const SizedBox(height: 8),
            TextButton(
              onPressed: _fetchDevices,
              child: const Text('Retry'),
            ),
          ],
        ),
      );
    }

    if (_devices.isEmpty) {
      return Center(
        child: Text(
          'No devices found',
          style: textTheme.bodyMedium?.copyWith(
            color: colorScheme.onSurfaceVariant,
          ),
        ),
      );
    }

    return ListView.separated(
      shrinkWrap: true,
      physics: const NeverScrollableScrollPhysics(),
      itemCount: _devices.length,
      separatorBuilder: (_, __) => const Divider(height: 1),
      itemBuilder: (context, index) {
        final device = _devices[index];
        final id = device.id;
        final name = device.name ?? '';
        final browser = device.browser ?? '';
        final os = device.os ?? '';
        final ip = device.ipAddress ?? '';
        final lastSeen = device.lastSeenAt;
        final trusted = device.trusted;
        final isActing = _actionIds.contains(id);

        // Build subtitle parts.
        final subtitleParts = <String>[
          if (browser.isNotEmpty) browser,
          if (os.isNotEmpty) os,
          if (ip.isNotEmpty) ip,
          if (lastSeen.isNotEmpty) _formatRelativeTime(lastSeen),
        ];

        return ListTile(
          leading: Icon(
            _deviceIcon(device),
            color: colorScheme.onSurfaceVariant,
          ),
          title: Row(
            children: [
              Flexible(
                child: Text(
                  name.isNotEmpty ? name : 'Unknown device',
                  overflow: TextOverflow.ellipsis,
                ),
              ),
              const SizedBox(width: 8),
              _TrustBadge(trusted: trusted),
            ],
          ),
          subtitle: subtitleParts.isNotEmpty
              ? Text(
                  subtitleParts.join(' \u2022 '),
                  style: textTheme.bodySmall?.copyWith(
                    color: colorScheme.onSurfaceVariant,
                  ),
                  maxLines: 2,
                  overflow: TextOverflow.ellipsis,
                )
              : null,
          trailing: isActing
              ? const SizedBox(
                  width: 20,
                  height: 20,
                  child: CircularProgressIndicator(strokeWidth: 2),
                )
              : Row(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    IconButton(
                      icon: Icon(
                        trusted
                            ? Icons.verified_user
                            : Icons.gpp_maybe_outlined,
                        color: trusted
                            ? colorScheme.primary
                            : colorScheme.onSurfaceVariant,
                      ),
                      tooltip: trusted ? 'Untrust device' : 'Trust device',
                      onPressed: () => _toggleTrust(device),
                    ),
                    IconButton(
                      icon: Icon(
                        Icons.delete_outline,
                        color: colorScheme.error,
                      ),
                      tooltip: 'Delete device',
                      onPressed: () => _deleteDevice(device),
                    ),
                  ],
                ),
        );
      },
    );
  }
}

/// A small chip badge showing trust status.
class _TrustBadge extends StatelessWidget {
  final bool trusted;

  const _TrustBadge({required this.trusted});

  @override
  Widget build(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;
    final textTheme = Theme.of(context).textTheme;

    if (trusted) {
      return Chip(
        label: Text(
          'Trusted',
          style: textTheme.labelSmall?.copyWith(
            color: colorScheme.onPrimary,
          ),
        ),
        backgroundColor: colorScheme.primary,
        padding: EdgeInsets.zero,
        materialTapTargetSize: MaterialTapTargetSize.shrinkWrap,
        visualDensity: VisualDensity.compact,
      );
    }

    return Chip(
      label: Text(
        'Untrusted',
        style: textTheme.labelSmall?.copyWith(
          color: colorScheme.onSurfaceVariant,
        ),
      ),
      backgroundColor: Colors.transparent,
      side: BorderSide(color: colorScheme.outline),
      padding: EdgeInsets.zero,
      materialTapTargetSize: MaterialTapTargetSize.shrinkWrap,
      visualDensity: VisualDensity.compact,
    );
  }
}
