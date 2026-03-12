/// "Or" divider widget — a horizontal line with centered text.
library;

import 'package:flutter/material.dart';

/// A horizontal divider with centered text, typically "or".
class OrDivider extends StatelessWidget {
  /// The text to display (default: "or").
  final String text;

  const OrDivider({this.text = 'or', super.key});

  @override
  Widget build(BuildContext context) {
    final color = Theme.of(context).colorScheme.outlineVariant;
    return Row(
      children: [
        Expanded(child: Divider(color: color)),
        Padding(
          padding: const EdgeInsets.symmetric(horizontal: 16),
          child: Text(
            text,
            style: Theme.of(context).textTheme.bodySmall?.copyWith(
                  color: Theme.of(context).colorScheme.onSurfaceVariant,
                ),
          ),
        ),
        Expanded(child: Divider(color: color)),
      ],
    );
  }
}
