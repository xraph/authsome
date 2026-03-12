/// Social provider icon painters for Google, GitHub, Apple, Microsoft, Twitter.
///
/// Uses [CustomPainter] with SVG path data — no external dependencies.
library;

import 'package:flutter/material.dart';

/// Builds a social icon widget for the given provider ID.
///
/// Returns `null` if the provider is not recognized.
Widget? buildSocialIcon(String providerId, {double size = 20}) {
  switch (providerId.toLowerCase()) {
    case 'google':
      return _GoogleIcon(size: size);
    case 'github':
      return Icon(Icons.code, size: size);
    case 'apple':
      return Icon(Icons.apple, size: size);
    case 'microsoft':
      return _MicrosoftIcon(size: size);
    case 'twitter':
    case 'x':
      return _XIcon(size: size);
    default:
      return null;
  }
}

class _GoogleIcon extends StatelessWidget {
  final double size;
  const _GoogleIcon({required this.size});

  @override
  Widget build(BuildContext context) {
    return CustomPaint(
      size: Size(size, size),
      painter: _GooglePainter(),
    );
  }
}

class _GooglePainter extends CustomPainter {
  @override
  void paint(Canvas canvas, Size size) {
    final s = size.width / 24;
    // Blue
    final bluePaint = Paint()..color = const Color(0xFF4285F4);
    final bluePath = Path()
      ..moveTo(21.8 * s, 10.2 * s)
      ..cubicTo(21.8 * s, 9.5 * s, 21.7 * s, 8.8 * s, 21.6 * s, 8.1 * s)
      ..lineTo(12 * s, 8.1 * s)
      ..lineTo(12 * s, 12.1 * s)
      ..lineTo(17.5 * s, 12.1 * s)
      ..cubicTo(17.3 * s, 13.3 * s, 16.5 * s, 14.4 * s, 15.4 * s, 15 * s)
      ..lineTo(15.4 * s, 17.4 * s)
      ..lineTo(18.7 * s, 17.4 * s)
      ..cubicTo(20.6 * s, 15.7 * s, 21.8 * s, 13.2 * s, 21.8 * s, 10.2 * s)
      ..close();
    canvas.drawPath(bluePath, bluePaint);

    // Green
    final greenPaint = Paint()..color = const Color(0xFF34A853);
    final greenPath = Path()
      ..moveTo(12 * s, 22 * s)
      ..cubicTo(14.7 * s, 22 * s, 17 * s, 21 * s, 18.7 * s, 17.4 * s)
      ..lineTo(15.4 * s, 15 * s)
      ..cubicTo(14.5 * s, 15.6 * s, 13.4 * s, 16 * s, 12 * s, 16 * s)
      ..cubicTo(9.4 * s, 16 * s, 7.2 * s, 14.1 * s, 6.4 * s, 11.6 * s)
      ..lineTo(3 * s, 11.6 * s)
      ..lineTo(3 * s, 14.1 * s)
      ..cubicTo(4.7 * s, 18.5 * s, 8 * s, 22 * s, 12 * s, 22 * s)
      ..close();
    canvas.drawPath(greenPath, greenPaint);

    // Yellow
    final yellowPaint = Paint()..color = const Color(0xFFFBBC05);
    final yellowPath = Path()
      ..moveTo(6.4 * s, 11.6 * s)
      ..cubicTo(6.2 * s, 11 * s, 6 * s, 10.5 * s, 6 * s, 10 * s)
      ..cubicTo(6 * s, 9.5 * s, 6.1 * s, 9 * s, 6.4 * s, 8.4 * s)
      ..lineTo(6.4 * s, 5.9 * s)
      ..lineTo(3 * s, 5.9 * s)
      ..cubicTo(2.4 * s, 7.1 * s, 2 * s, 8.5 * s, 2 * s, 10 * s)
      ..cubicTo(2 * s, 11.5 * s, 2.4 * s, 12.9 * s, 3 * s, 14.1 * s)
      ..lineTo(6.4 * s, 11.6 * s)
      ..close();
    canvas.drawPath(yellowPath, yellowPaint);

    // Red
    final redPaint = Paint()..color = const Color(0xFFEA4335);
    final redPath = Path()
      ..moveTo(12 * s, 4 * s)
      ..cubicTo(13.5 * s, 4 * s, 14.9 * s, 4.5 * s, 16 * s, 5.5 * s)
      ..lineTo(18.7 * s, 2.8 * s)
      ..cubicTo(17 * s, 1.2 * s, 14.7 * s, 0 * s, 12 * s, 0 * s)
      ..cubicTo(8 * s, 0 * s, 4.7 * s, 3.5 * s, 3 * s, 5.9 * s)
      ..lineTo(6.4 * s, 8.4 * s)
      ..cubicTo(7.2 * s, 5.9 * s, 9.4 * s, 4 * s, 12 * s, 4 * s)
      ..close();
    canvas.drawPath(redPath, redPaint);
  }

  @override
  bool shouldRepaint(covariant CustomPainter oldDelegate) => false;
}

class _MicrosoftIcon extends StatelessWidget {
  final double size;
  const _MicrosoftIcon({required this.size});

  @override
  Widget build(BuildContext context) {
    final half = size / 2 - 1;
    return SizedBox(
      width: size,
      height: size,
      child: Column(
        children: [
          Row(
            children: [
              Container(width: half, height: half, color: const Color(0xFFF25022)),
              const SizedBox(width: 2),
              Container(width: half, height: half, color: const Color(0xFF7FBA00)),
            ],
          ),
          const SizedBox(height: 2),
          Row(
            children: [
              Container(width: half, height: half, color: const Color(0xFF00A4EF)),
              const SizedBox(width: 2),
              Container(width: half, height: half, color: const Color(0xFFFFB900)),
            ],
          ),
        ],
      ),
    );
  }
}

class _XIcon extends StatelessWidget {
  final double size;
  const _XIcon({required this.size});

  @override
  Widget build(BuildContext context) {
    return Icon(Icons.close, size: size);
  }
}
