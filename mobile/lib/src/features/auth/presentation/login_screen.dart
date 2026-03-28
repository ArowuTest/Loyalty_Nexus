import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import '../../../core/api/api_client.dart';
import '../../../core/auth/auth_provider.dart';
import '../../../core/theme/nexus_theme.dart';

// ─── Step enum ────────────────────────────────────────────────────────────────

enum _Step { phone, otp }

// ─── Screen ───────────────────────────────────────────────────────────────────

class LoginScreen extends ConsumerStatefulWidget {
  const LoginScreen({super.key});
  @override ConsumerState<LoginScreen> createState() => _LoginScreenState();
}

class _LoginScreenState extends ConsumerState<LoginScreen>
    with SingleTickerProviderStateMixin {
  _Step  _step   = _Step.phone;
  bool   _loading = false;
  String _error   = '';

  final _phoneCtrl = TextEditingController();
  final _otpCtrl   = TextEditingController();

  // Animated progress line
  late final AnimationController _progressAnim;
  late final Animation<double>   _progress;

  @override
  void initState() {
    super.initState();
    _progressAnim = AnimationController(
        vsync: this, duration: const Duration(milliseconds: 400));
    _progress = CurvedAnimation(parent: _progressAnim, curve: Curves.easeOut);
  }

  @override
  void dispose() {
    _progressAnim.dispose();
    _phoneCtrl.dispose();
    _otpCtrl.dispose();
    super.dispose();
  }

  String get _cleanPhone => _phoneCtrl.text.replaceAll(RegExp(r'\D'), '');

  Future<void> _sendOtp() async {
    final phone = _cleanPhone;
    if (phone.length < 11) {
      setState(() => _error = 'Enter a valid 11-digit phone number');
      return;
    }
    setState(() { _loading = true; _error = ''; });
    try {
      await ref.read(dioProvider).apiPost<void>(
          '/auth/otp/send', data: {'phone_number': phone, 'purpose': 'login'});
      setState(() { _step = _Step.otp; _loading = false; });
      _progressAnim.animateTo(0.5);
    } on ApiException catch (e) {
      setState(() { _error = e.message; _loading = false; });
    } catch (e) {
      setState(() { _error = 'Something went wrong. Please try again.'; _loading = false; });
    }
  }

  Future<void> _verifyOtp() async {
    final code = _otpCtrl.text.trim();
    if (code.length < 4) {
      setState(() => _error = 'Enter the OTP sent to your phone');
      return;
    }
    setState(() { _loading = true; _error = ''; });
    try {
      final res = await ref.read(dioProvider).apiPost<Map<String, dynamic>>(
          '/auth/otp/verify',
          data: {'phone_number': _cleanPhone, 'code': code, 'purpose': 'login'});
      await ref.read(authStateProvider.notifier)
          .setAuth(res['token'] as String, _cleanPhone);
      _progressAnim.animateTo(1.0);
      if (mounted) context.go('/dashboard');
    } on ApiException catch (e) {
      setState(() { _error = e.message; _loading = false; });
    } catch (e) {
      setState(() { _error = 'Verification failed. Please try again.'; _loading = false; });
    }
  }

  void _goBack() {
    _progressAnim.animateTo(0.0);
    setState(() { _step = _Step.phone; _error = ''; _otpCtrl.clear(); });
  }

  @override
  Widget build(BuildContext context) {
    final isOtp = _step == _Step.otp;

    return Scaffold(
      backgroundColor: NexusColors.background,
      body: SafeArea(
        child: SingleChildScrollView(
          child: ConstrainedBox(
            constraints: BoxConstraints(
              minHeight: MediaQuery.of(context).size.height -
                  MediaQuery.of(context).padding.top -
                  MediaQuery.of(context).padding.bottom,
            ),
            child: IntrinsicHeight(
              child: Column(children: [

                // ── Hero ──
                _HeroBanner(),
                const SizedBox(height: 32),

                Expanded(
                  child: Container(
                    width: double.infinity,
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
                          color: NexusColors.primary,
                          minHeight: 2,
                        ),
                      ),

                      Expanded(child: Padding(
                        padding: const EdgeInsets.fromLTRB(24, 28, 24, 24),
                        child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [

                          // Step indicator
                          Row(children: [
                            _StepDot(active: !isOtp, done: isOtp, label: '1'),
                            Expanded(child: Container(height: 1,
                                color: isOtp ? NexusColors.primary : NexusColors.border)),
                            _StepDot(active: isOtp, done: false, label: '2'),
                          ]),
                          const SizedBox(height: 24),

                          // Title
                          AnimatedSwitcher(
                            duration: const Duration(milliseconds: 250),
                            child: Column(
                              key: ValueKey(isOtp),
                              crossAxisAlignment: CrossAxisAlignment.start,
                              children: [
                                Text(
                                  isOtp ? 'Enter OTP' : 'Welcome back!',
                                  style: const TextStyle(
                                    color: NexusColors.textPrimary,
                                    fontSize: 24, fontWeight: FontWeight.w900,
                                  ),
                                ),
                                const SizedBox(height: 4),
                                Text(
                                  isOtp
                                      ? 'Code sent to ${_maskPhone(_cleanPhone)}'
                                      : 'Enter your phone to continue',
                                  style: const TextStyle(
                                      color: NexusColors.textSecondary, fontSize: 14),
                                ),
                              ],
                            ),
                          ),
                          const SizedBox(height: 24),

                          // ── Phone input ──
                          if (!isOtp) ...[
                            TextField(
                              controller: _phoneCtrl,
                              keyboardType: TextInputType.phone,
                              autofocus: true,
                              inputFormatters: [
                                FilteringTextInputFormatter.digitsOnly,
                                LengthLimitingTextInputFormatter(11),
                              ],
                              style: const TextStyle(
                                  color: NexusColors.textPrimary, fontSize: 20,
                                  fontWeight: FontWeight.w700, letterSpacing: 3),
                              onSubmitted: (_) => _sendOtp(),
                              decoration: InputDecoration(
                                hintText: '080X XXX XXXX',
                                hintStyle: TextStyle(
                                    color: Colors.white.withOpacity(0.2),
                                    fontSize: 18, letterSpacing: 3, fontWeight: FontWeight.w400),
                                prefixIcon: const Padding(
                                  padding: EdgeInsets.symmetric(horizontal: 14),
                                  child: Row(mainAxisSize: MainAxisSize.min, children: [
                                    Text('🇳🇬', style: TextStyle(fontSize: 20)),
                                    SizedBox(width: 8),
                                    Text('+234', style: TextStyle(
                                        color: NexusColors.textSecondary, fontSize: 14)),
                                  ]),
                                ),
                                filled: true, fillColor: NexusColors.background,
                                border: OutlineInputBorder(
                                    borderRadius: BorderRadius.circular(16),
                                    borderSide: const BorderSide(color: NexusColors.border)),
                                enabledBorder: OutlineInputBorder(
                                    borderRadius: BorderRadius.circular(16),
                                    borderSide: const BorderSide(color: NexusColors.border)),
                                focusedBorder: OutlineInputBorder(
                                    borderRadius: BorderRadius.circular(16),
                                    borderSide: const BorderSide(
                                        color: NexusColors.primary, width: 1.5)),
                              ),
                            ),
                            const SizedBox(height: 10),
                            Text('We\'ll send a 4–6 digit OTP to this number via SMS',
                                style: const TextStyle(
                                    color: NexusColors.textSecondary, fontSize: 12)),
                          ],

                          // ── OTP input ──
                          if (isOtp) ...[
                            TextField(
                              controller: _otpCtrl,
                              keyboardType: TextInputType.number,
                              autofocus: true,
                              textAlign: TextAlign.center,
                              maxLength: 6,
                              inputFormatters: [FilteringTextInputFormatter.digitsOnly],
                              style: const TextStyle(
                                  color: NexusColors.textPrimary, fontSize: 32,
                                  fontWeight: FontWeight.w900, letterSpacing: 14),
                              onSubmitted: (_) => _verifyOtp(),
                              decoration: InputDecoration(
                                counterText: '',
                                hintText: '• • • •',
                                hintStyle: TextStyle(
                                    color: Colors.white.withOpacity(0.15),
                                    fontSize: 32, letterSpacing: 14),
                                filled: true, fillColor: NexusColors.background,
                                border: OutlineInputBorder(
                                    borderRadius: BorderRadius.circular(16),
                                    borderSide: const BorderSide(color: NexusColors.border)),
                                enabledBorder: OutlineInputBorder(
                                    borderRadius: BorderRadius.circular(16),
                                    borderSide: const BorderSide(color: NexusColors.border)),
                                focusedBorder: OutlineInputBorder(
                                    borderRadius: BorderRadius.circular(16),
                                    borderSide: const BorderSide(
                                        color: NexusColors.primary, width: 1.5)),
                              ),
                            ),
                            const SizedBox(height: 10),
                            Center(child: TextButton(
                              onPressed: _loading ? null : _sendOtp,
                              child: const Text('Resend OTP',
                                  style: TextStyle(color: NexusColors.primary, fontSize: 13)),
                            )),
                          ],

                          // ── Error ──
                          if (_error.isNotEmpty) ...[
                            const SizedBox(height: 12),
                            Container(
                              padding: const EdgeInsets.symmetric(
                                  horizontal: 14, vertical: 10),
                              decoration: BoxDecoration(
                                color: NexusColors.red.withOpacity(0.08),
                                borderRadius: BorderRadius.circular(12),
                                border: Border.all(
                                    color: NexusColors.red.withOpacity(0.25)),
                              ),
                              child: Row(children: [
                                const Icon(Icons.error_outline_rounded,
                                    color: NexusColors.red, size: 16),
                                const SizedBox(width: 8),
                                Expanded(child: Text(_error,
                                    style: const TextStyle(
                                        color: NexusColors.red, fontSize: 13))),
                              ]),
                            ),
                          ],

                          const Spacer(),

                          // ── Action buttons ──
                          if (isOtp) ...[
                            OutlinedButton.icon(
                              onPressed: _loading ? null : _goBack,
                              icon: const Icon(Icons.arrow_back_rounded, size: 16),
                              label: const Text('Change number'),
                              style: OutlinedButton.styleFrom(
                                foregroundColor: NexusColors.textSecondary,
                                side: const BorderSide(color: NexusColors.border),
                                minimumSize: const Size(double.infinity, 52),
                                shape: RoundedRectangleBorder(
                                    borderRadius: BorderRadius.circular(16)),
                              ),
                            ),
                            const SizedBox(height: 10),
                          ],

                          ElevatedButton(
                            onPressed: _loading ? null
                                : (isOtp ? _verifyOtp : _sendOtp),
                            style: ElevatedButton.styleFrom(
                              backgroundColor: NexusColors.primary,
                              minimumSize: const Size(double.infinity, 56),
                              shape: RoundedRectangleBorder(
                                  borderRadius: BorderRadius.circular(16)),
                            ),
                            child: _loading
                                ? const SizedBox(width: 24, height: 24,
                                    child: CircularProgressIndicator(
                                        color: Colors.white, strokeWidth: 2.5))
                                : Text(
                                    isOtp ? 'Verify & Enter →' : 'Send OTP →',
                                    style: const TextStyle(
                                        fontSize: 16, fontWeight: FontWeight.w800),
                                  ),
                          ),
                        ]),
                      )),
                    ]),
                  ),
                ),
              ]),
            ),
          ),
        ),
      ),
    );
  }
}

// ─── Sub-widgets ──────────────────────────────────────────────────────────────

class _HeroBanner extends StatelessWidget {
  @override
  Widget build(BuildContext context) => Padding(
    padding: const EdgeInsets.fromLTRB(24, 40, 24, 0),
    child: Column(children: [
      // Logo
      Container(
        width: 80, height: 80,
        decoration: BoxDecoration(
          gradient: const LinearGradient(
            colors: [Color(0xFF4A56EE), Color(0xFF8B5CF6)],
            begin: Alignment.topLeft, end: Alignment.bottomRight,
          ),
          borderRadius: BorderRadius.circular(24),
          boxShadow: [
            BoxShadow(color: const Color(0xFF4A56EE).withOpacity(0.4),
                blurRadius: 24, offset: const Offset(0, 8)),
          ],
        ),
        child: const Icon(Icons.bolt_rounded, color: Colors.white, size: 44),
      ),
      const SizedBox(height: 16),
      const Text('Loyalty Nexus', style: TextStyle(
          color: Colors.white, fontSize: 28, fontWeight: FontWeight.w900)),
      const SizedBox(height: 4),
      const Text('Recharge · Earn · Spin · Win',
          style: TextStyle(color: NexusColors.textSecondary, fontSize: 14)),
    ]),
  );
}

class _StepDot extends StatelessWidget {
  final bool active, done;
  final String label;
  const _StepDot({required this.active, required this.done, required this.label});

  @override
  Widget build(BuildContext context) {
    final Color bg = done || active ? NexusColors.primary : NexusColors.border;
    return Container(
      width: 28, height: 28,
      decoration: BoxDecoration(color: bg, shape: BoxShape.circle),
      child: Center(child: done
          ? const Icon(Icons.check_rounded, color: Colors.white, size: 14)
          : Text(label, style: TextStyle(
              color: active ? Colors.white : NexusColors.textSecondary,
              fontSize: 12, fontWeight: FontWeight.w800))),
    );
  }
}

String _maskPhone(String phone) {
  if (phone.length < 8) return phone;
  return '${phone.substring(0, 3)}****${phone.substring(phone.length - 3)}';
}
