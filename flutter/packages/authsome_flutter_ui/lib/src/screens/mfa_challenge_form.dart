/// MFA challenge form screen with multi-method support.
///
/// Supports TOTP (authenticator app), SMS, and recovery code methods.
/// Auto-discovers available methods from [ClientConfig] or accepts
/// an explicit override via the [methods] prop.
library;

import 'dart:async';

import 'package:flutter/material.dart';
import 'package:authsome_flutter/authsome_flutter.dart';

import '../theme/auth_theme.dart';
import '../widgets/auth_card.dart';
import '../widgets/error_display.dart';
import '../widgets/otp_input.dart';
import '../widgets/loading_indicator.dart';

/// MFA method identifiers.
enum MfaMethod {
  /// Time-based One-Time Password (authenticator app).
  totp,

  /// SMS verification code.
  sms,

  /// Recovery code.
  recovery,
}

/// A multi-method MFA challenge form wrapped in an [AuthCard].
///
/// Renders different input UIs depending on the selected method:
/// - **TOTP**: 6-digit [OtpInput] with auto-submit.
/// - **SMS**: "Send code" button → [OtpInput] with 60 s resend cooldown.
/// - **Recovery**: A [TextField] for a recovery code.
///
/// A method switcher at the bottom allows toggling between available methods.
class MfaChallengeForm extends StatefulWidget {
  /// The MFA enrollment ID for TOTP verification.
  final String enrollmentId;

  /// Called when MFA verification succeeds.
  final VoidCallback? onSuccess;

  /// Override auto-detected MFA methods.
  ///
  /// If null, methods are read from [ClientConfig.mfa.methods].
  final List<MfaMethod>? methods;

  /// The method shown initially (default: first in list).
  final MfaMethod? defaultMethod;

  /// Optional logo widget displayed above the title.
  final Widget? logo;

  // ── Localization overrides ──

  /// Card title (default: "Two-factor authentication").
  final String titleText;

  /// TOTP description (default: "Enter the 6-digit code from your authenticator app").
  final String totpDescriptionText;

  /// SMS description before sending (default: "We'll send a verification code to your phone").
  final String smsDescriptionText;

  /// SMS description after sending (default: "Enter the code sent to {phone}").
  final String? smsSentDescriptionText;

  /// Recovery description (default: "Enter one of your recovery codes").
  final String recoveryDescriptionText;

  /// Send code button label (default: "Send code").
  final String sendCodeLabel;

  /// Resend code button label (default: "Resend code").
  final String resendCodeLabel;

  /// Verify button label (default: "Verify").
  final String verifyLabel;

  /// TOTP switcher label (default: "Use authenticator app").
  final String totpSwitcherLabel;

  /// SMS switcher label (default: "Use SMS").
  final String smsSwitcherLabel;

  /// Recovery switcher label (default: "Use recovery code").
  final String recoverySwitcherLabel;

  const MfaChallengeForm({
    required this.enrollmentId,
    this.onSuccess,
    this.methods,
    this.defaultMethod,
    this.logo,
    this.titleText = 'Two-factor authentication',
    this.totpDescriptionText =
        'Enter the 6-digit code from your authenticator app',
    this.smsDescriptionText =
        "We'll send a verification code to your phone",
    this.smsSentDescriptionText,
    this.recoveryDescriptionText = 'Enter one of your recovery codes',
    this.sendCodeLabel = 'Send code',
    this.resendCodeLabel = 'Resend code',
    this.verifyLabel = 'Verify',
    this.totpSwitcherLabel = 'Use authenticator app',
    this.smsSwitcherLabel = 'Use SMS',
    this.recoverySwitcherLabel = 'Use recovery code',
    super.key,
  });

  @override
  State<MfaChallengeForm> createState() => _MfaChallengeFormState();
}

class _MfaChallengeFormState extends State<MfaChallengeForm> {
  final _recoveryController = TextEditingController();

  late List<MfaMethod> _availableMethods;
  late MfaMethod _activeMethod;

  String? _error;
  bool _isSubmitting = false;

  // SMS state.
  bool _isSmsSent = false;
  String? _phoneMasked;
  int _resendCooldown = 0;
  Timer? _resendTimer;

  AuthNotifier? _auth;

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    if (_auth == null) {
      _auth = context.auth;
      _auth!.addListener(_onAuthStateChanged);
      _availableMethods = _resolveMethods();
      _activeMethod = widget.defaultMethod ?? _availableMethods.first;
    }
  }

  @override
  void dispose() {
    _auth?.removeListener(_onAuthStateChanged);
    _recoveryController.dispose();
    _resendTimer?.cancel();
    super.dispose();
  }

  void _onAuthStateChanged() {
    if (!mounted) return;
    final auth = _auth!;

    if (auth.state is AuthAuthenticated) {
      widget.onSuccess?.call();
      return;
    }

    if (auth.error != null && mounted) {
      setState(() {
        _error = auth.error;
        _isSubmitting = false;
      });
    }
  }

  List<MfaMethod> _resolveMethods() {
    if (widget.methods != null && widget.methods!.isNotEmpty) {
      return widget.methods!;
    }

    final config = _auth?.clientConfig;
    if (config?.mfa?.enabled != true) return [MfaMethod.totp];

    final methods = <MfaMethod>[];
    for (final m in config!.mfa!.methods) {
      switch (m.toLowerCase()) {
        case 'totp':
          methods.add(MfaMethod.totp);
        case 'sms':
          methods.add(MfaMethod.sms);
        case 'recovery':
          methods.add(MfaMethod.recovery);
      }
    }
    // Always allow recovery as a fallback.
    if (!methods.contains(MfaMethod.recovery)) {
      methods.add(MfaMethod.recovery);
    }
    return methods.isEmpty ? [MfaMethod.totp] : methods;
  }

  void _switchMethod(MfaMethod method) {
    setState(() {
      _activeMethod = method;
      _error = null;
    });
  }

  // ── TOTP ──

  Future<void> _onTotpCompleted(String code) async {
    setState(() {
      _error = null;
      _isSubmitting = true;
    });

    try {
      await _auth!.submitMFACode(widget.enrollmentId, code);
    } catch (e) {
      if (mounted) {
        setState(() {
          _error = e.toString();
          _isSubmitting = false;
        });
      }
    }
  }

  // ── SMS ──

  Future<void> _onSendSms() async {
    setState(() {
      _error = null;
      _isSubmitting = true;
    });

    try {
      final result = await _auth!.sendSMSCode();
      if (mounted) {
        setState(() {
          _isSmsSent = result.sent;
          _phoneMasked = result.phoneMasked;
          _isSubmitting = false;
        });
        _startResendCooldown(result.expiresInSeconds);
      }
    } catch (e) {
      if (mounted) {
        setState(() {
          _error = e.toString();
          _isSubmitting = false;
        });
      }
    }
  }

  void _startResendCooldown(int seconds) {
    _resendTimer?.cancel();
    setState(() => _resendCooldown = seconds);
    _resendTimer = Timer.periodic(const Duration(seconds: 1), (timer) {
      if (!mounted) {
        timer.cancel();
        return;
      }
      setState(() {
        _resendCooldown--;
        if (_resendCooldown <= 0) {
          timer.cancel();
        }
      });
    });
  }

  Future<void> _onSmsCodeCompleted(String code) async {
    setState(() {
      _error = null;
      _isSubmitting = true;
    });

    try {
      await _auth!.submitSMSCode(code);
    } catch (e) {
      if (mounted) {
        setState(() {
          _error = e.toString();
          _isSubmitting = false;
        });
      }
    }
  }

  // ── Recovery ──

  Future<void> _onRecoverySubmit() async {
    final code = _recoveryController.text.trim();
    if (code.isEmpty) {
      setState(() => _error = 'Please enter a recovery code');
      return;
    }

    setState(() {
      _error = null;
      _isSubmitting = true;
    });

    try {
      await _auth!.submitRecoveryCode(code);
    } catch (e) {
      if (mounted) {
        setState(() {
          _error = e.toString();
          _isSubmitting = false;
        });
      }
    }
  }

  // ── Build ──

  String get _description => switch (_activeMethod) {
        MfaMethod.totp => widget.totpDescriptionText,
        MfaMethod.sms => _isSmsSent
            ? (widget.smsSentDescriptionText ??
                'Enter the code sent to ${_phoneMasked ?? 'your phone'}')
            : widget.smsDescriptionText,
        MfaMethod.recovery => widget.recoveryDescriptionText,
      };

  @override
  Widget build(BuildContext context) {
    final theme = AuthTheme.of(context);

    return AuthCard(
      title: widget.titleText,
      description: _description,
      logo: widget.logo,
      footer: _buildMethodSwitcher(context),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        mainAxisSize: MainAxisSize.min,
        children: [
          ErrorDisplay(error: _error),
          if (_error != null) SizedBox(height: theme.fieldSpacing),
          AnimatedSwitcher(
            duration: const Duration(milliseconds: 250),
            child: switch (_activeMethod) {
              MfaMethod.totp => _buildTotpView(context, theme),
              MfaMethod.sms => _buildSmsView(context, theme),
              MfaMethod.recovery => _buildRecoveryView(context, theme),
            },
          ),
        ],
      ),
    );
  }

  Widget _buildTotpView(BuildContext context, AuthThemeData theme) {
    return Column(
      key: const ValueKey('mfa-totp'),
      mainAxisSize: MainAxisSize.min,
      children: [
        OtpInput(
          onCompleted: _onTotpCompleted,
          enabled: !_isSubmitting,
          length: 6,
        ),
        if (_isSubmitting) ...[
          SizedBox(height: theme.fieldSpacing),
          const Center(child: LoadingIndicator(size: LoadingSize.sm)),
        ],
      ],
    );
  }

  Widget _buildSmsView(BuildContext context, AuthThemeData theme) {
    if (!_isSmsSent) {
      return Column(
        key: const ValueKey('mfa-sms-send'),
        crossAxisAlignment: CrossAxisAlignment.stretch,
        mainAxisSize: MainAxisSize.min,
        children: [
          FilledButton(
            onPressed: _isSubmitting ? null : _onSendSms,
            child: _isSubmitting
                ? const LoadingIndicator(size: LoadingSize.sm)
                : Text(widget.sendCodeLabel),
          ),
        ],
      );
    }

    return Column(
      key: const ValueKey('mfa-sms-verify'),
      mainAxisSize: MainAxisSize.min,
      children: [
        OtpInput(
          onCompleted: _onSmsCodeCompleted,
          enabled: !_isSubmitting,
          length: 6,
        ),
        SizedBox(height: theme.fieldSpacing),
        if (_isSubmitting)
          const Center(child: LoadingIndicator(size: LoadingSize.sm))
        else
          TextButton(
            onPressed: _resendCooldown > 0 ? null : _onSendSms,
            child: Text(
              _resendCooldown > 0
                  ? '${widget.resendCodeLabel} (${_resendCooldown}s)'
                  : widget.resendCodeLabel,
            ),
          ),
      ],
    );
  }

  Widget _buildRecoveryView(BuildContext context, AuthThemeData theme) {
    return Column(
      key: const ValueKey('mfa-recovery'),
      crossAxisAlignment: CrossAxisAlignment.stretch,
      mainAxisSize: MainAxisSize.min,
      children: [
        TextField(
          controller: _recoveryController,
          enabled: !_isSubmitting,
          textInputAction: TextInputAction.done,
          onSubmitted: (_) => _onRecoverySubmit(),
          decoration: const InputDecoration(
            labelText: 'Recovery code',
            hintText: 'xxxx-xxxx-xxxx',
            border: OutlineInputBorder(),
          ),
        ),
        SizedBox(height: theme.fieldSpacing),
        FilledButton(
          onPressed: _isSubmitting ? null : _onRecoverySubmit,
          child: _isSubmitting
              ? const LoadingIndicator(size: LoadingSize.sm)
              : Text(widget.verifyLabel),
        ),
      ],
    );
  }

  Widget? _buildMethodSwitcher(BuildContext context) {
    if (_availableMethods.length <= 1) return null;

    final colorScheme = Theme.of(context).colorScheme;
    final otherMethods =
        _availableMethods.where((m) => m != _activeMethod).toList();

    return Column(
      mainAxisSize: MainAxisSize.min,
      children: [
        Divider(color: colorScheme.outlineVariant),
        const SizedBox(height: 8),
        Wrap(
          spacing: 8,
          runSpacing: 4,
          alignment: WrapAlignment.center,
          children: otherMethods.map((m) {
            final label = switch (m) {
              MfaMethod.totp => widget.totpSwitcherLabel,
              MfaMethod.sms => widget.smsSwitcherLabel,
              MfaMethod.recovery => widget.recoverySwitcherLabel,
            };
            return TextButton(
              onPressed: _isSubmitting ? null : () => _switchMethod(m),
              child: Text(label),
            );
          }).toList(),
        ),
      ],
    );
  }
}
