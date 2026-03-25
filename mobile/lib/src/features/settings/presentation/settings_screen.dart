import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import '../../../core/auth/auth_provider.dart';
import '../../../core/theme/nexus_theme.dart';

class SettingsScreen extends ConsumerWidget {
  const SettingsScreen({super.key});
  @override
  Widget build(BuildContext ctx, WidgetRef ref) {
    final phone = ref.watch(authStateProvider).phoneNumber ?? '—';
    return Scaffold(
      appBar: AppBar(title: const Text('Settings')),
      body: ListView(padding: const EdgeInsets.all(20), children: [
        _tile(ctx, Icons.phone_android, 'Phone Number', phone, null),
        _tile(ctx, Icons.account_balance_wallet, 'Link MoMo Wallet', 'For cash prizes', null),
        _tile(ctx, Icons.notifications_outlined, 'Notifications', 'SMS & push alerts', null),
        _tile(ctx, Icons.privacy_tip_outlined, 'Privacy Policy', 'How we use your data', null),
        const SizedBox(height: 24),
        OutlinedButton.icon(
          onPressed: () async {
            await ref.read(authStateProvider.notifier).logout();
            if (ctx.mounted) ctx.go('/');
          },
          icon: const Icon(Icons.logout, color: NexusColors.red),
          label: const Text('Sign Out', style: TextStyle(color: NexusColors.red)),
          style: OutlinedButton.styleFrom(
            side: const BorderSide(color: NexusColors.red),
            minimumSize: const Size(double.infinity, 50)),
        ),
      ]),
    );
  }

  Widget _tile(BuildContext ctx, IconData icon, String title, String sub, VoidCallback? onTap) =>
    ListTile(
      contentPadding: const EdgeInsets.symmetric(horizontal: 0, vertical: 4),
      leading: Container(
        width: 40, height: 40,
        decoration: BoxDecoration(
          color: NexusColors.primary.withOpacity(0.1),
          borderRadius: BorderRadius.circular(12)),
        child: Icon(icon, color: NexusColors.primary, size: 20)),
      title: Text(title, style: const TextStyle(
        color: NexusColors.textPrimary, fontWeight: FontWeight.w500, fontSize: 14)),
      subtitle: Text(sub, style: const TextStyle(color: NexusColors.textSecondary, fontSize: 12)),
      trailing: onTap != null ? const Icon(Icons.chevron_right, color: NexusColors.textSecondary) : null,
      onTap: onTap,
    );
}