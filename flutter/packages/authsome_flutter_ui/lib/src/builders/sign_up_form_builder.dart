/// Headless sign-up form builder widget.
///
/// Manages email, password, and name form state internally and exposes it
/// via a builder callback. The consuming app provides its own UI while
/// this widget handles input state, submission, error handling, and
/// success detection.
///
/// ```dart
/// SignUpFormBuilder(
///   onSuccess: () => Navigator.of(context).pushReplacementNamed('/home'),
///   builder: (state) => Column(
///     children: [
///       TextField(
///         onChanged: state.setName,
///         decoration: const InputDecoration(labelText: 'Name'),
///       ),
///       TextField(
///         onChanged: state.setEmail,
///         decoration: const InputDecoration(labelText: 'Email'),
///       ),
///       TextField(
///         onChanged: state.setPassword,
///         obscureText: true,
///         decoration: const InputDecoration(labelText: 'Password'),
///       ),
///       if (state.error != null) Text(state.error!),
///       ElevatedButton(
///         onPressed: state.isLoading ? null : state.submit,
///         child: state.isLoading
///             ? const CircularProgressIndicator()
///             : const Text('Sign Up'),
///       ),
///     ],
///   ),
/// )
/// ```
library;

import 'package:flutter/material.dart';
import 'package:authsome_flutter/authsome_flutter.dart';

/// Immutable snapshot of the sign-up form state, passed to the builder.
class SignUpFormState {
  /// Current name value.
  final String name;

  /// Current email value.
  final String email;

  /// Current password value.
  final String password;

  /// Callback to update the name value.
  final ValueChanged<String> setName;

  /// Callback to update the email value.
  final ValueChanged<String> setEmail;

  /// Callback to update the password value.
  final ValueChanged<String> setPassword;

  /// Submits the sign-up form.
  final VoidCallback submit;

  /// Whether a sign-up request is currently in progress.
  final bool isLoading;

  /// Error message from the last failed sign-up attempt, or null.
  final String? error;

  /// Creates a [SignUpFormState].
  const SignUpFormState({
    required this.name,
    required this.email,
    required this.password,
    required this.setName,
    required this.setEmail,
    required this.setPassword,
    required this.submit,
    required this.isLoading,
    required this.error,
  });
}

/// Builder callback that receives the current [SignUpFormState].
typedef SignUpFormWidgetBuilder = Widget Function(SignUpFormState state);

/// A headless sign-up form widget that manages form state internally.
///
/// This widget creates and manages [TextEditingController]s for name,
/// email, and password fields, handles submission via
/// [AuthNotifier.signUp], captures errors, and invokes [onSuccess]
/// when the user successfully authenticates.
class SignUpFormBuilder extends StatefulWidget {
  /// Builder that receives the current form state and returns a widget tree.
  final SignUpFormWidgetBuilder builder;

  /// Called when sign-up succeeds and the auth state becomes [AuthAuthenticated].
  final VoidCallback? onSuccess;

  /// Creates a [SignUpFormBuilder].
  const SignUpFormBuilder({
    required this.builder,
    this.onSuccess,
    super.key,
  });

  @override
  State<SignUpFormBuilder> createState() => _SignUpFormBuilderState();
}

class _SignUpFormBuilderState extends State<SignUpFormBuilder> {
  final TextEditingController _nameController = TextEditingController();
  final TextEditingController _emailController = TextEditingController();
  final TextEditingController _passwordController = TextEditingController();

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
    final name = _nameController.text.trim();
    final email = _emailController.text.trim();
    final password = _passwordController.text;

    if (email.isEmpty || password.isEmpty) {
      setState(() {
        _error = 'Email and password are required.';
      });
      return;
    }

    setState(() {
      _isLoading = true;
      _error = null;
    });

    try {
      await context.auth.signUp(
        email,
        password,
        name: name.isNotEmpty ? name : null,
      );
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
    _nameController.dispose();
    _emailController.dispose();
    _passwordController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    // Re-check auth state on every build (triggered by InheritedNotifier).
    _checkAuthState();

    return widget.builder(
      SignUpFormState(
        name: _nameController.text,
        email: _emailController.text,
        password: _passwordController.text,
        setName: (value) {
          _nameController.text = value;
          setState(() {});
        },
        setEmail: (value) {
          _emailController.text = value;
          setState(() {});
        },
        setPassword: (value) {
          _passwordController.text = value;
          setState(() {});
        },
        submit: _submit,
        isLoading: _isLoading,
        error: _error,
      ),
    );
  }
}
