import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import '../../../core/api/api_client.dart';
import '../../../core/auth/auth_provider.dart';
import '../../../core/theme/nexus_theme.dart';

class DashboardScreen extends ConsumerStatefulWidget {
  const DashboardScreen({super.key});
  @override ConsumerState<DashboardScreen> createState() => _State();
}

class _State extends ConsumerState<DashboardScreen> {
  Map<String, dynamic>? wallet;
  Map<String, dynamic>? profile;
  bool loading = true;

  @override
  void initState() { super.initState(); _load(); }

  Future<void> _load() async {
    try {
      final dio = ref.read(dioProvider);
      final w = await dio.apiGet<Map<String, dynamic>>('/user/wallet');
      final p = await dio.apiGet<Map<String, dynamic>>('/user/profile');
      setState(() { wallet = w; profile = p; loading = false; });
    } catch (_) { setState(() => loading = false); }
  }

  @override
  Widget build(BuildContext ctx) {
    final pts = wallet?['pulse_points'] ?? 0;
    final spins = wallet?['spin_credits'] ?? 0;
    final tier = profile?['tier'] ?? 'BRONZE';
    return Scaffold(
      appBar: AppBar(
        title: const Text('Home'),
        actions: [
          IconButton(icon: const Icon(Icons.settings_outlined),
            onPressed: () => ctx.go('/settings')),
        ],
      ),
      body: loading
        ? const Center(child: CircularProgressIndicator(color: NexusColors.primary))
        : RefreshIndicator(
            onRefresh: _load,
            color: NexusColors.primary,
            child: SingleChildScrollView(
              physics: const AlwaysScrollableScrollPhysics(),
              padding: const EdgeInsets.all(20),
              child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                // Wallet card
                Container(
                  width: double.infinity,
                  padding: const EdgeInsets.all(20),
                  decoration: BoxDecoration(
                    gradient: const LinearGradient(
                      colors: [Color(0xFF4A56EE), Color(0xFF8B5CF6)],
                      begin: Alignment.topLeft, end: Alignment.bottomRight),
                    borderRadius: BorderRadius.circular(20)),
                  child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                    Row(mainAxisAlignment: MainAxisAlignment.spaceBetween, children: [
                      const Text('Pulse Points', style: TextStyle(color: Colors.white70, fontSize: 12)),
                      Container(
                        padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
                        decoration: BoxDecoration(
                          color: Colors.white.withOpacity(0.2),
                          borderRadius: BorderRadius.circular(20)),
                        child: Text(tier, style: const TextStyle(
                          color: Colors.white, fontSize: 11, fontWeight: FontWeight.bold))),
                    ]),
                    const SizedBox(height: 8),
                    Text('\$pts pts', style: const TextStyle(
                      color: Colors.white, fontSize: 36, fontWeight: FontWeight.bold,
                      fontFamily: 'Syne')),
                    const SizedBox(height: 16),
                    Row(children: [
                      Expanded(child: _statBox('Spin Credits', '\$spins')),
                      const SizedBox(width: 12),
                      Expanded(child: _statBox('Lifetime', '\${wallet?['lifetime_points'] ?? 0} pts')),
                    ]),
                  ])),
                const SizedBox(height: 24),
                Text('Quick Actions', style: Theme.of(ctx).textTheme.titleLarge),
                const SizedBox(height: 12),
                GridView.count(
                  crossAxisCount: 2, shrinkWrap: true,
                  physics: const NeverScrollableScrollPhysics(),
                  crossAxisSpacing: 12, mainAxisSpacing: 12, childAspectRatio: 1.6,
                  children: [
                    _actionCard(ctx, '🎡', 'Spin & Win', 'Use credits', '/spin', NexusColors.primary),
                    _actionCard(ctx, '🧠', 'AI Studio', '17 free tools', '/studio', const Color(0xFF8B5CF6)),
                    _actionCard(ctx, '🌍', 'Regional Wars', 'State rank', '/wars', NexusColors.green),
                    _actionCard(ctx, '🎁', 'My Prizes', 'Claim rewards', '/prizes', NexusColors.gold),
                  ],
                ),
                const SizedBox(height: 24),
                Container(
                  padding: const EdgeInsets.all(16),
                  decoration: BoxDecoration(
                    color: NexusColors.surface,
                    borderRadius: BorderRadius.circular(16),
                    border: Border.all(color: NexusColors.border)),
                  child: Row(children: [
                    const Icon(Icons.bolt, color: NexusColors.primary),
                    const SizedBox(width: 12),
                    Expanded(child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                      const Text('Recharge to earn more', style: TextStyle(
                        color: NexusColors.textPrimary, fontWeight: FontWeight.w600)),
                      const Text('₦200+ → 2 points + 1 spin credit',
                        style: TextStyle(color: NexusColors.textSecondary, fontSize: 12)),
                    ])),
                  ]),
                ),
              ]),
            ),
          ),
    );
  }

  Widget _statBox(String label, String value) => Container(
    padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
    decoration: BoxDecoration(
      color: Colors.white.withOpacity(0.15),
      borderRadius: BorderRadius.circular(10)),
    child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
      Text(label, style: const TextStyle(color: Colors.white70, fontSize: 10)),
      Text(value, style: const TextStyle(
        color: Colors.white, fontWeight: FontWeight.bold, fontSize: 14)),
    ]));

  Widget _actionCard(BuildContext ctx, String icon, String title, String sub,
      String route, Color color) => GestureDetector(
    onTap: () => ctx.go(route),
    child: Container(
      padding: const EdgeInsets.all(14),
      decoration: BoxDecoration(
        color: NexusColors.surface,
        borderRadius: BorderRadius.circular(16),
        border: Border.all(color: NexusColors.border)),
      child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
        Text(icon, style: const TextStyle(fontSize: 22)),
        const SizedBox(height: 4),
        Text(title, style: TextStyle(
          color: NexusColors.textPrimary, fontWeight: FontWeight.w600, fontSize: 13)),
        Text(sub, style: const TextStyle(color: NexusColors.textSecondary, fontSize: 11)),
      ])));
}