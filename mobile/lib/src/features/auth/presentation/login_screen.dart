import 'dart:async';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import '../../../core/api/api_client.dart';
import '../../../core/auth/auth_provider.dart';
import '../../../core/theme/nexus_theme.dart';

// ─── Step ─────────────────────────────────────────────────────────────────────
enum _Step { phone, otp, success }

// ─── Screen ───────────────────────────────────────────────────────────────────
class LoginScreen extends ConsumerStatefulWidget {
  const LoginScreen({super.key});
  @override ConsumerState<LoginScreen> createState() => _LoginScreenState();
}

class _LoginScreenState extends ConsumerState<LoginScreen>
    with SingleTickerProviderStateMixin {
  _Step  _step    = _Step.phone;
  bool   _loading = false;
  String _error   = '';
  bool   _sending = false; // resend guard

  // Resend countdown
  int          _resendSeconds = 0;
  Timer?       _resendTimer;

  final _phoneCtrl = TextEditingController();
  // 6 individual OTP boxes
  final _otpCtrls  = List.generate(6, (_) => TextEditingController());
  final _otpFoci   = List.generate(6, (_) => FocusNode());

  // Progress animation
  late final AnimationController _progressCtrl;
  late final Animation<double>   _progress;

  @override
  void initState() {
    super.initState();
    _progressCtrl = AnimationController(
        vsync: this, duration: const Duration(milliseconds: 450));
    _progress = CurvedAnimation(parent: _progressCtrl, curve: Curves.easeOut);
  }

  @override
  void dispose() {
    _progressCtrl.dispose();
    _phoneCtrl.dispose();
    _resendTimer?.cancel();
    for (final c in _otpCtrls) { c.dispose(); }
    for (final f in _otpFoci)  { f.dispose(); }
    super.dispose();
  }

  // ── Helpers ─────────────────────────────────────────────────────────────────

  String get _cleanPhone => _phoneCtrl.text.replaceAll(RegExp(r'\D'), '');
  String get _otpCode    => _otpCtrls.map((c) => c.text).join();

  void _startResendTimer() {
    _resendSeconds = 60;
    _resendTimer?.cancel();
    _resendTimer = Timer.periodic(const Duration(seconds: 1), (_) {
      if (!mounted) return;
      setState(() {
        _resendSeconds--;
        if (_resendSeconds <= 0) _resendTimer?.cancel();
      });
    });
  }

  void _clearOtp() {
    for (final c in _otpCtrls) { c.clear(); }
    _otpFoci[0].requestFocus();
  }

  void _handleOtpInput(int index, String val) {
    final digit = val.replaceAll(RegExp(r'\D'), '');
    if (digit.isEmpty) {
      // Backspace — move back
      if (index > 0) _otpFoci[index - 1].requestFocus();
      return;
    }
    // Handle paste of all 6 digits
    if (digit.length >= 6) {
      for (int i = 0; i < 6; i++) {
        _otpCtrls[i].text = digit[i];
      }
      _otpFoci[5].requestFocus();
      _verifyOtp();
      return;
    }
    _otpCtrls[index].text = digit[0];
    if (index < 5) {
      _otpFoci[index + 1].requestFocus();
    } else {
      // Last box filled — auto-submit
      _verifyOtp();
    }
  }

  // ── API calls ────────────────────────────────────────────────────────────────

  Future<void> _sendOtp({bool resend = false}) async {
    if (_sending) return;
    final phone = _cleanPhone;
    if (phone.length < 11) {
      setState(() => _error = 'Enter a valid 11-digit phone number');
      return;
    }
    setState(() { _loading = !resend; _sending = true; _error = ''; });
    try {
      await ref.read(dioProvider).apiPost<void>(
          '/auth/otp/send',
          data: {'phone_number': phone, 'purpose': 'login'});
      if (mounted) {
        setState(() { _step = _Step.otp; });
        _progressCtrl.animateTo(0.5);
        _startResendTimer();
        await Future.delayed(const Duration(milliseconds: 100));
        _otpFoci[0].requestFocus();
      }
    } on ApiException catch (e) {
      setState(() => _error = e.message);
    } finally {
      if (mounted) setState(() { _loading = false; _sending = false; });
    }
  }

  Future<void> _verifyOtp() async {
    if (_loading) return;
    final code = _otpCode;
    if (code.length < 6) {
      setState(() => _error = 'Enter all 6 digits');
      return;
    }
    setState(() { _loading = true; _error = ''; });
    try {
      final res = await ref.read(dioProvider).apiPost<Map<String, dynamic>>(
          '/auth/otp/verify',
          data: {
            'phone_number': _cleanPhone,
            'code':         code,
            'purpose':      'login',
          });
      final token     = res['token'] as String;
      final isNewUser = res['is_new_user'] as bool? ?? false;

      await ref.read(authStateProvider.notifier).setAuth(
        token:     token,
        phone:     _cleanPhone,
        isNewUser: isNewUser,
      );

      setState(() { _step = _Step.success; });
      _progressCtrl.animateTo(1.0);

      await Future.delayed(const Duration(milliseconds: 900));
      if (mounted) {
        context.go(isNewUser ? '/register' : '/dashboard');
      }
    } on ApiException catch (e) {
      setState(() => _error = e.message);
      _clearOtp();
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  void _goBack() {
    _progressCtrl.animateTo(0.0);
    _resendTimer?.cancel();
    setState(() { _step = _Step.phone; _error = ''; });
    _clearOtp();
  }

  // ── UI ───────────────────────────────────────────────────────────────────────

  @override
  Widget build(BuildContext context) {
    final isOtp     = _step == _Step.otp;
    final isSuccess = _step == _Step.success;

    return Scaffold(
      backgroundColor: NexusColors.background,
      resizeToAvoidBottomInset: true,
      body: SafeArea(
        child: Column(children: [
          // ── Hero ──
          _Hero(),
          const SizedBox(height: 24),

          // ── Form card ──
          Expanded(
            child: Container(
              decoration: const BoxDecoration(
                color: NexusColors.surface,
                borderRadius: BorderRadius.vertical(top: Radius.circular(28)),
              ),
              child: Column(children: [
                // Progress bar
                AnimatedBuilder(
                  animation: _progress,
                  builder: (_, __) => LinearProgressIndicator(
                    value: _progress.value,
                    backgroundColor: NexusColors.border,
                    valueColor: const AlwaysStoppedAnimation(NexusColors.primary),
                    minHeight: 2,
                  ),
                ),

                Expanded(child: SingleChildScrollView(
                  padding: const EdgeInsets.fromLTRB(24, 28, 24, 24),
                  child: AnimatedSwitcher(
                    duration: const Duration(milliseconds: 300),
                    switchInCurve:  Curves.easeOut,
                    switchOutCurve: Curves.easeIn,
                    transitionBuilder: (child, anim) => FadeTransition(
                      opacity:  anim,
                      child:    SlideTransition(
                        position: Tween<Offset>(
                          begin: const Offset(0.05, 0),
                          end: Offset.zero,
                        ).animate(anim),
                        child: child,
                      ),
                    ),
                    child: isSuccess
                        ? _SuccessView(key: const ValueKey('success'))
                        : isOtp
                            ? _OtpView(
                                key: const ValueKey('otp'),
                                phone:        _cleanPhone,
                                ctrls:        _otpCtrls,
                                foci:         _otpFoci,
                                error:        _error,
                                loading:      _loading,
                                resendSeconds: _resendSeconds,
                                onInput:      _handleOtpInput,
                                onVerify:     _verifyOtp,
                                onResend:     () => _sendOtp(resend: true),
                                onBack:       _goBack,
                              )
                            : _PhoneView(
                                key: const ValueKey('phone'),
                                ctrl:    _phoneCtrl,
                                error:   _error,
                                loading: _loading,
                                onSend:  _sendOtp,
                                onChanged: (_) { if (_error.isNotEmpty) setState(() => _error = ''); },
                              ),
                  ),
                )),
              ]),
            ),
          ),
        ]),
      ),
    );
  }
}

// ─── Hero ─────────────────────────────────────────────────────────────────────

class _Hero extends StatelessWidget {
  @override
  Widget build(BuildContext context) => Padding(
    padding: const EdgeInsets.fromLTRB(24, 32, 24, 0),
    child: Column(children: [
      // Logo
      Container(
        width: 76, height: 76,
        decoration: BoxDecoration(
          gradient: NexusColors.gradientBrand,
          borderRadius: NexusRadius.lg,
          boxShadow: NexusShadows.glow,
        ),
        child: const Icon(Icons.bolt_rounded, color: Colors.white, size: 42),
      ),
      const SizedBox(height: 14),
      const Text('Loyalty Nexus',
          style: TextStyle(color: NexusColors.textPrimary, fontSize: 26,
              fontWeight: FontWeight.w900, letterSpacing: -0.5)),
      const SizedBox(height: 4),
      const Text('Recharge · Earn · Spin · Win',
          style: TextStyle(color: NexusColors.textSecondary, fontSize: 13)),
    ]),
  );
}

// ─── Phone view ───────────────────────────────────────────────────────────────

class _PhoneView extends StatelessWidget {
  final TextEditingController ctrl;
  final String error;
  final bool loading;
  final VoidCallback onSend;
  final ValueChanged<String> onChanged;
  const _PhoneView({super.key, required this.ctrl, required this.error,
      required this.loading, required this.onSend, required this.onChanged});

  @override
  Widget build(BuildContext context) => Column(
    crossAxisAlignment: CrossAxisAlignment.start, children: [
    _StepRow(current: 0),
    const SizedBox(height: 22),
    const Text('Welcome back!',
        style: TextStyle(color: NexusColors.textPrimary, fontSize: 24,
            fontWeight: FontWeight.w900)),
    const SizedBox(height: 4),
    const Text('Enter your phone number to continue',
        style: TextStyle(color: NexusColors.textSecondary, fontSize: 14)),
    const SizedBox(height: 24),

    // Phone field
    TextField(
      controller: ctrl,
      keyboardType: TextInputType.phone,
      autofocus:    true,
      onChanged:    onChanged,
      onSubmitted:  (_) => onSend(),
      inputFormatters: [
        FilteringTextInputFormatter.digitsOnly,
        LengthLimitingTextInputFormatter(11),
      ],
      style: const TextStyle(color: NexusColors.textPrimary, fontSize: 18,
          fontWeight: FontWeight.w700, letterSpacing: 4),
      decoration: InputDecoration(
        hintText: '08XX XXX XXXX',
        hintStyle: const TextStyle(color: NexusColors.textMuted, fontSize: 16,
            letterSpacing: 3, fontWeight: FontWeight.w400),
        prefixIcon: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 14),
          child: Row(mainAxisSize: MainAxisSize.min, children: [
            const Text('🇳🇬', style: TextStyle(fontSize: 20)),
            const SizedBox(width: 8),
            Text('+234',
                style: const TextStyle(color: NexusColors.textSecondary, fontSize: 14)),
          ]),
        ),
        errorText:  error.isNotEmpty ? error : null,
      ),
    ),
    const SizedBox(height: 10),
    const Text('We send a 6-digit OTP to this number via SMS.',
        style: TextStyle(color: NexusColors.textSecondary, fontSize: 12)),
    const SizedBox(height: 32),

    ElevatedButton(
      onPressed: loading ? null : onSend,
      child: loading
          ? const SizedBox(width: 22, height: 22,
              child: CircularProgressIndicator(color: Colors.white, strokeWidth: 2.5))
          : const Row(mainAxisAlignment: MainAxisAlignment.center, children: [
              Text('Send OTP', style: TextStyle(fontSize: 16, fontWeight: FontWeight.w800)),
              SizedBox(width: 8),
              Icon(Icons.arrow_forward_rounded, size: 18),
            ]),
    ),
  ]);
}

// ─── OTP view ─────────────────────────────────────────────────────────────────

class _OtpView extends StatelessWidget {
  final String phone, error;
  final bool loading;
  final int resendSeconds;
  final List<TextEditingController> ctrls;
  final List<FocusNode> foci;
  final void Function(int, String) onInput;
  final VoidCallback onVerify, onResend, onBack;
  const _OtpView({
    super.key, required this.phone, required this.error, required this.loading,
    required this.resendSeconds, required this.ctrls, required this.foci,
    required this.onInput, required this.onVerify, required this.onResend,
    required this.onBack,
  });

  String _maskPhone(String p) {
    if (p.length < 8) return p;
    return '${p.substring(0, 3)}****${p.substring(p.length - 4)}';
  }

  @override
  Widget build(BuildContext context) => Column(
    crossAxisAlignment: CrossAxisAlignment.start, children: [
    _StepRow(current: 1),
    const SizedBox(height: 22),
    const Text('Enter OTP',
        style: TextStyle(color: NexusColors.textPrimary, fontSize: 24,
            fontWeight: FontWeight.w900)),
    const SizedBox(height: 4),
    Text('Code sent to ${_maskPhone(phone)}',
        style: const TextStyle(color: NexusColors.textSecondary, fontSize: 14)),
    const SizedBox(height: 28),

    // 6-box OTP
    Row(mainAxisAlignment: MainAxisAlignment.spaceBetween,
      children: List.generate(6, (i) => _OtpBox(
        ctrl:    ctrls[i],
        focus:   foci[i],
        onInput: (v) => onInput(i, v),
        onBack:  i > 0 ? () {
          ctrls[i].clear();
          foci[i - 1].requestFocus();
        } : null,
      )),
    ),

    // Error
    if (error.isNotEmpty) ...[
      const SizedBox(height: 14),
      _ErrorBox(message: error),
    ],

    const SizedBox(height: 20),

    // Resend
    Center(child: resendSeconds > 0
        ? Text('Resend in ${resendSeconds}s',
            style: const TextStyle(color: NexusColors.textSecondary, fontSize: 13))
        : TextButton(
            onPressed: loading ? null : onResend,
            child: const Text('Resend OTP'),
          )),

    const SizedBox(height: 28),

    OutlinedButton.icon(
      onPressed: loading ? null : onBack,
      icon: const Icon(Icons.arrow_back_rounded, size: 16),
      label: const Text('Change number'),
    ),
    const SizedBox(height: 10),
    ElevatedButton(
      onPressed: loading ? null : onVerify,
      child: loading
          ? const SizedBox(width: 22, height: 22,
              child: CircularProgressIndicator(color: Colors.white, strokeWidth: 2.5))
          : const Text('Verify & Continue →',
              style: TextStyle(fontSize: 16, fontWeight: FontWeight.w800)),
    ),
  ]);
}

// ─── OTP single box ───────────────────────────────────────────────────────────

class _OtpBox extends StatelessWidget {
  final TextEditingController ctrl;
  final FocusNode focus;
  final ValueChanged<String> onInput;
  final VoidCallback? onBack;
  const _OtpBox({required this.ctrl, required this.focus, required this.onInput, this.onBack});

  @override
  Widget build(BuildContext context) => SizedBox(
    width: 46, height: 56,
    child: KeyboardListener(
      focusNode: FocusNode(),
      onKeyEvent: (event) {
        if (event is KeyDownEvent &&
            event.logicalKey == LogicalKeyboardKey.backspace &&
            ctrl.text.isEmpty) {
          onBack?.call();
        }
      },
      child: TextField(
        controller: ctrl,
        focusNode:  focus,
        keyboardType: TextInputType.number,
        textAlign: TextAlign.center,
        maxLength: 1,
        inputFormatters: [FilteringTextInputFormatter.digitsOnly],
        onChanged: onInput,
        style: const TextStyle(color: NexusColors.textPrimary,
            fontSize: 22, fontWeight: FontWeight.w800),
        decoration: InputDecoration(
          counterText: '',
          contentPadding: EdgeInsets.zero,
          filled: true,
          fillColor: NexusColors.background,
          border: OutlineInputBorder(
            borderRadius: NexusRadius.md,
            borderSide: const BorderSide(color: NexusColors.border),
          ),
          enabledBorder: OutlineInputBorder(
            borderRadius: NexusRadius.md,
            borderSide: const BorderSide(color: NexusColors.border),
          ),
          focusedBorder: OutlineInputBorder(
            borderRadius: NexusRadius.md,
            borderSide: const BorderSide(color: NexusColors.primary, width: 2),
          ),
        ),
      ),
    ),
  );
}

// ─── Success view ─────────────────────────────────────────────────────────────

class _SuccessView extends StatelessWidget {
  const _SuccessView({super.key});
  @override
  Widget build(BuildContext context) => Center(child: Column(
    mainAxisAlignment: MainAxisAlignment.center,
    children: [
      const SizedBox(height: 40),
      Container(
        width: 80, height: 80,
        decoration: BoxDecoration(
          color: NexusColors.greenDim,
          shape: BoxShape.circle,
          border: Border.all(color: NexusColors.green.withValues(alpha: 0.4), width: 2),
        ),
        child: const Icon(Icons.check_rounded, color: NexusColors.green, size: 44),
      ),
      const SizedBox(height: 20),
      const Text('Verified!', style: TextStyle(color: NexusColors.textPrimary,
          fontSize: 24, fontWeight: FontWeight.w900)),
      const SizedBox(height: 8),
      const Text('Taking you to your dashboard…',
          style: TextStyle(color: NexusColors.textSecondary, fontSize: 14)),
    ],
  ));
}

// ─── Shared small widgets ─────────────────────────────────────────────────────

class _StepRow extends StatelessWidget {
  final int current;
  const _StepRow({required this.current});
  @override
  Widget build(BuildContext context) => Row(children: [
    _Dot(n: 1, active: current == 0, done: current > 0),
    Expanded(child: Container(height: 1, color: current > 0 ? NexusColors.primary : NexusColors.border)),
    _Dot(n: 2, active: current == 1, done: current > 1),
  ]);
}

class _Dot extends StatelessWidget {
  final int n;
  final bool active, done;
  const _Dot({required this.n, required this.active, required this.done});
  @override
  Widget build(BuildContext context) {
    final Color bg = done || active ? NexusColors.primary : NexusColors.border;
    return Container(
      width: 28, height: 28,
      decoration: BoxDecoration(color: bg, shape: BoxShape.circle,
          boxShadow: active ? NexusShadows.glow : null),
      child: Center(child: done
          ? const Icon(Icons.check_rounded, color: Colors.white, size: 14)
          : Text('$n', style: TextStyle(
              color: active ? Colors.white : NexusColors.textSecondary,
              fontSize: 12, fontWeight: FontWeight.w800))),
    );
  }
}

class _ErrorBox extends StatelessWidget {
  final String message;
  const _ErrorBox({required this.message});
  @override
  Widget build(BuildContext context) => Container(
    padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 10),
    decoration: BoxDecoration(
      color: NexusColors.redDim,
      borderRadius: NexusRadius.md,
      border: Border.all(color: NexusColors.red.withValues(alpha: 0.3)),
    ),
    child: Row(children: [
      const Icon(Icons.error_outline_rounded, color: NexusColors.red, size: 16),
      const SizedBox(width: 8),
      Expanded(child: Text(message,
          style: const TextStyle(color: NexusColors.red, fontSize: 13))),
    ]),
  );
}
