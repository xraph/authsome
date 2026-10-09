/// One custom signup field, rendered from its [SignupFieldConfig].
library;

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:authsome_flutter/authsome_flutter.dart';

/// Checks [value] against [field]'s validation rules and returns the first
/// problem, or null when the value is acceptable.
///
/// React leans on the browser's form validation (`required`, `minLength`,
/// `pattern`, ...) for these rules. Flutter has no such layer, so the
/// sign-up form calls this before submitting.
String? validateSignupField(SignupFieldConfig field, String value) {
  final rules = field.validation;
  final trimmed = value.trim();
  final isToggle = field.type == 'checkbox' || field.type == 'switch';
  if (rules?.required == true &&
      (trimmed.isEmpty || (isToggle && trimmed != 'true'))) {
    return '${field.label} is required';
  }
  if (rules == null || trimmed.isEmpty) return null;
  if (rules.minLen != null && trimmed.length < rules.minLen!) {
    return '${field.label} must be at least ${rules.minLen} characters';
  }
  if (rules.maxLen != null && trimmed.length > rules.maxLen!) {
    return '${field.label} must be at most ${rules.maxLen} characters';
  }
  if (rules.min != null || rules.max != null) {
    final number = num.tryParse(trimmed);
    if (number == null) return '${field.label} must be a number';
    if (rules.min != null && number < rules.min!) {
      return '${field.label} must be at least ${rules.min}';
    }
    if (rules.max != null && number > rules.max!) {
      return '${field.label} must be at most ${rules.max}';
    }
  }
  final pattern = rules.pattern;
  if (pattern != null && pattern.isNotEmpty) {
    // HTML's pattern attribute matches the whole value; anchor to match.
    RegExp? re;
    try {
      re = RegExp('^(?:$pattern)\$');
    } catch (_) {
      re = null; // A pattern Dart can't parse is skipped, not enforced.
    }
    if (re != null && !re.hasMatch(trimmed)) {
      return '${field.label} is not in the expected format';
    }
  }
  return null;
}

/// Renders a single [SignupFieldConfig] the way React's `DynamicField`
/// does: text-like types become a [TextFormField], `textarea` a multi-line
/// one, `select` and `radio` a dropdown, `checkbox` and `switch` a toggle
/// whose value is the string "true" or "false".
class SignupFieldInput extends StatelessWidget {
  final SignupFieldConfig field;
  final String value;
  final ValueChanged<String> onChanged;
  final bool enabled;

  /// Validation message to show under the field.
  final String? errorText;

  const SignupFieldInput({
    required this.field,
    required this.value,
    required this.onChanged,
    this.enabled = true,
    this.errorText,
    super.key,
  });

  String get _label {
    final required = field.validation?.required ?? false;
    return required ? field.label : '${field.label} (optional)';
  }

  @override
  Widget build(BuildContext context) {
    switch (field.type) {
      case 'checkbox':
      case 'switch':
        final checked = value == 'true';
        final subtitle =
            field.description != null ? Text(field.description!) : null;
        final tile = field.type == 'switch'
            ? SwitchListTile(
                value: checked,
                onChanged: enabled ? (v) => onChanged(v.toString()) : null,
                title: Text(field.label),
                subtitle: subtitle,
                contentPadding: EdgeInsets.zero,
              )
            : CheckboxListTile(
                value: checked,
                onChanged:
                    enabled ? (v) => onChanged((v ?? false).toString()) : null,
                title: Text(field.label),
                subtitle: subtitle,
                controlAffinity: ListTileControlAffinity.leading,
                contentPadding: EdgeInsets.zero,
              );
        if (errorText == null) return tile;
        return Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          mainAxisSize: MainAxisSize.min,
          children: [
            tile,
            Text(
              errorText!,
              style: TextStyle(
                color: Theme.of(context).colorScheme.error,
                fontSize: 12,
              ),
            ),
          ],
        );

      case 'select':
      case 'radio':
        final options = field.options ?? const <SignupFieldOption>[];
        final selected = options.any((o) => o.value == value) ? value : null;
        return DropdownButtonFormField<String>(
          // The key carries the value so an outside change (a config
          // default arriving late) re-seeds the field.
          key: ValueKey('signup-field-${field.key}-$selected'),
          initialValue: selected,
          isExpanded: true,
          items: [
            for (final o in options)
              DropdownMenuItem(value: o.value, child: Text(o.label)),
          ],
          onChanged: enabled ? (v) => onChanged(v ?? '') : null,
          decoration: InputDecoration(
            labelText: _label,
            hintText: field.placeholder ??
                'Select ${field.label.toLowerCase()}',
            helperText: field.description,
            errorText: errorText,
            border: const OutlineInputBorder(),
          ),
        );

      default:
        final isTextarea = field.type == 'textarea';
        return TextFormField(
          key: ValueKey('signup-field-${field.key}'),
          initialValue: value,
          enabled: enabled,
          onChanged: onChanged,
          keyboardType: _keyboardFor(field.type),
          minLines: isTextarea ? 3 : 1,
          maxLines: isTextarea ? 6 : 1,
          inputFormatters: [
            if (field.validation?.maxLen != null)
              LengthLimitingTextInputFormatter(field.validation!.maxLen),
          ],
          textInputAction:
              isTextarea ? TextInputAction.newline : TextInputAction.next,
          decoration: InputDecoration(
            labelText: _label,
            hintText: field.placeholder,
            helperText: field.description,
            errorText: errorText,
            border: const OutlineInputBorder(),
          ),
        );
    }
  }

  static TextInputType _keyboardFor(String type) {
    switch (type) {
      case 'email':
        return TextInputType.emailAddress;
      case 'number':
        return const TextInputType.numberWithOptions(decimal: true);
      case 'tel':
        return TextInputType.phone;
      case 'url':
        return TextInputType.url;
      case 'date':
        return TextInputType.datetime;
      case 'textarea':
        return TextInputType.multiline;
      default:
        return TextInputType.text;
    }
  }
}
