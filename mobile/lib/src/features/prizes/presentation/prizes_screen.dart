import 'dart:async';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../../core/api/api_client.dart';
import '../../../core/theme/nexus_theme.dart';

// ─── Providers ────────────────────────────────────────────────────────────────

final _winsProvider = FutureProvider.autoDispose<List<dynamic>>((ref) async {
  return ref.read(spinApiProvider).getMyWins();
});

// ─── Enums / helpers ─────────────────────────────────────────────────────────

enum _Tab { all, pending, claimed }

const _prizeIcons = <String, IconData>{
  'airtime':      Icons.phone_android_rounded,
  'data_bundle':  Icons.wifi_rounded,
  'pulse_points': Icons.bolt_rounded,
  'momo_cash':    Icons.account_balance_wallet_rounded,
  'try_again':    Icons.close_rounded,
};

const _prizeColors = <String, Color>{
  'airtime':      Color(0xFF3B82F6),
  'data_bundle':  Color(0xFFA855F7),
  'pulse_points': Color(0xFFF5A623),
  'momo_cash':    Color(0xFF22C55E),
  'try_again':    Color(0xFF6B7280),
};

const _prizeBgColors = <String, Color>{
  'airtime':      Color(0x1A3B82F6),
  'data_bundle':  Color(0x1AA855F7),
  'pulse_points': Color(0x1AF5A623),
  'momo_cash':    Color(0x1A22C55E),
  'try_again':    Color(0x0AFFFFFF),
};

String _timeAgo(String iso) {
  try {
    final d = DateTime.now().difference(DateTime.parse(iso));
    if (d.inHours < 1) return 'just now';
    if (d.inHours < 24) return '${d.inHours}h ago';
    return '${d.inDays}d ago';
  } catch (_) { return ''; }
}

String _expiresIn(String? iso) {
  if (iso == null) return '';
  try {
    final diff = DateTime.parse(iso).difference(DateTime.now());
    if (diff.isNegative) return 'Expired';
    if (diff.inHours < 1) return '< 1h left';
    if (diff.inHours < 24) return '${diff.inHours}h left';
    return '${diff.inDays}d left';
  } catch (_) { return ''; }
}

Map<String, dynamic> _claimBadge(String status) {
  switch (status) {
    case 'PENDING':              return {'label': 'Claim Now', 'color': const Color(0xFFF5A623), 'bg': const Color(0x1AF5A623)};
    case 'PENDING_ADMIN_REVIEW': return {'label': 'Under Review', 'color': const Color(0xFFF59E0B), 'bg': const Color(0x1AF59E0B)};
    case 'APPROVED':             return {'label': 'Approved', 'color': const Color(0xFF22C55E), 'bg': const Color(0x1A22C55E)};
    case 'CLAIMED':              return {'label': 'Claimed ✓', 'color': const Color(0xFF6B7280), 'bg': const Color(0x0AFFFFFF)};
    case 'REJECTED':             return {'label': 'Rejected', 'color': const Color(0xFFEF4444), 'bg': const Color(0x1AEF4444)};
    case 'EXPIRED':              return {'label': 'Expired', 'color': const Color(0xFF6B7280), 'bg': const Color(0x08FFFFFF)};
    default:                     return {'label': status, 'color': const Color(0xFF6B7280), 'bg': const Color(0x08FFFFFF)};
  }
}

String _fulfillLabel(String status) {
  switch (status) {
    case 'completed':          return '✓ Credited';
    case 'pending':            return '⏳ Processing';
    case 'pending_claim':      return '⚡ Awaiting claim';
    case 'pending_momo_setup': return '📱 Need MoMo';
    case 'processing':         return '⚙️ In progress';
    case 'failed':             return '✗ Failed';
    case 'na':                 return '';
    default:                   return status;
  }
}

Color _fulfillColor(String status) {
  switch (status) {
    case 'completed':  return const Color(0xFF22C55E);
    case 'processing': return const Color(0xFF3B82F6);
    case 'pending_momo_setup': return const Color(0xFFF97316);
    case 'failed':     return const Color(0xFFEF4444);
    default:           return const Color(0xFF9CA3AF);
  }
}

// ─── Main Screen ──────────────────────────────────────────────────────────────

class PrizesScreen extends ConsumerStatefulWidget {
  const PrizesScreen({super.key});
  @override ConsumerState<PrizesScreen> createState() => _PrizesScreenState();
}

class _PrizesScreenState extends ConsumerState<PrizesScreen>
    with SingleTickerProviderStateMixin {
  late final TabController _tabs;
  Map<String, dynamic>? _claimWin;

  @override
  void initState() {
    super.initState();
    _tabs = TabController(length: 3, vsync: this);
  }

  @override
  void dispose() { _tabs.dispose(); super.dispose(); }

  List<dynamic> _filter(List<dynamic> all, _Tab tab) {
    final wins = all.where((w) => (w as Map)['prize_type'] != 'try_again').toList();
    switch (tab) {
      case _Tab.pending:
        return wins.where((w) {
          final s = (w as Map)['claim_status'] as String? ?? '';
          return s == 'PENDING' || s == 'PENDING_ADMIN_REVIEW';
        }).toList();
      case _Tab.claimed:
        return wins.where((w) {
          final s = (w as Map)['claim_status'] as String? ?? '';
          return s == 'CLAIMED' || s == 'APPROVED';
        }).toList();
      case _Tab.all:
        return wins;
    }
  }

  int _pendingCount(List<dynamic> all) => all
      .where((w) => (w as Map)['claim_status'] == 'PENDING' && w['prize_type'] != 'try_again')
      .length;

  @override
  Widget build(BuildContext context) {
    final winsAsync = ref.watch(_winsProvider);

    return Scaffold(
      backgroundColor: NexusColors.background,
      appBar: AppBar(
        backgroundColor: NexusColors.surface,
        title: const Text('My Prizes 🏆'),
        bottom: TabBar(
          controller: _tabs,
          indicatorColor: NexusColors.gold,
          labelColor: NexusColors.gold,
          unselectedLabelColor: NexusColors.textSecondary,
          indicatorWeight: 2,
          labelStyle: const TextStyle(fontSize: 13, fontWeight: FontWeight.w700),
          tabs: [
            const Tab(text: 'All'),
            Tab(
              child: winsAsync.whenOrNull(
                data: (all) {
                  final c = _pendingCount(all);
                  return Row(mainAxisSize: MainAxisSize.min, children: [
                    const Text('Pending'),
                    if (c > 0) ...[
                      const SizedBox(width: 6),
                      Container(
                        padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 1),
                        decoration: BoxDecoration(
                          color: NexusColors.gold,
                          borderRadius: BorderRadius.circular(999),
                        ),
                        child: Text('$c',
                            style: const TextStyle(color: Colors.black, fontSize: 10,
                                fontWeight: FontWeight.w800)),
                      ),
                    ],
                  ]);
                },
              ) ?? const Text('Pending'),
            ),
            const Tab(text: 'Claimed'),
          ],
        ),
      ),
      body: winsAsync.when(
        loading: () => const Center(child: CircularProgressIndicator(color: NexusColors.gold)),
        error: (e, _) => _ErrorView(onRetry: () => ref.invalidate(_winsProvider)),
        data: (all) => TabBarView(
          controller: _tabs,
          children: _Tab.values.map((tab) {
            final wins = _filter(all, tab);
            return RefreshIndicator(
              color: NexusColors.gold,
              onRefresh: () async => ref.invalidate(_winsProvider),
              child: wins.isEmpty
                  ? _EmptyWins(tab: tab)
                  : ListView.separated(
                      padding: const EdgeInsets.all(16),
                      itemCount: wins.length + 1,
                      separatorBuilder: (_, __) => const SizedBox(height: 10),
                      itemBuilder: (ctx, i) {
                        if (i == wins.length) return _HowItWorksCard();
                        final win = wins[i] as Map<String, dynamic>;
                        return _WinCard(
                          win: win,
                          onClaim: () => setState(() => _claimWin = win),
                        );
                      },
                    ),
            );
          }).toList(),
        ),
      ),
      // Claim modal overlay
      floatingActionButtonLocation: FloatingActionButtonLocation.centerFloat,
      bottomSheet: _claimWin != null
          ? _ClaimModal(
              win: _claimWin!,
              onClose: () => setState(() => _claimWin = null),
              onSuccess: () {
                ref.invalidate(_winsProvider);
                setState(() => _claimWin = null);
                ScaffoldMessenger.of(context).showSnackBar(const SnackBar(
                  content: Text('Prize claim submitted! 🎉'),
                  backgroundColor: NexusColors.green,
                ));
              },
            )
          : null,
    );
  }
}

// ─── Win Card ─────────────────────────────────────────────────────────────────

class _WinCard extends StatelessWidget {
  final Map<String, dynamic> win;
  final VoidCallback onClaim;
  const _WinCard({required this.win, required this.onClaim});

  bool get _canClaim {
    final cs = win['claim_status'] as String? ?? '';
    final pt = win['prize_type'] as String? ?? '';
    return cs == 'PENDING' && (pt == 'airtime' || pt == 'data_bundle' || pt == 'momo_cash');
  }

  @override
  Widget build(BuildContext context) {
    final prizeType  = win['prize_type'] as String? ?? '';
    final label      = win['prize_label'] as String? ?? prizeType;
    final claimSt    = win['claim_status'] as String? ?? '';
    final fulfillSt  = win['fulfillment_status'] as String? ?? '';
    final createdAt  = win['created_at'] as String? ?? '';
    final expiresAt  = win['expires_at'] as String?;
    final icon       = _prizeIcons[prizeType] ?? Icons.card_giftcard_rounded;
    final iconColor  = _prizeColors[prizeType] ?? Colors.white38;
    final bgColor    = _prizeBgColors[prizeType] ?? const Color(0x0AFFFFFF);
    final badge      = _claimBadge(claimSt);
    final expiry     = _expiresIn(expiresAt);
    final isExpiring = expiry.isNotEmpty && !expiry.contains('d') && expiry != 'Expired';

    return GestureDetector(
      onTap: _canClaim ? onClaim : null,
      child: AnimatedContainer(
        duration: const Duration(milliseconds: 200),
        padding: const EdgeInsets.all(14),
        decoration: BoxDecoration(
          color: NexusColors.surface,
          borderRadius: BorderRadius.circular(16),
          border: Border.all(
            color: _canClaim ? NexusColors.gold.withOpacity(0.3)
                : bgColor,
          ),
        ),
        child: Row(children: [
          // Icon badge
          Container(
            width: 46, height: 46,
            decoration: BoxDecoration(
              color: bgColor,
              borderRadius: BorderRadius.circular(14),
            ),
            child: Icon(icon, color: iconColor, size: 22),
          ),
          const SizedBox(width: 12),

          // Info
          Expanded(child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
            Row(children: [
              Flexible(child: Text(label,
                  style: const TextStyle(color: NexusColors.textPrimary,
                      fontWeight: FontWeight.w700, fontSize: 13),
                  overflow: TextOverflow.ellipsis)),
              const SizedBox(width: 6),
              Container(
                padding: const EdgeInsets.symmetric(horizontal: 7, vertical: 2),
                decoration: BoxDecoration(
                  color: badge['bg'] as Color,
                  borderRadius: BorderRadius.circular(999),
                ),
                child: Text(badge['label'] as String,
                    style: TextStyle(color: badge['color'] as Color,
                        fontSize: 10, fontWeight: FontWeight.w700)),
              ),
            ]),
            const SizedBox(height: 3),
            Row(children: [
              if (fulfillSt.isNotEmpty && fulfillSt != 'na')
                Text(_fulfillLabel(fulfillSt),
                    style: TextStyle(color: _fulfillColor(fulfillSt), fontSize: 10)),
              if (fulfillSt.isNotEmpty && fulfillSt != 'na') const SizedBox(width: 8),
              Text(_timeAgo(createdAt),
                  style: const TextStyle(color: NexusColors.textSecondary, fontSize: 10)),
              if (expiry.isNotEmpty && claimSt == 'PENDING') ...[
                const SizedBox(width: 8),
                Icon(Icons.access_time_rounded, size: 10,
                    color: isExpiring ? const Color(0xFFF59E0B) : NexusColors.textSecondary),
                const SizedBox(width: 2),
                Text(expiry, style: TextStyle(
                  fontSize: 10,
                  color: isExpiring ? const Color(0xFFF59E0B) : NexusColors.textSecondary,
                )),
              ],
            ]),
          ])),

          // CTA
          if (_canClaim) ...[
            const SizedBox(width: 8),
            Row(children: [
              Text('Claim', style: TextStyle(color: NexusColors.gold, fontSize: 12,
                  fontWeight: FontWeight.w700)),
              const Icon(Icons.chevron_right_rounded, color: NexusColors.gold, size: 16),
            ]),
          ] else if (claimSt == 'CLAIMED') ...[
            const Icon(Icons.check_circle_rounded, color: NexusColors.green, size: 20),
          ] else if (claimSt == 'PENDING_ADMIN_REVIEW') ...[
            const Icon(Icons.access_time_rounded, color: Color(0xFFF59E0B), size: 20),
          ],
        ]),
      ),
    );
  }
}

// ─── Claim Modal ──────────────────────────────────────────────────────────────

class _ClaimModal extends ConsumerStatefulWidget {
  final Map<String, dynamic> win;
  final VoidCallback onClose, onSuccess;
  const _ClaimModal({required this.win, required this.onClose, required this.onSuccess});
  @override ConsumerState<_ClaimModal> createState() => _ClaimModalState();
}

class _ClaimModalState extends ConsumerState<_ClaimModal> {
  final _momoCtrl = TextEditingController();
  bool _claiming  = false;
  String? _err;

  bool get _needsMomo => widget.win['prize_type'] == 'momo_cash';
  bool get _canSubmit => !_claiming && (!_needsMomo || _momoCtrl.text.length >= 10);

  Future<void> _claim() async {
    setState(() { _claiming = true; _err = null; });
    try {
      await ref.read(spinApiProvider).claimPrize(
        widget.win['id'] as String,
        _needsMomo ? {'momo_number': _momoCtrl.text.trim()} : {},
      );
      widget.onSuccess();
    } on ApiException catch (e) {
      setState(() { _claiming = false; _err = e.message; });
    } catch (e) {
      setState(() { _claiming = false; _err = e.toString(); });
    }
  }

  @override
  void dispose() { _momoCtrl.dispose(); super.dispose(); }

  @override
  Widget build(BuildContext context) {
    final pt = widget.win['prize_type'] as String? ?? '';
    final label = widget.win['prize_label'] as String? ?? pt;
    final icon = _prizeIcons[pt] ?? Icons.card_giftcard_rounded;
    final color = _prizeColors[pt] ?? Colors.white;

    return Container(
      decoration: const BoxDecoration(
        color: NexusColors.surface,
        borderRadius: BorderRadius.vertical(top: Radius.circular(24)),
      ),
      child: Column(mainAxisSize: MainAxisSize.min, children: [
        // Handle
        Center(child: Container(
          margin: const EdgeInsets.only(top: 12, bottom: 20),
          width: 36, height: 4,
          decoration: BoxDecoration(
            color: Colors.white.withOpacity(0.15),
            borderRadius: BorderRadius.circular(2),
          ),
        )),

        Padding(
          padding: EdgeInsets.only(
            left: 24, right: 24,
            bottom: MediaQuery.of(context).viewInsets.bottom + 24,
          ),
          child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
            // Prize preview
            Row(children: [
              Container(
                width: 52, height: 52,
                decoration: BoxDecoration(
                  color: (_prizeBgColors[pt] ?? const Color(0x0AFFFFFF)),
                  borderRadius: BorderRadius.circular(14),
                ),
                child: Icon(icon, color: color, size: 26),
              ),
              const SizedBox(width: 14),
              Expanded(child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                Text('Claim Prize', style: const TextStyle(
                    color: NexusColors.textSecondary, fontSize: 11,
                    fontWeight: FontWeight.w600, letterSpacing: 0.5)),
                Text(label, style: const TextStyle(
                    color: NexusColors.textPrimary, fontSize: 18,
                    fontWeight: FontWeight.w800)),
              ])),
              GestureDetector(
                onTap: widget.onClose,
                child: const Icon(Icons.close_rounded, color: NexusColors.textSecondary),
              ),
            ]),

            const SizedBox(height: 20),

            if (_needsMomo) ...[
              const Text('YOUR MTN MOMO NUMBER',
                  style: TextStyle(color: NexusColors.textSecondary, fontSize: 10,
                      fontWeight: FontWeight.w700, letterSpacing: 0.8)),
              const SizedBox(height: 8),
              TextField(
                controller: _momoCtrl,
                keyboardType: TextInputType.phone,
                inputFormatters: [
                  FilteringTextInputFormatter.digitsOnly,
                  LengthLimitingTextInputFormatter(11),
                ],
                style: const TextStyle(color: NexusColors.textPrimary, fontSize: 16),
                onChanged: (_) => setState(() {}),
                decoration: InputDecoration(
                  hintText: '08031234567',
                  hintStyle: const TextStyle(color: NexusColors.textSecondary),
                  prefixIcon: const Icon(Icons.phone_android_rounded, color: NexusColors.primary),
                  filled: true,
                  fillColor: NexusColors.background,
                  border: OutlineInputBorder(borderRadius: BorderRadius.circular(12),
                      borderSide: const BorderSide(color: NexusColors.border)),
                  enabledBorder: OutlineInputBorder(borderRadius: BorderRadius.circular(12),
                      borderSide: const BorderSide(color: NexusColors.border)),
                  focusedBorder: OutlineInputBorder(borderRadius: BorderRadius.circular(12),
                      borderSide: const BorderSide(color: NexusColors.primary, width: 1.5)),
                ),
              ),
              const SizedBox(height: 6),
              const Text("Cash will be paid to your MoMo wallet within 24h of admin approval.",
                  style: TextStyle(color: NexusColors.gold, fontSize: 11)),
              const SizedBox(height: 16),
            ] else ...[
              Container(
                padding: const EdgeInsets.all(14),
                decoration: BoxDecoration(
                  color: NexusColors.green.withOpacity(0.08),
                  borderRadius: BorderRadius.circular(12),
                  border: Border.all(color: NexusColors.green.withOpacity(0.2)),
                ),
                child: Row(children: [
                  const Icon(Icons.check_circle_outline_rounded,
                      color: NexusColors.green, size: 16),
                  const SizedBox(width: 10),
                  Expanded(child: Text(
                    pt == 'airtime' || pt == 'data_bundle'
                        ? 'Airtime/Data will be credited to your registered number within minutes.'
                        : 'Your claim will be processed immediately.',
                    style: const TextStyle(color: NexusColors.green, fontSize: 12),
                  )),
                ]),
              ),
              const SizedBox(height: 16),
            ],

            if (_err != null) ...[
              Container(
                padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
                decoration: BoxDecoration(
                  color: NexusColors.red.withOpacity(0.1),
                  borderRadius: BorderRadius.circular(8),
                  border: Border.all(color: NexusColors.red.withOpacity(0.3)),
                ),
                child: Row(children: [
                  const Icon(Icons.error_outline_rounded, color: NexusColors.red, size: 14),
                  const SizedBox(width: 8),
                  Expanded(child: Text(_err!,
                      style: const TextStyle(color: NexusColors.red, fontSize: 12))),
                ]),
              ),
              const SizedBox(height: 12),
            ],

            Row(children: [
              Expanded(child: OutlinedButton(
                onPressed: widget.onClose,
                style: OutlinedButton.styleFrom(
                  foregroundColor: NexusColors.textSecondary,
                  side: const BorderSide(color: NexusColors.border),
                  padding: const EdgeInsets.symmetric(vertical: 14),
                  shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(14)),
                ),
                child: const Text('Cancel'),
              )),
              const SizedBox(width: 10),
              Expanded(child: ElevatedButton(
                onPressed: _canSubmit ? _claim : null,
                style: ElevatedButton.styleFrom(
                  backgroundColor: NexusColors.gold,
                  foregroundColor: Colors.black,
                  padding: const EdgeInsets.symmetric(vertical: 14),
                  shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(14)),
                ),
                child: _claiming
                    ? const SizedBox(width: 20, height: 20,
                        child: CircularProgressIndicator(color: Colors.black, strokeWidth: 2))
                    : Text(_needsMomo ? 'Submit Claim' : 'Claim Now',
                        style: const TextStyle(fontWeight: FontWeight.w800)),
              )),
            ]),
          ]),
        ),
      ]),
    );
  }
}

// ─── How it works card ────────────────────────────────────────────────────────

class _HowItWorksCard extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    return Container(
      margin: const EdgeInsets.only(top: 8),
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: NexusColors.surface,
        borderRadius: BorderRadius.circular(16),
        border: Border.all(color: Colors.white.withOpacity(0.05)),
      ),
      child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
        Text('How Claiming Works',
            style: TextStyle(color: Colors.white.withOpacity(0.4), fontSize: 11,
                fontWeight: FontWeight.w700, letterSpacing: 0.8)),
        const SizedBox(height: 10),
        for (final row in [
          ('📱', 'Airtime & Data: Auto-credited to your number within minutes of claiming'),
          ('💵', 'MoMo Cash: Submit your MoMo number → reviewed by admin → paid within 24h'),
          ('💎', 'Pulse Points: Instantly credited at spin time — nothing to claim'),
          ('⏰', 'Claims expire in 7 days — claim promptly!'),
        ]) ...[
          Row(crossAxisAlignment: CrossAxisAlignment.start, children: [
            Text(row.$1, style: const TextStyle(fontSize: 13)),
            const SizedBox(width: 8),
            Expanded(child: Text(row.$2,
                style: const TextStyle(color: NexusColors.textSecondary, fontSize: 11))),
          ]),
          const SizedBox(height: 6),
        ],
      ]),
    );
  }
}

// ─── Empty states ─────────────────────────────────────────────────────────────

class _EmptyWins extends StatelessWidget {
  final _Tab tab;
  const _EmptyWins({required this.tab});

  @override
  Widget build(BuildContext context) {
    final msg = switch (tab) {
      _Tab.pending => 'No prizes waiting to be claimed',
      _Tab.claimed => 'No claimed prizes yet',
      _Tab.all     => 'No prizes yet\nGo spin the wheel! 🎡',
    };
    return Center(child: Column(mainAxisAlignment: MainAxisAlignment.center, children: [
      const Text('🎁', style: TextStyle(fontSize: 52)),
      const SizedBox(height: 16),
      Text(msg, textAlign: TextAlign.center,
          style: const TextStyle(color: NexusColors.textSecondary, fontSize: 15)),
    ]));
  }
}

class _ErrorView extends StatelessWidget {
  final VoidCallback onRetry;
  const _ErrorView({required this.onRetry});
  @override
  Widget build(BuildContext context) => Center(child: Column(
    mainAxisAlignment: MainAxisAlignment.center, children: [
      const Icon(Icons.wifi_off_rounded, size: 52, color: NexusColors.textSecondary),
      const SizedBox(height: 12),
      const Text('Could not load prizes', style: TextStyle(color: NexusColors.textSecondary)),
      const SizedBox(height: 12),
      ElevatedButton(onPressed: onRetry, child: const Text('Retry')),
    ],
  ));
}
