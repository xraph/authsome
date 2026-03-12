/// Password input field with visibility toggle.
library;

import 'package:flutter/material.dart';

/// A [TextField] with a suffix button to toggle password visibility.
class PasswordInput extends StatefulWidget {
  /// Text editing controller.
  final TextEditingController? controller;

  /// Hint text shown when the field is empty.
  final String hintText;

  /// Label text above the field.
  final String? labelText;

  /// Whether the field is enabled (default: true).
  final bool enabled;

  /// Called when the text changes.
  final ValueChanged<String>? onChanged;

  /// Called when the user submits (e.g. presses Enter).
  final VoidCallback? onSubmitted;

  /// Focus node.
  final FocusNode? focusNode;

  /// Text input action (e.g. [TextInputAction.done]).
  final TextInputAction? textInputAction;

  const PasswordInput({
    this.controller,
    this.hintText = 'Password',
    this.labelText,
    this.enabled = true,
    this.onChanged,
    this.onSubmitted,
    this.focusNode,
    this.textInputAction,
    super.key,
  });

  @override
  State<PasswordInput> createState() => _PasswordInputState();
}

class _PasswordInputState extends State<PasswordInput> {
  bool _obscure = true;

  @override
  Widget build(BuildContext context) {
    return TextField(
      controller: widget.controller,
      obscureText: _obscure,
      enabled: widget.enabled,
      onChanged: widget.onChanged,
      focusNode: widget.focusNode,
      textInputAction: widget.textInputAction,
      onSubmitted: widget.onSubmitted != null ? (_) => widget.onSubmitted!() : null,
      decoration: InputDecoration(
        hintText: widget.hintText,
        labelText: widget.labelText,
        border: const OutlineInputBorder(),
        suffixIcon: IconButton(
          icon: Icon(_obscure ? Icons.visibility_off : Icons.visibility),
          onPressed: () => setState(() => _obscure = !_obscure),
        ),
      ),
    );
  }
}
