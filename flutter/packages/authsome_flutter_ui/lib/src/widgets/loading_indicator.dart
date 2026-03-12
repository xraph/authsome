/// Standardized loading indicator.
library;

import 'package:flutter/material.dart';

/// Size presets for the loading indicator.
enum LoadingSize {
  /// 16x16
  sm,
  /// 24x24
  md,
  /// 36x36
  lg,
}

/// A centered [CircularProgressIndicator] with configurable size.
class LoadingIndicator extends StatelessWidget {
  /// Size preset (default: [LoadingSize.md]).
  final LoadingSize size;

  const LoadingIndicator({this.size = LoadingSize.md, super.key});

  double get _dimension => switch (size) {
        LoadingSize.sm => 16,
        LoadingSize.md => 24,
        LoadingSize.lg => 36,
      };

  double get _strokeWidth => switch (size) {
        LoadingSize.sm => 2,
        LoadingSize.md => 3,
        LoadingSize.lg => 4,
      };

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      width: _dimension,
      height: _dimension,
      child: CircularProgressIndicator.adaptive(
        strokeWidth: _strokeWidth,
      ),
    );
  }
}
