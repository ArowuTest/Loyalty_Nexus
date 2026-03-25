import 'package:flutter/material.dart';
import '../../../core/theme/nexus_theme.dart';

const _lb = [
  (rank: 1, state: 'Lagos',  pts: 48750, medal: '🥇'),
  (rank: 2, state: 'Abuja',  pts: 41200, medal: '🥈'),
  (rank: 3, state: 'Rivers', pts: 38600, medal: '🥉'),
  (rank: 4, state: 'Kano',   pts: 35100, medal: '4'),
  (rank: 5, state: 'Oyo',    pts: 31500, medal: '5'),
];

class WarsScreen extends StatelessWidget {
  const WarsScreen({super.key});
  @override
  Widget build(BuildContext ctx) => Scaffold(
    appBar: AppBar(title: const Text('Regional Wars 🌍')),
    body: Padding(
      padding: const EdgeInsets.all(20),
      child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
        Container(
          width: double.infinity, padding: const EdgeInsets.all(20),
          decoration: BoxDecoration(
            gradient: const LinearGradient(colors: [Color(0xFF10B981), Color(0xFF059669)]),
            borderRadius: BorderRadius.circular(16)),
          child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
            const Text('Monthly Prize Pool', style: TextStyle(color: Colors.white70, fontSize: 12)),
            const SizedBox(height: 4),
            const Text('₦500,000', style: TextStyle(
              color: Colors.white, fontSize: 32, fontWeight: FontWeight.bold)),
            const Text('Top 3 states share the pool • 14 days left',
              style: TextStyle(color: Colors.white70, fontSize: 12)),
          ])),
        const SizedBox(height: 24),
        Text('State Leaderboard', style: Theme.of(ctx).textTheme.titleLarge),
        const SizedBox(height: 12),
        Expanded(child: ListView.separated(
          itemCount: _lb.length,
          separatorBuilder: (_, __) => const SizedBox(height: 8),
          itemBuilder: (_, i) {
            final row = _lb[i];
            return Container(
              padding: const EdgeInsets.all(14),
              decoration: BoxDecoration(
                color: NexusColors.surface,
                borderRadius: BorderRadius.circular(12),
                border: Border.all(color: row.rank <= 3
                  ? NexusColors.gold.withOpacity(0.3) : NexusColors.border)),
              child: Row(children: [
                Text(row.medal, style: const TextStyle(fontSize: 20)),
                const SizedBox(width: 12),
                Expanded(child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                  Text(row.state, style: const TextStyle(
                    color: NexusColors.textPrimary, fontWeight: FontWeight.w600)),
                ])),
                Text('\${row.pts.toString().replaceAllMapped(RegExp(r"(\d)(?=(\d{3})+\$)"), (m) => "\${m[1]},")} pts',
                  style: const TextStyle(color: NexusColors.textPrimary, fontWeight: FontWeight.bold)),
              ]));
          })),
      ]),
    ));
}