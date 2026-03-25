import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import '../../../core/api/api_client.dart';
import '../../../core/auth/auth_provider.dart';
import '../../../core/theme/nexus_theme.dart';

enum _Step { phone, otp }

class LoginScreen extends ConsumerStatefulWidget {
  const LoginScreen({super.key});
  @override ConsumerState<LoginScreen> createState() => _State();
}

class _State extends ConsumerState<LoginScreen> {
  _Step step = _Step.phone;
  final pc = TextEditingController();
  final oc = TextEditingController();
  bool loading = false;
  String error = '';

  Future<void> sendOTP() async {
    final p = pc.text.replaceAll(RegExp(r'\D'), '');
    if (p.length < 11) { setState(() => error = 'Enter valid 11-digit number'); return; }
    setState(() { loading = true; error = ''; });
    try {
      await ref.read(dioProvider).apiPost('/auth/otp/send', data: {'phone_number': p, 'purpose': 'login'});
      setState(() { step = _Step.otp; loading = false; });
    } on ApiException catch (e) { setState(() { error = e.message; loading = false; }); }
  }

  Future<void> verifyOTP() async {
    final p = pc.text.replaceAll(RegExp(r'\D'), '');
    final c = oc.text.trim();
    if (c.length < 4) { setState(() => error = 'Enter the OTP code'); return; }
    setState(() { loading = true; error = ''; });
    try {
      final res = await ref.read(dioProvider).apiPost<Map<String, dynamic>>(
        '/auth/otp/verify', data: {'phone_number': p, 'code': c, 'purpose': 'login'});
      await ref.read(authStateProvider.notifier).setAuth(res['token'] as String, p);
      if (mounted) context.go('/dashboard');
    } on ApiException catch (e) { setState(() { error = e.message; loading = false; }); }
  }

  @override
  Widget build(BuildContext ctx) => Scaffold(
    body: SafeArea(child: SingleChildScrollView(
      padding: const EdgeInsets.all(24),
      child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
        const SizedBox(height: 48),
        Center(child: Column(children: [
          Container(
            width: 72, height: 72,
            decoration: BoxDecoration(
              gradient: const LinearGradient(colors: [NexusColors.primary, Color(0xFF8B5CF6)]),
              borderRadius: BorderRadius.circular(20)),
            child: const Icon(Icons.bolt, color: Colors.white, size: 40)),
          const SizedBox(height: 16),
          Text('Loyalty Nexus', style: Theme.of(ctx).textTheme.displayMedium),
          const SizedBox(height: 4),
          Text('Recharge. Earn. Spin. Win.', style: Theme.of(ctx).textTheme.bodyMedium),
        ])),
        const SizedBox(height: 48),
        Text(step == _Step.phone ? 'Enter your phone' : 'Enter OTP',
          style: Theme.of(ctx).textTheme.titleLarge),
        const SizedBox(height: 4),
        Text(step == _Step.phone ? "We'll send a code via SMS" : 'Code sent to your phone',
          style: Theme.of(ctx).textTheme.bodyMedium),
        const SizedBox(height: 20),
        if (step == _Step.phone)
          TextField(
            controller: pc, keyboardType: TextInputType.phone,
            style: const TextStyle(color: NexusColors.textPrimary, fontSize: 18),
            decoration: const InputDecoration(hintText: '080X XXX XXXX',
              prefixIcon: Icon(Icons.phone_android, color: NexusColors.textSecondary)),
            onSubmitted: (_) => sendOTP())
        else
          TextField(
            controller: oc, keyboardType: TextInputType.number,
            textAlign: TextAlign.center, maxLength: 6,
            style: const TextStyle(color: NexusColors.textPrimary, fontSize: 28,
              fontWeight: FontWeight.bold, letterSpacing: 12),
            decoration: const InputDecoration(hintText: '— — — —', counterText: ''),
            onSubmitted: (_) => verifyOTP()),
        if (error.isNotEmpty) ...[
          const SizedBox(height: 12),
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
            decoration: BoxDecoration(
              color: NexusColors.red.withOpacity(0.1),
              borderRadius: BorderRadius.circular(8),
              border: Border.all(color: NexusColors.red.withOpacity(0.3))),
            child: Text(error, style: const TextStyle(color: NexusColors.red, fontSize: 13))),
        ],
        const SizedBox(height: 24),
        ElevatedButton(
          onPressed: loading ? null : (step == _Step.phone ? sendOTP : verifyOTP),
          child: loading
            ? const SizedBox(width: 24, height: 24,
                child: CircularProgressIndicator(color: Colors.white, strokeWidth: 2))
            : Text(step == _Step.phone ? 'Send OTP →' : 'Verify & Enter →')),
        if (step == _Step.otp)
          TextButton(
            onPressed: () => setState(() { step = _Step.phone; oc.clear(); error = ''; }),
            child: const Text('← Change number', style: TextStyle(color: NexusColors.textSecondary))),
      ]),
    )),
  );
}