import 'package:authsome_flutter_ui/authsome_flutter_ui.dart';
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

class VerifyEmailPage extends StatelessWidget {
  final String? email;

  const VerifyEmailPage({super.key, this.email});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Verify your email')),
      body: Center(
        child: SingleChildScrollView(
          padding: const EdgeInsets.all(24),
          child: email == null
              ? const Text(
                  'No email provided. Open this page from the link in your inbox.',
                )
              : EmailVerificationForm(
                  email: email!,
                  onSuccess: () => context.go('/'),
                ),
        ),
      ),
    );
  }
}
