/// Single sign-on buttons, one per configured SSO connection.
library;

import 'package:flutter/material.dart';
import 'package:authsome_flutter/authsome_flutter.dart';

import '../theme/auth_theme.dart';

/// A stacked list of "Continue with {name}" buttons for SSO connections.
class SSOButtons extends StatelessWidget {
  /// The connections to offer, usually `clientConfig.sso.connections`.
  final List<SSOConnectionConfig> connections;

  /// Called with the connection id when a button is tapped.
  final ValueChanged<String> onConnectionTap;

  /// Disables every button while a request is in flight.
  final bool isLoading;

  const SSOButtons({
    required this.connections,
    required this.onConnectionTap,
    this.isLoading = false,
    super.key,
  });

  @override
  Widget build(BuildContext context) {
    if (connections.isEmpty) return const SizedBox.shrink();
    final theme = AuthTheme.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      mainAxisSize: MainAxisSize.min,
      children: [
        for (var i = 0; i < connections.length; i++) ...[
          if (i > 0) SizedBox(height: theme.fieldSpacing / 2),
          OutlinedButton.icon(
            key: ValueKey('sso-${connections[i].id}'),
            onPressed:
                isLoading ? null : () => onConnectionTap(connections[i].id),
            icon: const Icon(Icons.vpn_key_outlined, size: 18),
            label: Text('Continue with ${connections[i].name}'),
          ),
        ],
      ],
    );
  }
}
