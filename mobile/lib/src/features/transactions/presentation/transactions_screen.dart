import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../../core/api/api_client.dart';
import '../../../core/theme/nexus_theme.dart';

// ─── Providers ────────────────────────────────────────────────────────────────

final _transactionsProvider =
    FutureProvider.autoDispose<Map<String, dynamic>>((ref) async {
  final r = await ref.read(userApiProvider).getTransactions(page: 1, limit: 50);
  return {'transactions': r};
});

final _rechargesProvider =
    FutureProvider.autoDispose<Map<String, dynamic>>((ref) async {
  return ref.read(userApiProvider).getRecharges(page: 1, limit: 50);
});

// ─── Screen ───────────────────────────────────────────────────────────────────

class TransactionsScreen extends ConsumerStatefulWidget {
  const TransactionsScreen({super.key});
  @override
  ConsumerState<TransactionsScreen> createState() => _TransactionsScreenState();
}

class _TransactionsScreenState extends ConsumerState<TransactionsScreen>
    with SingleTickerProviderStateMixin {
  late final TabController _tab;

  @override
  void initState() {
    super.initState();
    _tab = TabController(length: 2, vsync: this);
    _tab.addListener(() => setState(() {}));
  }

  @override
  void dispose() {
    _tab.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: NexusColors.background,
      appBar: AppBar(
        backgroundColor: NexusColors.background,
        surfaceTintColor: Colors.transparent,
        title: const Text('Transaction History'),
        centerTitle: false,
        bottom: TabBar(
          controller: _tab,
          indicatorColor: NexusColors.primary,
          labelColor: NexusColors.primary,
          unselectedLabelColor: NexusColors.textSecondary,
          labelStyle: const TextStyle(fontSize: 13, fontWeight: FontWeight.w700),
          tabs: const [
            Tab(text: 'Points Activity'),
            Tab(text: 'Recharges'),
          ],
        ),
      ),
      body: TabBarView(
        controller: _tab,
        children: const [
          _TransactionsTab(),
          _RechargesTab(),
        ],
      ),
    );
  }
}

// ─── Transactions Tab ─────────────────────────────────────────────────────────

class _TransactionsTab extends ConsumerWidget {
  const _TransactionsTab();

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final async = ref.watch(_transactionsProvider);
    return async.when(
      loading: () => const _ListSkeleton(),
      error: (e, _) => _ErrorState(onRetry: () => ref.invalidate(_transactionsProvider)),
      data: (data) {
        final txns = (data['transactions'] as List?) ?? [];
        if (txns.isEmpty) return const _EmptyState(label: 'No transactions yet');
        return RefreshIndicator(
          color: NexusColors.primary,
          onRefresh: () async => ref.invalidate(_transactionsProvider),
          child: ListView.separated(
            padding: const EdgeInsets.fromLTRB(16, 12, 16, 100),
            itemCount: txns.length,
            separatorBuilder: (_, __) => const SizedBox(height: 8),
            itemBuilder: (_, i) => _TxTile(tx: txns[i] as Map),
          ),
        );
      },
    );
  }
}

// ─── Recharges Tab ────────────────────────────────────────────────────────────

class _RechargesTab extends ConsumerWidget {
  const _RechargesTab();

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final async = ref.watch(_rechargesProvider);
    return async.when(
      loading: () => const _ListSkeleton(),
      error: (e, _) => _ErrorState(onRetry: () => ref.invalidate(_rechargesProvider)),
      data: (data) {
        final recharges = (data['recharges'] ?? data['data'] ?? data['results'] ?? []) as List;
        if (recharges.isEmpty) return const _EmptyState(label: 'No recharges yet');
        return RefreshIndicator(
          color: NexusColors.primary,
          onRefresh: () async => ref.invalidate(_rechargesProvider),
          child: ListView.separated(
            padding: const EdgeInsets.fromLTRB(16, 12, 16, 100),
            itemCount: recharges.length,
            separatorBuilder: (_, __) => const SizedBox(height: 8),
            itemBuilder: (_, i) => _RechargeTile(r: recharges[i] as Map),
          ),
        );
      },
    );
  }
}

// ─── Transaction tile ─────────────────────────────────────────────────────────

class _TxTile extends StatelessWidget {
  final Map tx;
  const _TxTile({required this.tx});

  @override
  Widget build(BuildContext context) {
    final type    = tx['transaction_type']?.toString() ?? '';
    final pts     = tx['points'] as int? ?? 0;
    final isEarn  = pts > 0;
    final date    = _fmt(tx['created_at']?.toString() ?? '');
    final ref     = tx['reference']?.toString();

    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 12),
      decoration: BoxDecoration(
        color: NexusColors.surface,
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: NexusColors.border),
      ),
      child: Row(children: [
        // Icon
        Container(
          width: 38, height: 38,
          decoration: BoxDecoration(
            color: isEarn
                ? NexusColors.primary.withValues(alpha: 0.12)
                : NexusColors.red.withValues(alpha: 0.12),
            borderRadius: BorderRadius.circular(10),
          ),
          child: Icon(
            isEarn ? Icons.add_rounded : Icons.remove_rounded,
            color: isEarn ? NexusColors.primary : NexusColors.red,
            size: 18,
          ),
        ),
        const SizedBox(width: 12),
        // Details
        Expanded(child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
          Text(
            _label(type),
            style: const TextStyle(
              color: NexusColors.textPrimary, fontSize: 13, fontWeight: FontWeight.w600),
          ),
          if (ref != null) ...[
            const SizedBox(height: 2),
            Text(ref, style: const TextStyle(color: NexusColors.textSecondary, fontSize: 10)),
          ],
          const SizedBox(height: 2),
          Text(date, style: const TextStyle(color: NexusColors.textSecondary, fontSize: 11)),
        ])),
        // Points
        Text(
          '${isEarn ? "+" : ""}$pts pts',
          style: TextStyle(
            color: isEarn ? NexusColors.primary : NexusColors.red,
            fontSize: 14,
            fontWeight: FontWeight.w800,
          ),
        ),
      ]),
    );
  }

  String _label(String type) => switch (type) {
    'recharge'      => 'Recharge Bonus',
    'spin_win'      => 'Spin Win',
    'bonus'         => 'Bonus Award',
    'referral'      => 'Referral Bonus',
    'deduction'     => 'Points Used',
    'birthday'      => 'Birthday Bonus',
    _               => type.replaceAll('_', ' ').split(' ')
                           .map((w) => w.isNotEmpty ? '${w[0].toUpperCase()}${w.substring(1)}' : '')
                           .join(' '),
  };
}

// ─── Recharge tile ────────────────────────────────────────────────────────────

class _RechargeTile extends StatelessWidget {
  final Map r;
  const _RechargeTile({required this.r});

  static const _netColors = {
    'MTN': Color(0xFFFFCC00),
    'AIRTEL': Color(0xFFE40000),
    'GLO': Color(0xFF008000),
    '9MOBILE': Color(0xFF006400),
  };

  @override
  Widget build(BuildContext context) {
    final phone     = r['phone_number']?.toString() ?? r['beneficiary']?.toString() ?? '—';
    final network   = (r['network'] ?? r['network_provider'] ?? '').toString().toUpperCase();
    final amount    = r['amount'] as int? ?? 0;
    final pts       = r['points_earned'] as int? ?? 0;
    final status    = (r['status'] ?? 'unknown').toString().toLowerCase();
    final date      = _fmt(r['created_at']?.toString() ?? '');
    final netColor  = _netColors[network] ?? NexusColors.primary;

    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 12),
      decoration: BoxDecoration(
        color: NexusColors.surface,
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: NexusColors.border),
      ),
      child: Row(children: [
        // Network badge
        Container(
          width: 38, height: 38,
          decoration: BoxDecoration(
            color: netColor.withValues(alpha: 0.12),
            borderRadius: BorderRadius.circular(10),
          ),
          child: Center(
            child: Text(
              network.isEmpty ? '📶' : network[0],
              style: TextStyle(color: netColor, fontSize: 14, fontWeight: FontWeight.w800),
            ),
          ),
        ),
        const SizedBox(width: 12),
        // Details
        Expanded(child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
          Text(
            phone,
            style: const TextStyle(
              color: NexusColors.textPrimary, fontSize: 13, fontWeight: FontWeight.w600),
          ),
          const SizedBox(height: 2),
          Row(children: [
            if (network.isNotEmpty) ...[
              Text(network, style: TextStyle(color: netColor, fontSize: 10, fontWeight: FontWeight.w700)),
              const Text(' · ', style: TextStyle(color: NexusColors.textSecondary, fontSize: 10)),
            ],
            Text(date, style: const TextStyle(color: NexusColors.textSecondary, fontSize: 10)),
          ]),
        ])),
        // Right side
        Column(crossAxisAlignment: CrossAxisAlignment.end, children: [
          Text(
            '₦${(amount / 100).toStringAsFixed(0)}',
            style: const TextStyle(
              color: NexusColors.textPrimary, fontSize: 14, fontWeight: FontWeight.w800),
          ),
          if (pts > 0)
            Text(
              '+$pts pts',
              style: const TextStyle(color: NexusColors.primary, fontSize: 11, fontWeight: FontWeight.w600),
            ),
          const SizedBox(height: 2),
          _StatusChip(status: status),
        ]),
      ]),
    );
  }
}

class _StatusChip extends StatelessWidget {
  final String status;
  const _StatusChip({required this.status});

  @override
  Widget build(BuildContext context) {
    final (color, label) = switch (status) {
      'successful' || 'success' || 'completed' => (NexusColors.green, 'Success'),
      'failed' || 'error'                       => (NexusColors.red,   'Failed'),
      'pending'                                 => (NexusColors.gold,  'Pending'),
      _                                         => (NexusColors.textSecondary, status),
    };
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.12),
        borderRadius: BorderRadius.circular(4),
      ),
      child: Text(label, style: TextStyle(color: color, fontSize: 9, fontWeight: FontWeight.w700)),
    );
  }
}

// ─── Shared widgets ───────────────────────────────────────────────────────────

class _ListSkeleton extends StatelessWidget {
  const _ListSkeleton();
  @override
  Widget build(BuildContext context) => ListView.separated(
    padding: const EdgeInsets.fromLTRB(16, 12, 16, 40),
    itemCount: 6,
    separatorBuilder: (_, __) => const SizedBox(height: 8),
    itemBuilder: (_, __) => Container(
      height: 64,
      decoration: BoxDecoration(
        color: NexusColors.surface,
        borderRadius: BorderRadius.circular(12),
      ),
    ),
  );
}

class _EmptyState extends StatelessWidget {
  final String label;
  const _EmptyState({required this.label});
  @override
  Widget build(BuildContext context) => Center(
    child: Column(mainAxisSize: MainAxisSize.min, children: [
      const Text('📋', style: TextStyle(fontSize: 48)),
      const SizedBox(height: 12),
      Text(label, style: const TextStyle(color: NexusColors.textSecondary, fontSize: 14)),
    ]),
  );
}

class _ErrorState extends StatelessWidget {
  final VoidCallback onRetry;
  const _ErrorState({required this.onRetry});
  @override
  Widget build(BuildContext context) => Center(
    child: Column(mainAxisSize: MainAxisSize.min, children: [
      const Text('😕', style: TextStyle(fontSize: 48)),
      const SizedBox(height: 12),
      const Text('Could not load data', style: TextStyle(color: NexusColors.textSecondary)),
      const SizedBox(height: 12),
      ElevatedButton(onPressed: onRetry, child: const Text('Retry')),
    ]),
  );
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

String _fmt(String iso) {
  if (iso.isEmpty) return '';
  try {
    final dt = DateTime.parse(iso).toLocal();
    final months = ['Jan','Feb','Mar','Apr','May','Jun','Jul','Aug','Sep','Oct','Nov','Dec'];
    final h = dt.hour.toString().padLeft(2, '0');
    final m = dt.minute.toString().padLeft(2, '0');
    return '${dt.day} ${months[dt.month - 1]} ${dt.year}, $h:$m';
  } catch (_) { return iso; }
}
