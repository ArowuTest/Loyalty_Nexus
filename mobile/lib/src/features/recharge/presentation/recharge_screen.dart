import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:url_launcher/url_launcher.dart';

import '../../../core/auth/auth_provider.dart';
import '../../../core/theme/nexus_theme.dart';
import 'recharge_models.dart';
import 'recharge_providers.dart';

// ─── Recharge Screen ──────────────────────────────────────────────────────────

class RechargeScreen extends ConsumerStatefulWidget {
  const RechargeScreen({super.key});

  @override
  ConsumerState<RechargeScreen> createState() => _RechargeScreenState();
}

class _RechargeScreenState extends ConsumerState<RechargeScreen>
    with SingleTickerProviderStateMixin {
  // Tabs: 0 = Airtime, 1 = Data
  late final TabController _tabController;

  String? _selectedNetwork;
  final _phoneCtrl  = TextEditingController();
  final _amountCtrl = TextEditingController();
  DataBundle? _selectedBundle;
  bool _loading = false;

  static const _presets = [100, 200, 500, 1000, 2000, 5000];

  // Network badge colours
  static const _networkColors = <String, Color>{
    'MTN':     Color(0xFFFFCC00),
    'GLO':     Color(0xFF4CAF50),
    'AIRTEL':  Color(0xFFFF5722),
    '9MOBILE': Color(0xFF00897B),
  };

  @override
  void initState() {
    super.initState();
    _tabController = TabController(length: 2, vsync: this);

    // Pre-fill phone for logged-in users
    WidgetsBinding.instance.addPostFrameCallback((_) {
      final auth = ref.read(authStateProvider);
      if (auth.phoneNumber != null && auth.phoneNumber!.isNotEmpty) {
        _phoneCtrl.text = auth.phoneNumber!;
      }
    });

    _tabController.addListener(() {
      if (!_tabController.indexIsChanging) {
        setState(() {
          _selectedBundle = null;
          _amountCtrl.clear();
        });
      }
    });
  }

  @override
  void dispose() {
    _tabController.dispose();
    _phoneCtrl.dispose();
    _amountCtrl.dispose();
    super.dispose();
  }

  // ── helpers ──────────────────────────────────────────────────────────────────

  Color _networkColor(String code) =>
      _networkColors[code.toUpperCase()] ?? NexusColors.primary;

  String _networkEmoji(String code) {
    switch (code.toUpperCase()) {
      case 'MTN':     return '🟡';
      case 'GLO':     return '🟢';
      case 'AIRTEL':  return '🔴';
      case '9MOBILE': return '🟦';
      default:        return '📶';
    }
  }

  bool get _isAirtime => _tabController.index == 0;

  int? get _parsedAmount {
    final v = int.tryParse(_amountCtrl.text.replaceAll(',', ''));
    if (v == null || v < 50) return null;
    return v;
  }

  bool get _canProceed {
    final phone = _phoneCtrl.text.trim();
    if (phone.length < 10) return false;
    if (_selectedNetwork == null) return false;
    if (_isAirtime) return _parsedAmount != null;
    return _selectedBundle != null;
  }

  Future<void> _proceed(List<NetworkOperator> networks) async {
    if (!_canProceed || _loading) return;

    // Validate the selected network is active/enabled
    final net = networks.firstWhere((n) => n.code == _selectedNetwork,
        orElse: () => const NetworkOperator(
            code: '', name: '', isActive: false,
            airtimeEnabled: false, dataEnabled: false));
    if (!net.isActive) {
      _showSnack('This network is not currently available');
      return;
    }

    final phone   = _phoneCtrl.text.trim();
    final authState = ref.read(authStateProvider);
    final userId  = authState.isAuthenticated ? authState.phoneNumber : null;
    final api     = ref.read(rechargeApiProvider);

    setState(() => _loading = true);
    try {
      InitiateRechargeResponse res;
      if (_isAirtime) {
        res = await api.initiateAirtime(
          phone:       phone,
          networkCode: _selectedNetwork!,
          amount:      _parsedAmount!,
          userId:      userId,
        );
      } else {
        res = await api.initiateData(
          phone:         phone,
          networkCode:   _selectedNetwork!,
          variationCode: _selectedBundle!.id,
          amount:        _selectedBundle!.price,
          userId:        userId,
        );
      }

      // Open Paystack URL in device browser
      final uri = Uri.tryParse(res.paymentUrl);
      if (uri != null && await canLaunchUrl(uri)) {
        await launchUrl(uri, mode: LaunchMode.externalApplication);
      }

      if (mounted) {
        context.push('/recharge/success', extra: res.reference);
      }
    } on Exception catch (e) {
      if (mounted) _showSnack(e.toString().replaceFirst('ApiException: ', ''));
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  void _showSnack(String msg) {
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
        content: Text(msg),
        backgroundColor: NexusColors.surface,
        behavior: SnackBarBehavior.floating,
      ),
    );
  }

  // ── Build ─────────────────────────────────────────────────────────────────────

  @override
  Widget build(BuildContext context) {
    final networks = ref.watch(rechargeNetworksProvider);

    return Scaffold(
      backgroundColor: NexusColors.background,
      appBar: AppBar(
        backgroundColor: NexusColors.surface,
        elevation: 0,
        title: const Text(
          'Recharge',
          style: TextStyle(
            color: Colors.white,
            fontWeight: FontWeight.bold,
            fontSize: 20,
          ),
        ),
        leading: Navigator.canPop(context)
            ? IconButton(
                icon: const Icon(Icons.arrow_back_ios_new, color: Colors.white70),
                onPressed: () => context.pop(),
              )
            : null,
        bottom: PreferredSize(
          preferredSize: const Size.fromHeight(44),
          child: _buildDoublePointsBanner(),
        ),
      ),
      body: networks.when(
        loading: () => const Center(
          child: CircularProgressIndicator(color: NexusColors.gold),
        ),
        error: (err, _) => _buildError(err.toString()),
        data: (nets) => _buildBody(nets),
      ),
    );
  }

  Widget _buildDoublePointsBanner() {
    return Container(
      color: NexusColors.goldDim,
      width: double.infinity,
      padding: const EdgeInsets.symmetric(vertical: 6),
      child: const Row(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          Text('⚡', style: TextStyle(fontSize: 14)),
          SizedBox(width: 6),
          Text(
            'EARN DOUBLE POINTS on every recharge!',
            style: TextStyle(
              color: NexusColors.gold,
              fontSize: 12,
              fontWeight: FontWeight.w700,
              letterSpacing: 0.4,
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildError(String err) {
    return Center(
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          const Icon(Icons.wifi_off_rounded, color: Colors.white38, size: 48),
          const SizedBox(height: 12),
          Text(
            'Could not load networks',
            style: TextStyle(color: Colors.white.withValues(alpha: 0.5)),
          ),
          const SizedBox(height: 16),
          ElevatedButton(
            onPressed: () => ref.refresh(rechargeNetworksProvider),
            style: ElevatedButton.styleFrom(
              backgroundColor: NexusColors.primary,
              foregroundColor: Colors.white,
            ),
            child: const Text('Retry'),
          ),
        ],
      ),
    );
  }

  Widget _buildBody(List<NetworkOperator> networks) {
    return Column(
      children: [
        Expanded(
          child: SingleChildScrollView(
            padding: const EdgeInsets.all(16),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                // ── Type toggle (Airtime / Data) ──
                _SectionLabel(text: 'Recharge Type'),
                const SizedBox(height: 8),
                _buildTypeToggle(),
                const SizedBox(height: 20),

                // ── Phone number ──
                _SectionLabel(text: 'Phone Number'),
                const SizedBox(height: 8),
                _buildPhoneInput(),
                const SizedBox(height: 20),

                // ── Network selection ──
                _SectionLabel(text: 'Select Network'),
                const SizedBox(height: 10),
                _buildNetworkGrid(networks),
                const SizedBox(height: 20),

                // ── Amount / Bundles ──
                TabBarView(
                  controller: _tabController,
                  physics: const NeverScrollableScrollPhysics(),
                  children: [
                    _buildAirtimeSection(),
                    _buildDataSection(),
                  ],
                ),

                const SizedBox(height: 20),

                // ── Summary ──
                if (_canProceed) ...[
                  _buildSummaryCard(networks),
                  const SizedBox(height: 20),
                ],

                // ── Auth nudge for guests ──
                _buildAuthNudge(),
                const SizedBox(height: 80),
              ],
            ),
          ),
        ),

        // ── Sticky CTA ──
        _buildStickyButton(networks),
      ],
    );
  }

  Widget _buildTypeToggle() {
    return Container(
      decoration: BoxDecoration(
        color: NexusColors.surface,
        borderRadius: BorderRadius.circular(12),
      ),
      child: TabBar(
        controller: _tabController,
        indicator: BoxDecoration(
          color: NexusColors.primary,
          borderRadius: BorderRadius.circular(10),
        ),
        indicatorSize: TabBarIndicatorSize.tab,
        dividerColor: Colors.transparent,
        labelColor: Colors.white,
        unselectedLabelColor: Colors.white54,
        labelStyle: const TextStyle(fontWeight: FontWeight.w700, fontSize: 14),
        tabs: const [
          Tab(text: '📞  Airtime'),
          Tab(text: '📶  Data'),
        ],
      ),
    );
  }

  Widget _buildPhoneInput() {
    return TextField(
      controller: _phoneCtrl,
      keyboardType: TextInputType.phone,
      inputFormatters: [FilteringTextInputFormatter.digitsOnly],
      maxLength: 11,
      style: const TextStyle(color: Colors.white, fontSize: 16),
      decoration: InputDecoration(
        counterText: '',
        hintText: '080XXXXXXXX',
        hintStyle: const TextStyle(color: Colors.white38),
        prefixIcon:
            const Icon(Icons.phone_android_rounded, color: Colors.white54),
        filled: true,
        fillColor: NexusColors.surface,
        border: OutlineInputBorder(
          borderRadius: BorderRadius.circular(12),
          borderSide: BorderSide.none,
        ),
        focusedBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(12),
          borderSide: BorderSide(color: NexusColors.primary, width: 1.5),
        ),
      ),
      onChanged: (_) => setState(() {}),
    );
  }

  Widget _buildNetworkGrid(List<NetworkOperator> networks) {
    return GridView.builder(
      shrinkWrap: true,
      physics: const NeverScrollableScrollPhysics(),
      gridDelegate: const SliverGridDelegateWithFixedCrossAxisCount(
        crossAxisCount:    4,
        crossAxisSpacing:  8,
        mainAxisSpacing:   8,
        childAspectRatio:  1.1,
      ),
      itemCount: networks.length,
      itemBuilder: (_, i) {
        final net     = networks[i];
        final enabled = net.isActive &&
            (_isAirtime ? net.airtimeEnabled : net.dataEnabled);
        final selected = _selectedNetwork == net.code;

        return GestureDetector(
          onTap: enabled
              ? () => setState(() {
                    _selectedNetwork = net.code;
                    _selectedBundle  = null;
                  })
              : null,
          child: AnimatedContainer(
            duration: const Duration(milliseconds: 200),
            decoration: BoxDecoration(
              color: selected ? _networkColor(net.code).withValues(alpha: 0.15) : NexusColors.surface,
              borderRadius: BorderRadius.circular(12),
              border: Border.all(
                color: selected
                    ? _networkColor(net.code)
                    : (enabled ? Colors.white12 : Colors.white06),
                width: selected ? 2 : 1,
              ),
            ),
            child: Opacity(
              opacity: enabled ? 1.0 : 0.35,
              child: Column(
                mainAxisAlignment: MainAxisAlignment.center,
                children: [
                  Text(
                    _networkEmoji(net.code),
                    style: const TextStyle(fontSize: 22),
                  ),
                  const SizedBox(height: 4),
                  Text(
                    net.name,
                    style: TextStyle(
                      color:      enabled ? Colors.white : Colors.white38,
                      fontSize:   11,
                      fontWeight: selected ? FontWeight.bold : FontWeight.normal,
                    ),
                    textAlign: TextAlign.center,
                    maxLines:  1,
                    overflow:  TextOverflow.ellipsis,
                  ),
                  if (!enabled)
                    const Text(
                      'soon',
                      style: TextStyle(color: Colors.white24, fontSize: 9),
                    ),
                ],
              ),
            ),
          ),
        );
      },
    );
  }

  // ── Airtime section ───────────────────────────────────────────────────────────

  Widget _buildAirtimeSection() {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        _SectionLabel(text: 'Amount (₦)'),
        const SizedBox(height: 10),
        GridView.builder(
          shrinkWrap:  true,
          physics:     const NeverScrollableScrollPhysics(),
          gridDelegate: const SliverGridDelegateWithFixedCrossAxisCount(
            crossAxisCount:   3,
            crossAxisSpacing: 8,
            mainAxisSpacing:  8,
            childAspectRatio: 2.6,
          ),
          itemCount: _presets.length,
          itemBuilder: (_, i) {
            final val      = _presets[i];
            final selected = _amountCtrl.text == val.toString();
            return GestureDetector(
              onTap: () => setState(() => _amountCtrl.text = val.toString()),
              child: AnimatedContainer(
                duration: const Duration(milliseconds: 150),
                decoration: BoxDecoration(
                  color: selected
                      ? NexusColors.primary.withValues(alpha: 0.2)
                      : NexusColors.surface,
                  borderRadius: BorderRadius.circular(10),
                  border: Border.all(
                    color: selected ? NexusColors.primary : Colors.white12,
                    width: selected ? 2 : 1,
                  ),
                ),
                alignment: Alignment.center,
                child: Text(
                  '₦${_fmt(val)}',
                  style: TextStyle(
                    color:      selected ? NexusColors.primary : Colors.white70,
                    fontWeight: FontWeight.w600,
                    fontSize:   13,
                  ),
                ),
              ),
            );
          },
        ),
        const SizedBox(height: 12),
        TextField(
          controller: _amountCtrl,
          keyboardType: TextInputType.number,
          inputFormatters: [FilteringTextInputFormatter.digitsOnly],
          style: const TextStyle(color: Colors.white, fontSize: 16),
          decoration: InputDecoration(
            hintText:  'Enter custom amount',
            hintStyle: const TextStyle(color: Colors.white38),
            prefixText: '₦ ',
            prefixStyle: const TextStyle(color: Colors.white60),
            filled:     true,
            fillColor:  NexusColors.surface,
            border:     OutlineInputBorder(
              borderRadius: BorderRadius.circular(12),
              borderSide:   BorderSide.none,
            ),
            focusedBorder: OutlineInputBorder(
              borderRadius: BorderRadius.circular(12),
              borderSide:
                  BorderSide(color: NexusColors.primary, width: 1.5),
            ),
          ),
          onChanged: (_) => setState(() {}),
        ),
      ],
    );
  }

  // ── Data bundles section ──────────────────────────────────────────────────────

  Widget _buildDataSection() {
    if (_selectedNetwork == null) {
      return _HintBox(
        icon:  Icons.wifi_find_rounded,
        label: 'Select a network to see data bundles',
      );
    }

    final bundles = ref.watch(rechargeBundlesProvider(_selectedNetwork!));
    return bundles.when(
      loading: () => const Center(
        child: Padding(
          padding: EdgeInsets.all(24),
          child: CircularProgressIndicator(color: NexusColors.gold),
        ),
      ),
      error: (e, _) => _HintBox(
        icon:  Icons.error_outline_rounded,
        label: 'Could not load bundles. Tap to retry.',
        onTap: () => ref.refresh(rechargeBundlesProvider(_selectedNetwork!)),
      ),
      data: (bundles) {
        if (bundles.isEmpty) {
          return _HintBox(
            icon:  Icons.signal_cellular_off_rounded,
            label: 'No data bundles available for this network',
          );
        }
        return ListView.separated(
          shrinkWrap: true,
          physics:    const NeverScrollableScrollPhysics(),
          itemCount:  bundles.length,
          separatorBuilder: (_, __) => const SizedBox(height: 8),
          itemBuilder: (_, i) {
            final b        = bundles[i];
            final selected = _selectedBundle?.id == b.id;
            return GestureDetector(
              onTap: () => setState(() => _selectedBundle = b),
              child: AnimatedContainer(
                duration: const Duration(milliseconds: 150),
                padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 14),
                decoration: BoxDecoration(
                  color: selected
                      ? NexusColors.primary.withValues(alpha: 0.15)
                      : NexusColors.surface,
                  borderRadius: BorderRadius.circular(12),
                  border: Border.all(
                    color: selected ? NexusColors.primary : Colors.white12,
                    width: selected ? 2 : 1,
                  ),
                ),
                child: Row(
                  children: [
                    Icon(
                      Icons.data_usage_rounded,
                      color: selected ? NexusColors.primary : Colors.white38,
                      size:  20,
                    ),
                    const SizedBox(width: 12),
                    Expanded(
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text(
                            b.name,
                            style: TextStyle(
                              color:      selected ? Colors.white : Colors.white70,
                              fontWeight: FontWeight.w600,
                              fontSize:   14,
                            ),
                          ),
                          if (b.validity.isNotEmpty)
                            Text(
                              b.validity,
                              style: const TextStyle(
                                color:    Colors.white38,
                                fontSize: 12,
                              ),
                            ),
                        ],
                      ),
                    ),
                    Text(
                      '₦${_fmt(b.price)}',
                      style: TextStyle(
                        color:      selected ? NexusColors.primary : NexusColors.gold,
                        fontWeight: FontWeight.bold,
                        fontSize:   15,
                      ),
                    ),
                    const SizedBox(width: 4),
                    Icon(
                      selected
                          ? Icons.check_circle_rounded
                          : Icons.radio_button_unchecked,
                      color: selected ? NexusColors.primary : Colors.white24,
                      size:  20,
                    ),
                  ],
                ),
              ),
            );
          },
        );
      },
    );
  }

  // ── Summary card ──────────────────────────────────────────────────────────────

  Widget _buildSummaryCard(List<NetworkOperator> networks) {
    final netName = networks
        .firstWhere(
          (n) => n.code == _selectedNetwork,
          orElse: () => const NetworkOperator(
            code: '', name: '', isActive: false,
            airtimeEnabled: false, dataEnabled: false,
          ),
        )
        .name;

    final amount = _isAirtime
        ? '₦${_fmt(_parsedAmount ?? 0)}'
        : '₦${_fmt(_selectedBundle?.price ?? 0)}';
    final label  = _isAirtime
        ? 'Airtime'
        : (_selectedBundle?.name ?? '');

    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color:        NexusColors.surfaceHigh,
        borderRadius: BorderRadius.circular(16),
        border:       Border.all(color: NexusColors.goldDim),
      ),
      child: Column(
        children: [
          _SummaryRow(label: 'Type',    value: _isAirtime ? 'Airtime' : 'Data Bundle'),
          _SummaryRow(label: 'Network', value: netName),
          _SummaryRow(label: 'Phone',   value: _phoneCtrl.text),
          _SummaryRow(label: label,     value: amount),
          const Divider(color: Colors.white12, height: 20),
          const Row(
            children: [
              Text('⚡ ', style: TextStyle(fontSize: 14)),
              Expanded(
                child: Text(
                  'You will earn DOUBLE Pulse Points on this recharge!',
                  style: TextStyle(
                    color:      NexusColors.gold,
                    fontSize:   12,
                    fontWeight: FontWeight.w600,
                  ),
                ),
              ),
            ],
          ),
        ],
      ),
    );
  }

  // ── Auth nudge ────────────────────────────────────────────────────────────────

  Widget _buildAuthNudge() {
    final auth = ref.watch(authStateProvider);
    if (auth.isAuthenticated || auth.isLoading) return const SizedBox.shrink();

    return Container(
      padding: const EdgeInsets.all(14),
      decoration: BoxDecoration(
        color:        NexusColors.surface,
        borderRadius: BorderRadius.circular(12),
        border:       Border.all(color: Colors.white12),
      ),
      child: Row(
        children: [
          const Icon(Icons.info_outline_rounded, color: Colors.white38, size: 18),
          const SizedBox(width: 10),
          Expanded(
            child: Text(
              'Sign in to earn & track your Pulse Points',
              style: TextStyle(
                color:    Colors.white.withValues(alpha: 0.55),
                fontSize: 13,
              ),
            ),
          ),
          TextButton(
            onPressed: () => context.go('/'),
            style: TextButton.styleFrom(
              foregroundColor: NexusColors.primary,
              padding: const EdgeInsets.symmetric(horizontal: 8),
            ),
            child: const Text('Sign In', style: TextStyle(fontSize: 12)),
          ),
        ],
      ),
    );
  }

  // ── Sticky CTA button ─────────────────────────────────────────────────────────

  Widget _buildStickyButton(List<NetworkOperator> networks) {
    final amount = _isAirtime
        ? (_parsedAmount != null ? '₦${_fmt(_parsedAmount!)}' : null)
        : (_selectedBundle != null ? '₦${_fmt(_selectedBundle!.price)}' : null);

    return SafeArea(
      top: false,
      child: Container(
        padding: const EdgeInsets.fromLTRB(16, 12, 16, 8),
        decoration: BoxDecoration(
          color: NexusColors.surface,
          boxShadow: [
            BoxShadow(
              color:       Colors.black.withValues(alpha: 0.3),
              blurRadius:  8,
              offset:      const Offset(0, -2),
            ),
          ],
        ),
        child: SizedBox(
          width:  double.infinity,
          height: 52,
          child: ElevatedButton(
            onPressed: _canProceed && !_loading
                ? () => _proceed(networks)
                : null,
            style: ElevatedButton.styleFrom(
              backgroundColor:         NexusColors.primary,
              disabledBackgroundColor: Colors.white12,
              foregroundColor:         Colors.white,
              disabledForegroundColor: Colors.white38,
              shape: RoundedRectangleBorder(
                borderRadius: BorderRadius.circular(14),
              ),
              elevation: 0,
            ),
            child: _loading
                ? const SizedBox(
                    width:  20,
                    height: 20,
                    child: CircularProgressIndicator(
                      color:      Colors.white,
                      strokeWidth: 2,
                    ),
                  )
                : Text(
                    amount != null
                        ? 'Pay $amount with Paystack'
                        : _isAirtime
                            ? 'Enter details to continue'
                            : 'Select a bundle to continue',
                    style: const TextStyle(
                      fontWeight: FontWeight.w700,
                      fontSize:   16,
                    ),
                  ),
          ),
        ),
      ),
    );
  }

  // ── Utility ───────────────────────────────────────────────────────────────────

  String _fmt(int n) {
    if (n >= 1000) {
      final whole = n ~/ 1000;
      final frac  = n % 1000;
      return frac == 0 ? '${whole}k' : '$n';
    }
    return n.toString();
  }
}

// ─── Shared small widgets ─────────────────────────────────────────────────────

class _SectionLabel extends StatelessWidget {
  final String text;
  const _SectionLabel({required this.text});

  @override
  Widget build(BuildContext context) => Text(
        text,
        style: const TextStyle(
          color:       Colors.white60,
          fontSize:    13,
          fontWeight:  FontWeight.w600,
          letterSpacing: 0.5,
        ),
      );
}

class _SummaryRow extends StatelessWidget {
  final String label;
  final String value;
  const _SummaryRow({required this.label, required this.value});

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 4),
      child: Row(
        children: [
          Text(
            '$label:',
            style: const TextStyle(color: Colors.white54, fontSize: 13),
          ),
          const SizedBox(width: 8),
          Expanded(
            child: Text(
              value,
              textAlign: TextAlign.end,
              style: const TextStyle(
                color:      Colors.white,
                fontSize:   13,
                fontWeight: FontWeight.w600,
              ),
            ),
          ),
        ],
      ),
    );
  }
}

class _HintBox extends StatelessWidget {
  final IconData icon;
  final String   label;
  final VoidCallback? onTap;
  const _HintBox({required this.icon, required this.label, this.onTap});

  @override
  Widget build(BuildContext context) {
    return GestureDetector(
      onTap: onTap,
      child: Container(
        padding: const EdgeInsets.all(24),
        decoration: BoxDecoration(
          color:        NexusColors.surface,
          borderRadius: BorderRadius.circular(12),
        ),
        child: Column(
          children: [
            Icon(icon, color: Colors.white24, size: 36),
            const SizedBox(height: 8),
            Text(
              label,
              textAlign: TextAlign.center,
              style: const TextStyle(color: Colors.white38, fontSize: 13),
            ),
          ],
        ),
      ),
    );
  }
}
