import 'package:authsome_flutter_ui/authsome_flutter_ui.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('SocialButtons', () {
    testWidgets('renders nothing when providers is empty', (tester) async {
      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: SocialButtons(
              providers: const [],
              onProviderClick: (_) {},
            ),
          ),
        ),
      );
      expect(find.byType(InkWell), findsNothing);
    });

    testWidgets('renders one button per provider', (tester) async {
      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: SocialButtons(
              providers: const [
                SocialProvider(id: 'google', name: 'Google'),
                SocialProvider(id: 'github', name: 'GitHub'),
              ],
              onProviderClick: (_) {},
            ),
          ),
        ),
      );
      expect(find.text('Google'), findsOneWidget);
      expect(find.text('GitHub'), findsOneWidget);
    });

    testWidgets('tapping a button fires onProviderClick with the right id',
        (tester) async {
      String? clicked;
      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: SocialButtons(
              providers: const [
                SocialProvider(id: 'google', name: 'Google'),
              ],
              onProviderClick: (id) => clicked = id,
            ),
          ),
        ),
      );
      await tester.tap(find.text('Google'));
      expect(clicked, 'google');
    });
  });
}
