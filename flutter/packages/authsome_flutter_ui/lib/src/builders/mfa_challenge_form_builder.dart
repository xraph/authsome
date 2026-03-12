/// Headless MFA challenge form builder widget.
///
/// Manages the MFA verification code input state internally and exposes
/// it via a builder callback. The consuming app provides its own UI
/// while this widget handles input state, submission via
/// [AuthNotifier.submitMFACode], error handling, and success detection.
///
/// ```dart
/// MfaChallengeFormBuilder(
///   enrollmentId: 'enr_abc123',
///   onSuccess: () => Navigator.of(context).pushReplacementNamed('/home'),
///   builder: (state) => Column(
///     children: [
///       TextField(
///         onChanged: state.setCode,
///         keyboardType: TextInputType.number,
///         decoration: const InputDecoration(labelText: 'Verification Code'),
///       ),
///       if (state.error != null) Text(state.error!),
///       ElevatedButton(
///         onPressed: state.isLoading ? null : state.submit,
///         child: state.isLoading
///             ? const CircularProgressIndicator()
///             : const Text('Verify'),
///       ),
///     ],
///   ),
/// )
/// ```
library;

import 'package:flutter/material.dart';
import 'package:authsome_flutter/authsome_flutter.dart';

/// Immutable snapshot of the MFA challenge form state, passed to the builder.
class MfaChallengeFormState {
  /// Current verification code value.
  final String code;

  /// Callback to update the code value.
  final ValueChanged<String> setCode;

  /// Submits the MFA verification code.
  final VoidCallback submit;

  /// Whether an MFA verification request is currently in progress.
  final bool isLoading;

  /// Error message from the last failed verification attempt, or null.
  final String? error;

  /// Creates a [MfaChallengeFormState].
  const MfaChallengeFormState({
    required this.code,
    required this.setCode,
    required this.submit,
    required this.isLoading,
    required this.error,
  });
}

/// Builder callback that receives the current [MfaChallengeFormState].
typedef MfaChallengeFormWidgetBuilder = Widget Function(
  MfaChallengeFormState state,
);

/// A headless MFA challenge form widget that manages code input state.
///
/// This widget creates and manages a [TextEditingController] for the
/// verification code field, handles submission via
/// [AuthNotifier.submitMFACode] using the provided [enrollmentId],
/// captures errors, and invokes [onSuccess] when the user successfully
/// completes the MFA challenge and becomes authenticated.
class MfaChallengeFormBuilder extends StatefulWidget {
  /// The MFA enrollment ID to verify against.
  final String enrollmentId;

  /// Builder that receives the current form state and returns a widget tree.
  final MfaChallengeFormWidgetBuilder builder;

  /// Called when MFA verification succeeds and the auth state becomes
  /// [AuthAuthenticated].
  final VoidCallback? onSuccess;

  /// Creates a [MfaChallengeFormBuilder].
  const MfaChallengeFormBuilder({
    required this.enrollmentId,
    required this.builder,
    this.onSuccess,
    super.key,
  });

  @override
  State<MfaChallengeFormBuilder> createState() =>
      _MfaChallengeFormBuilderState();
}

class _MfaChallengeFormBuilderState extends State<MfaChallengeFormBuilder> {
  final TextEditingController _codeController = TextEditingController();

  bool _isLoading = false;
  String? _error;
  bool _successHandled = false;

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    _checkAuthState();
  }

  void _checkAuthState() {
    final auth = context.auth;
    if (auth.state is AuthAuthenticated && !_successHandled) {
      _successHandled = true;
      if (_isLoading) {
        setState(() {
          _isLoading = false;
          _error = null;
        });
      }
      // Schedule the callback for after the current frame to avoid
      // calling setState during build.
      WidgetsBinding.instance.addPostFrameCallback((_) {
        widget.onSuccess?.call();
      });
    }
  }

  Future<void> _submit() async {
    final code = _codeController.text.trim();

    if (code.isEmpty) {
      setState(() {
        _error = 'Verification code is required.';
      });
      return;
    }

    setState(() {
      _isLoading = true;
      _error = null;
    });

    try {
      await context.auth.submitMFACode(widget.enrollmentId, code);
      // Success state will be detected in didChangeDependencies
      // when the auth notifier triggers a rebuild.
    } catch (e) {
      if (mounted) {
        setState(() {
          _isLoading = false;
          _error = e.toString();
        });
      }
    }
  }

  @override
  void dispose() {
    _codeController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    // Re-check auth state on every build (triggered by InheritedNotifier).
    _checkAuthState();

    return widget.builder(
      MfaChallengeFormState(
        code: _codeController.text,
        setCode: (value) {
          _codeController.text = value;
          setState(() {});
        },
        submit: _submit,
        isLoading: _isLoading,
        error: _error,
      ),
    );
  }
}
