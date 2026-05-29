import 'package:authsome_flutter_ui/authsome_flutter_ui.dart';
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

class MagicLinkPage extends StatelessWidget {
  const MagicLinkPage({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Magic link')),
      body: Center(
        child: SingleChildScrollView(
          padding: const EdgeInsets.all(24),
          child: MagicLinkForm(
            onSignInTap: () => context.go('/sign-in'),
          ),
        ),
      ),
    );
  }
}
