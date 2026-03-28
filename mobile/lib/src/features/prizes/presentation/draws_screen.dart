import 'dart:async';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../../core/api/api_client.dart';
import '../../../core/theme/nexus_theme.dart';

// ─── Providers ────────────────────────────────────────────────────────────────

final _drawsProvider = FutureProvider.autoDispose<List<dynamic>>((ref) async {
  return ref.read(drawsApiProvider).getDraws();
});

final _winnersProvider = FutureProvider.autoDispose.family<List<dynamic>, String>((ref, drawId) async {
  return ref.read(drawsApiProvider).getWinners(drawId);
});

// ─── Helpers ─────────────────────────────────────────────────────────────────

String _prizeLabel(Map draw) {
  final kobo = (draw['prize_pool_kobo'] as num?)?.toInt() ??
      ((draw['prize_pool'] as num?)?.toInt() ?? 0) * 100;
  if (kobo <= 0) return 'Prize TBD';
  final n = kobo / 100;
  if (n >= 1000000) return '₦${(n / 1000000).toStringAsFixed(1)}M';
  if (n >= 1000) return '₦${(n / 1000).toStringAsFixed(0)}K';
  return '₦${n.toStringAsFixed(0)}';
}

String _maskPhone(String? phone) {
  if (phone == null || phone.length < 8) return '****';
  return '${phone.substring(0, 4)}****${phone.substring(phone.length - 3)}';
}

String _formatShortDate(String iso) {
  try {
    final d = DateTime.parse(iso).toLocal();
    const months = ['Jan','Feb','Mar','Apr','May','Jun',
                    'Jul','Aug','Sep','Oct','Nov','Dec'];
    return '${months[d.month - 1]} ${d.day}';
  } catch (_) { return iso; }
}

String _timeRemaining(String? iso) {
  if (iso == null) return '';
  try {
    final diff = DateTime.parse(iso).difference(DateTime.now());
    if (diff.isNegative) return 'Ended';
    final d = diff.inDays;
    final h = diff.inHours % 24;
    final m = diff.inMinutes % 60;
    if (d > 0) return '${d}d ${h}h ${m}m';
    if (h > 0) return '${h}h ${m}m';
    return '${m}m';
  } catch (_) { return ''; }
}

bool _isEnded(Map draw) {
  final s = (draw['status'] as String? ?? '').toLowerCase();
  return s == 'completed' || s == 'ended';
}

// ─── Main Screen ──────────────────────────────────────────────────────────────

class DrawsScreen extends ConsumerWidget {
  const DrawsScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final drawsAsync = ref.watch(_drawsProvider);

    return Scaffold(
      backgroundColor: NexusColors.background,
      appBar: AppBar(
        backgroundColor: NexusColors.surface,
        title: const Text('Draws & Winners'),
        actions: [
          IconButton(
            icon: const Icon(Icons.refresh_rounded),
            onPressed: () => ref.invalidate(_drawsProvider),
          ),
        ],
      ),
      body: drawsAsync.when(
        loading: () => _DrawsShimmer(),
        error: (e, _) => Center(child: Column(
          mainAxisAlignment: MainAxisAlignment.center, children: [
            const Icon(Icons.wifi_off_rounded, size: 52, color: NexusColors.textSecondary),
            const SizedBox(height: 12),
            const Text('Could not load draws',
                style: TextStyle(color: NexusColors.textSecondary)),
            const SizedBox(height: 12),
            ElevatedButton(
              onPressed: () => ref.invalidate(_drawsProvider),
              child: const Text('Retry'),
            ),
          ],
        )),
        data: (draws) {
          if (draws.isEmpty) {
            return const Center(child: Column(
              mainAxisAlignment: MainAxisAlignment.center, children: [
                Text('🎁', style: TextStyle(fontSize: 52)),
                SizedBox(height: 16),
                Text('No draws available yet',
                    style: TextStyle(color: NexusColors.textSecondary, fontSize: 16)),
                SizedBox(height: 8),
                Text('Keep spinning to earn entries!',
                    style: TextStyle(color: NexusColors.textSecondary, fontSize: 13)),
              ],
            ));
          }

          // Separate active vs completed
          final active    = draws.where((d) => !_isEnded(d as Map)).toList();
          final completed = draws.where((d) => _isEnded(d as Map)).toList();

          return RefreshIndicator(
            color: NexusColors.gold,
            onRefresh: () async => ref.invalidate(_drawsProvider),
            child: ListView(
              padding: const EdgeInsets.all(16),
              children: [
                if (active.isNotEmpty) ...[
                  _SectionHeader(title: 'Upcoming Draws', count: active.length),
                  const SizedBox(height: 8),
                  ...active.map((d) => Padding(
                    padding: const EdgeInsets.only(bottom: 12),
                    child: _DrawCard(draw: d as Map<String, dynamic>),
                  )),
                ],
                if (completed.isNotEmpty) ...[
                  const SizedBox(height: 8),
                  _SectionHeader(title: 'Past Draws', count: completed.length),
                  const SizedBox(height: 8),
                  ...completed.map((d) => Padding(
                    padding: const EdgeInsets.only(bottom: 12),
                    child: _DrawCard(draw: d as Map<String, dynamic>),
                  )),
                ],
              ],
            ),
          );
        },
      ),
    );
  }
}

// ─── Section Header ──────────────────────────────────────────────────────────

class _SectionHeader extends StatelessWidget {
  final String title;
  final int count;
  const _SectionHeader({required this.title, required this.count});

  @override
  Widget build(BuildContext context) => Row(children: [
    Text(title.toUpperCase(),
        style: const TextStyle(color: NexusColors.textSecondary, fontSize: 11,
            fontWeight: FontWeight.w700, letterSpacing: 0.8)),
    const SizedBox(width: 8),
    Container(
      padding: const EdgeInsets.symmetric(horizontal: 7, vertical: 2),
      decoration: BoxDecoration(
        color: Colors.white.withOpacity(0.08),
        borderRadius: BorderRadius.circular(999),
      ),
      child: Text('$count', style: const TextStyle(
          color: NexusColors.textSecondary, fontSize: 10, fontWeight: FontWeight.w700)),
    ),
  ]);
}

// ─── Draw Card ────────────────────────────────────────────────────────────────

class _DrawCard extends ConsumerStatefulWidget {
  final Map<String, dynamic> draw;
  const _DrawCard({required this.draw});
  @override ConsumerState<_DrawCard> createState() => _DrawCardState();
}

class _DrawCardState extends ConsumerState<_DrawCard> {
  bool _expanded = false;
  bool _winnersLoaded = false;
  Timer? _timer;
  String _timeLeft = '';

  bool get _ended => _isEnded(widget.draw);

  @override
  void initState() {
    super.initState();
    _timeLeft = _timeRemaining(widget.draw['draw_date'] as String?);
    if (!_ended) {
      _timer = Timer.periodic(const Duration(minutes: 1), (_) {
        if (mounted) setState(() {
          _timeLeft = _timeRemaining(widget.draw['draw_date'] as String?);
        });
      });
    }
  }

  @override
  void dispose() { _timer?.cancel(); super.dispose(); }

  void _toggleWinners() {
    setState(() {
      _expanded = !_expanded;
      if (_expanded && !_winnersLoaded) _winnersLoaded = true;
    });
  }

  @override
  Widget build(BuildContext context) {
    final draw      = widget.draw;
    final name      = draw['name'] as String? ?? 'Monthly Draw';
    final desc      = draw['description'] as String?;
    final status    = (draw['status'] as String? ?? 'SCHEDULED').toUpperCase();
    final recur     = (draw['recurrence'] as String? ?? '').toUpperCase();
    final drawDate  = draw['draw_date'] as String? ?? '';
    final entries   = draw['entry_count'] as int? ?? 0;
    final prize     = _prizeLabel(draw);
    final drawId    = draw['id'] as String;

    return Container(
      decoration: BoxDecoration(
        color: NexusColors.surface,
        borderRadius: BorderRadius.circular(18),
        border: Border.all(color: Colors.white.withOpacity(_ended ? 0.04 : 0.08)),
      ),
      clipBehavior: Clip.hardEdge,
      child: Column(children: [
        // ── Gradient top strip ──
        Container(
          height: 3,
          decoration: BoxDecoration(
            gradient: _ended
                ? null
                : const LinearGradient(colors: [Color(0xFF4A56EE), Color(0xFFF5A623), Color(0xFF4A56EE)]),
            color: _ended ? Colors.white10 : null,
          ),
        ),

        // ── Header ──
        Padding(
          padding: const EdgeInsets.fromLTRB(16, 14, 16, 0),
          child: Row(crossAxisAlignment: CrossAxisAlignment.start, children: [
            Expanded(child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
              // Title + badges
              Row(children: [
                Flexible(child: Text(name, style: const TextStyle(
                    color: NexusColors.textPrimary, fontSize: 16, fontWeight: FontWeight.w800))),
                const SizedBox(width: 8),
                if (recur.isNotEmpty)
                  _Chip(label: recur, color: NexusColors.primary.withOpacity(0.2),
                      textColor: NexusColors.primary),
              ]),
              if (desc != null) ...[
                const SizedBox(height: 3),
                Text(desc, style: const TextStyle(color: NexusColors.textSecondary, fontSize: 12),
                    maxLines: 1, overflow: TextOverflow.ellipsis),
              ],
            ])),
            const SizedBox(width: 12),
            // Prize + status
            Column(crossAxisAlignment: CrossAxisAlignment.end, children: [
              Text(prize, style: const TextStyle(
                  color: NexusColors.gold, fontSize: 20, fontWeight: FontWeight.w800)),
              const Text('Prize Pool', style: TextStyle(color: NexusColors.textSecondary, fontSize: 10)),
              const SizedBox(height: 4),
              _StatusBadge(status: status),
            ]),
          ]),
        ),

        const SizedBox(height: 12),

        // ── Stats row ──
        Padding(
          padding: const EdgeInsets.symmetric(horizontal: 16),
          child: Row(children: [
            _StatTile(icon: Icons.calendar_today_rounded,
                label: 'Draw Date', value: _formatShortDate(drawDate)),
            const SizedBox(width: 8),
            _StatTile(icon: Icons.confirmation_number_rounded,
                label: 'Entries', value: '$entries'),
            if (!_ended && _timeLeft.isNotEmpty) ...[
              const SizedBox(width: 8),
              _StatTile(icon: Icons.access_time_rounded,
                  label: 'Time Left', value: _timeLeft,
                  valueColor: const Color(0xFFF59E0B)),
            ],
          ]),
        ),

        const SizedBox(height: 14),

        // ── Winners toggle ──
        if (_ended) ...[
          GestureDetector(
            onTap: _toggleWinners,
            child: Container(
              width: double.infinity,
              padding: const EdgeInsets.symmetric(vertical: 11),
              decoration: BoxDecoration(
                color: Colors.white.withOpacity(0.04),
                border: const Border(top: BorderSide(color: NexusColors.border)),
              ),
              child: Row(mainAxisAlignment: MainAxisAlignment.center, children: [
                Icon(Icons.emoji_events_rounded, size: 15,
                    color: _expanded ? NexusColors.gold : NexusColors.textSecondary),
                const SizedBox(width: 6),
                Text(_expanded ? 'Hide Winners' : 'Show Winners',
                    style: TextStyle(
                      color: _expanded ? NexusColors.gold : NexusColors.textSecondary,
                      fontSize: 12, fontWeight: FontWeight.w700,
                    )),
                const SizedBox(width: 4),
                Icon(_expanded ? Icons.expand_less : Icons.expand_more,
                    size: 16,
                    color: _expanded ? NexusColors.gold : NexusColors.textSecondary),
              ]),
            ),
          ),

          // Winners list
          if (_expanded && _winnersLoaded)
            _WinnersList(drawId: drawId),
        ] else ...[
          Container(
            width: double.infinity,
            padding: const EdgeInsets.symmetric(vertical: 10),
            decoration: const BoxDecoration(
              border: Border(top: BorderSide(color: NexusColors.border)),
            ),
            child: Center(child: Row(mainAxisSize: MainAxisSize.min, children: [
              const Icon(Icons.bolt_rounded, size: 13, color: NexusColors.primary),
              const SizedBox(width: 5),
              Text('Spin the wheel to earn entries',
                  style: TextStyle(color: NexusColors.primary.withOpacity(0.7),
                      fontSize: 11, fontWeight: FontWeight.w600)),
            ])),
          ),
        ],
      ]),
    );
  }
}

// ─── Winners list ─────────────────────────────────────────────────────────────

class _WinnersList extends ConsumerWidget {
  final String drawId;
  const _WinnersList({required this.drawId});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final winnersAsync = ref.watch(_winnersProvider(drawId));
    return winnersAsync.when(
      loading: () => const Padding(
        padding: EdgeInsets.symmetric(vertical: 16),
        child: Center(child: NexusShimmer(width: double.infinity, height: 44, radius: NexusRadius.sm)),
      ),
      error: (_, __) => const Padding(
        padding: EdgeInsets.all(12),
        child: Text('Failed to load winners', style: TextStyle(color: NexusColors.textSecondary, fontSize: 12)),
      ),
      data: (winners) {
        if (winners.isEmpty) {
          return const Padding(
            padding: EdgeInsets.symmetric(vertical: 16),
            child: Center(child: Text('No winners announced yet',
                style: TextStyle(color: NexusColors.textSecondary, fontSize: 12))),
          );
        }
        return Column(children: [
          for (int i = 0; i < winners.length; i++) ...[
            const Divider(color: NexusColors.border, height: 1, indent: 16, endIndent: 16),
            _WinnerRow(winner: winners[i] as Map, rank: i + 1),
          ],
        ]);
      },
    );
  }
}

class _WinnerRow extends StatelessWidget {
  final Map winner;
  final int rank;
  const _WinnerRow({required this.winner, required this.rank});

  String get _emoji {
    switch (rank) {
      case 1: return '🥇';
      case 2: return '🥈';
      case 3: return '🥉';
      default: return '🎟️';
    }
  }

  @override
  Widget build(BuildContext context) {
    final phone  = _maskPhone(winner['phone_number'] as String?);
    final label  = winner['prize_label'] as String? ?? '–';
    final rankN  = winner['rank'] as int? ?? rank;

    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
      child: Row(children: [
        Text(_emoji, style: const TextStyle(fontSize: 18)),
        const SizedBox(width: 10),
        Expanded(child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
          Text('Rank #$rankN — $phone',
              style: const TextStyle(color: NexusColors.textPrimary, fontSize: 12,
                  fontWeight: FontWeight.w700)),
          Text(label, style: const TextStyle(color: NexusColors.textSecondary, fontSize: 11)),
        ])),
      ]),
    );
  }
}

// ─── Small widgets ────────────────────────────────────────────────────────────

class _Chip extends StatelessWidget {
  final String label;
  final Color color, textColor;
  const _Chip({required this.label, required this.color, required this.textColor});
  @override
  Widget build(BuildContext context) => Container(
    padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
    decoration: BoxDecoration(color: color, borderRadius: BorderRadius.circular(999)),
    child: Text(label, style: TextStyle(color: textColor, fontSize: 9, fontWeight: FontWeight.w800)),
  );
}

class _StatusBadge extends StatelessWidget {
  final String status;
  const _StatusBadge({required this.status});
  @override
  Widget build(BuildContext context) {
    final (Color bg, Color fg) = switch (status) {
      'SCHEDULED'   => (NexusColors.primary.withOpacity(0.15), NexusColors.primary),
      'IN_PROGRESS' => (NexusColors.gold.withOpacity(0.15), NexusColors.gold),
      'COMPLETED'   => (NexusColors.green.withOpacity(0.15), NexusColors.green),
      _             => (Colors.white.withOpacity(0.08), Colors.white38),
    };
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
      decoration: BoxDecoration(color: bg, borderRadius: BorderRadius.circular(999),
          border: Border.all(color: fg.withOpacity(0.4))),
      child: Text(status, style: TextStyle(color: fg, fontSize: 9, fontWeight: FontWeight.w800)),
    );
  }
}

class _StatTile extends StatelessWidget {
  final IconData icon;
  final String label, value;
  final Color? valueColor;
  const _StatTile({required this.icon, required this.label, required this.value, this.valueColor});
  @override
  Widget build(BuildContext context) => Expanded(child: Container(
    padding: const EdgeInsets.symmetric(vertical: 8, horizontal: 10),
    decoration: BoxDecoration(
      color: Colors.white.withOpacity(0.04),
      borderRadius: BorderRadius.circular(12),
    ),
    child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
      Row(children: [
        Icon(icon, size: 11, color: NexusColors.textSecondary),
        const SizedBox(width: 4),
        Text(label, style: const TextStyle(color: NexusColors.textSecondary, fontSize: 10)),
      ]),
      const SizedBox(height: 3),
      Text(value, style: TextStyle(
          color: valueColor ?? NexusColors.textPrimary,
          fontSize: 12, fontWeight: FontWeight.w800)),
    ]),
  ));
}

// ── Draws shimmer ─────────────────────────────────────────────────────────────
class _DrawsShimmer extends StatelessWidget {
  @override
  Widget build(BuildContext context) => ListView.builder(
    padding: const EdgeInsets.fromLTRB(16, 16, 16, 100),
    itemCount: 4,
    itemBuilder: (_, __) => Padding(
      padding: const EdgeInsets.only(bottom: 12),
      child: NexusShimmer(width: double.infinity, height: 140, radius: NexusRadius.md),
    ),
  );
}
