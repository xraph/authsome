import 'package:authsome_flutter_ui/authsome_flutter_ui.dart';
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

class MfaChallengePage extends StatelessWidget {
  const MfaChallengePage({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Multi-factor auth')),
      body: Center(
        child: SingleChildScrollView(
          padding: const EdgeInsets.all(24),
          child: MfaChallengeForm(
            onSuccess: () => context.go('/'),
          ),
        ),
      ),
    );
  }
}
