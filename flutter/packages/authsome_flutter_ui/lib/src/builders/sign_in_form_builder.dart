/// Headless sign-in form builder widget.
///
/// Manages email/password form state internally and exposes it via a
/// builder callback. The consuming app provides its own UI while this
/// widget handles input state, submission, error handling, and
/// success detection.
///
/// ```dart
/// SignInFormBuilder(
///   onSuccess: () => Navigator.of(context).pushReplacementNamed('/home'),
///   builder: (state) => Column(
///     children: [
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
///             : const Text('Sign In'),
///       ),
///     ],
///   ),
/// )
/// ```
library;

import 'package:flutter/material.dart';
import 'package:authsome_flutter/authsome_flutter.dart';

/// Immutable snapshot of the sign-in form state, passed to the builder.
class SignInFormState {
  /// Current email value.
  final String email;

  /// Current password value.
  final String password;

  /// Callback to update the email value.
  final ValueChanged<String> setEmail;

  /// Callback to update the password value.
  final ValueChanged<String> setPassword;

  /// Submits the sign-in form.
  final VoidCallback submit;

  /// Whether a sign-in request is currently in progress.
  final bool isLoading;

  /// Error message from the last failed sign-in attempt, or null.
  final String? error;

  /// Creates a [SignInFormState].
  const SignInFormState({
    required this.email,
    required this.password,
    required this.setEmail,
    required this.setPassword,
    required this.submit,
    required this.isLoading,
    required this.error,
  });
}

/// Builder callback that receives the current [SignInFormState].
typedef SignInFormWidgetBuilder = Widget Function(SignInFormState state);

/// A headless sign-in form widget that manages form state internally.
///
/// This widget creates and manages [TextEditingController]s for email
/// and password fields, handles submission via [AuthNotifier.signIn],
/// captures errors, and invokes [onSuccess] when the user successfully
/// authenticates.
class SignInFormBuilder extends StatefulWidget {
  /// Builder that receives the current form state and returns a widget tree.
  final SignInFormWidgetBuilder builder;

  /// Called when sign-in succeeds and the auth state becomes [AuthAuthenticated].
  final VoidCallback? onSuccess;

  /// Creates a [SignInFormBuilder].
  const SignInFormBuilder({
    required this.builder,
    this.onSuccess,
    super.key,
  });

  @override
  State<SignInFormBuilder> createState() => _SignInFormBuilderState();
}

class _SignInFormBuilderState extends State<SignInFormBuilder> {
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
      await context.auth.signIn(email, password);
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
    _emailController.dispose();
    _passwordController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    // Re-check auth state on every build (triggered by InheritedNotifier).
    _checkAuthState();

    return widget.builder(
      SignInFormState(
        email: _emailController.text,
        password: _passwordController.text,
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
