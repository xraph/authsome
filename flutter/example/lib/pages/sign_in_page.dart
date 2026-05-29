import 'package:authsome_flutter_ui/authsome_flutter_ui.dart';
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

class SignInPage extends StatelessWidget {
  const SignInPage({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Sign in')),
      body: Center(
        child: SingleChildScrollView(
          padding: const EdgeInsets.all(24),
          child: SignInForm(
            onSuccess: () => context.go('/'),
            onSignUpTap: () => context.go('/sign-up'),
            onForgotPasswordTap: () => context.go('/forgot-password'),
          ),
        ),
      ),
    );
  }
}
