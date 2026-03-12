/// OTP (one-time password) input widget.
///
/// Displays a row of digit boxes with a hidden TextField for keyboard input.
/// Auto-submits when all digits are entered.
library;

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

/// A 6-digit OTP input that auto-submits on completion.
class OtpInput extends StatefulWidget {
  /// Called when all digits have been entered.
  final ValueChanged<String> onCompleted;

  /// Called on each character change.
  final ValueChanged<String>? onChanged;

  /// Number of digits (default: 6).
  final int length;

  /// Whether the input is enabled (default: true).
  final bool enabled;

  /// Focus node.
  final FocusNode? focusNode;

  const OtpInput({
    required this.onCompleted,
    this.onChanged,
    this.length = 6,
    this.enabled = true,
    this.focusNode,
    super.key,
  });

  @override
  State<OtpInput> createState() => _OtpInputState();
}

class _OtpInputState extends State<OtpInput> {
  late final TextEditingController _controller;
  late final FocusNode _focusNode;
  String _value = '';

  @override
  void initState() {
    super.initState();
    _controller = TextEditingController();
    _focusNode = widget.focusNode ?? FocusNode();
  }

  @override
  void dispose() {
    _controller.dispose();
    if (widget.focusNode == null) _focusNode.dispose();
    super.dispose();
  }

  void _onChanged(String value) {
    // Filter to digits only.
    final digits = value.replaceAll(RegExp(r'[^0-9]'), '');
    if (digits.length > widget.length) return;

    setState(() => _value = digits);
    widget.onChanged?.call(digits);

    if (digits.length == widget.length) {
      widget.onCompleted(digits);
    }
  }

  @override
  Widget build(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;

    return GestureDetector(
      onTap: () => _focusNode.requestFocus(),
      child: Stack(
        alignment: Alignment.center,
        children: [
          // Hidden text field to receive keyboard input.
          Opacity(
            opacity: 0,
            child: SizedBox(
              height: 1,
              child: TextField(
                controller: _controller,
                focusNode: _focusNode,
                enabled: widget.enabled,
                keyboardType: TextInputType.number,
                inputFormatters: [
                  FilteringTextInputFormatter.digitsOnly,
                  LengthLimitingTextInputFormatter(widget.length),
                ],
                onChanged: _onChanged,
                decoration: const InputDecoration(border: InputBorder.none),
              ),
            ),
          ),
          // Visible digit boxes.
          Row(
            mainAxisAlignment: MainAxisAlignment.center,
            children: List.generate(widget.length, (index) {
              final hasDigit = index < _value.length;
              final isFocused = _focusNode.hasFocus && index == _value.length;

              return Container(
                width: 44,
                height: 52,
                margin: EdgeInsets.only(right: index < widget.length - 1 ? 8 : 0),
                decoration: BoxDecoration(
                  border: Border.all(
                    color: isFocused
                        ? colorScheme.primary
                        : colorScheme.outline,
                    width: isFocused ? 2 : 1,
                  ),
                  borderRadius: BorderRadius.circular(8),
                  color: widget.enabled
                      ? colorScheme.surface
                      : colorScheme.surfaceContainerHighest,
                ),
                alignment: Alignment.center,
                child: Text(
                  hasDigit ? _value[index] : '',
                  style: Theme.of(context).textTheme.headlineSmall?.copyWith(
                        color: colorScheme.onSurface,
                      ),
                ),
              );
            }),
          ),
        ],
      ),
    );
  }
}
