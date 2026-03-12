/// Social login buttons with multiple layout options.
library;

import 'package:flutter/material.dart';

import '../theme/social_icons.dart';

/// Layout options for social login buttons.
enum SocialButtonLayout {
  /// 2-column grid (1-column if single provider).
  grid,

  /// Icon-only horizontal row with tooltips.
  iconRow,

  /// Full-width stacked buttons.
  vertical,
}

/// A social login provider definition.
class SocialProvider {
  /// Unique provider ID (e.g. "google", "github").
  final String id;

  /// Display name.
  final String name;

  /// Optional custom icon widget. If null, a default icon is used.
  final Widget? icon;

  const SocialProvider({required this.id, required this.name, this.icon});
}

/// Renders social login buttons in the specified layout.
class SocialButtons extends StatelessWidget {
  /// List of social providers to display.
  final List<SocialProvider> providers;

  /// Called when a provider button is tapped.
  final ValueChanged<String> onProviderClick;

  /// Whether the buttons should show a loading state.
  final bool isLoading;

  /// Layout mode (default: [SocialButtonLayout.grid]).
  final SocialButtonLayout layout;

  const SocialButtons({
    required this.providers,
    required this.onProviderClick,
    this.isLoading = false,
    this.layout = SocialButtonLayout.grid,
    super.key,
  });

  @override
  Widget build(BuildContext context) {
    if (providers.isEmpty) return const SizedBox.shrink();

    return switch (layout) {
      SocialButtonLayout.grid => _buildGrid(context),
      SocialButtonLayout.iconRow => _buildIconRow(context),
      SocialButtonLayout.vertical => _buildVertical(context),
    };
  }

  Widget _buildGrid(BuildContext context) {
    final crossAxisCount = providers.length == 1 ? 1 : 2;
    return Wrap(
      spacing: 8,
      runSpacing: 8,
      children: providers.map((p) {
        final width = crossAxisCount == 1
            ? double.infinity
            : (MediaQuery.of(context).size.width - 80) / 2;
        return SizedBox(
          width: providers.length == 1 ? double.infinity : width.clamp(0, 170),
          child: _SocialButton(
            provider: p,
            isLoading: isLoading,
            onTap: () => onProviderClick(p.id),
            showLabel: true,
          ),
        );
      }).toList(),
    );
  }

  Widget _buildIconRow(BuildContext context) {
    return Row(
      mainAxisAlignment: MainAxisAlignment.center,
      children: providers.map((p) {
        return Padding(
          padding: const EdgeInsets.symmetric(horizontal: 4),
          child: Tooltip(
            message: 'Continue with ${p.name}',
            child: _SocialIconButton(
              provider: p,
              isLoading: isLoading,
              onTap: () => onProviderClick(p.id),
            ),
          ),
        );
      }).toList(),
    );
  }

  Widget _buildVertical(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: providers.map((p) {
        return Padding(
          padding: const EdgeInsets.only(bottom: 8),
          child: _SocialButton(
            provider: p,
            isLoading: isLoading,
            onTap: () => onProviderClick(p.id),
            showLabel: true,
          ),
        );
      }).toList(),
    );
  }
}

class _SocialButton extends StatelessWidget {
  final SocialProvider provider;
  final bool isLoading;
  final VoidCallback onTap;
  final bool showLabel;

  const _SocialButton({
    required this.provider,
    required this.isLoading,
    required this.onTap,
    required this.showLabel,
  });

  @override
  Widget build(BuildContext context) {
    final icon = provider.icon ?? buildSocialIcon(provider.id) ?? const Icon(Icons.login, size: 20);

    return OutlinedButton(
      onPressed: isLoading ? null : onTap,
      style: OutlinedButton.styleFrom(
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(8)),
      ),
      child: Row(
        mainAxisAlignment: MainAxisAlignment.center,
        mainAxisSize: MainAxisSize.min,
        children: [
          SizedBox(width: 20, height: 20, child: icon),
          if (showLabel) ...[
            const SizedBox(width: 8),
            Text(provider.name),
          ],
        ],
      ),
    );
  }
}

class _SocialIconButton extends StatelessWidget {
  final SocialProvider provider;
  final bool isLoading;
  final VoidCallback onTap;

  const _SocialIconButton({
    required this.provider,
    required this.isLoading,
    required this.onTap,
  });

  @override
  Widget build(BuildContext context) {
    final icon = provider.icon ?? buildSocialIcon(provider.id) ?? const Icon(Icons.login, size: 20);

    return IconButton.outlined(
      onPressed: isLoading ? null : onTap,
      icon: SizedBox(width: 20, height: 20, child: icon),
    );
  }
}
