import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:url_launcher/url_launcher.dart';

import '../../../core/api/api_client.dart';
import '../../../core/auth/auth_provider.dart';
import '../../../core/theme/nexus_theme.dart';

// ═══════════════════════════════════════════════════════════════════════════════
// MODELS
// ═══════════════════════════════════════════════════════════════════════════════

enum RechargeType { airtime, data }

@immutable
class NetworkOperator {
  final String code;
  final String name;
  final bool isActive;
  final bool airtimeEnabled;
  final bool dataEnabled;

  const NetworkOperator({
    required this.code,
    required this.name,
    required this.isActive,
    required this.airtimeEnabled,
    required this.dataEnabled,
  });

  factory NetworkOperator.fromJson(Map<String, dynamic> j) => NetworkOperator(
        code:           j['code']            as String? ?? '',
        name:           j['name']            as String? ?? '',
        isActive:       j['is_active']       as bool?   ?? false,
        airtimeEnabled: j['airtime_enabled'] as bool?   ?? false,
        dataEnabled:    j['data_enabled']    as bool?   ?? false,
      );

  bool isEnabledFor(RechargeType type) =>
      isActive && (type == RechargeType.airtime ? airtimeEnabled : dataEnabled);
}

@immutable
class DataBundle {
  final String id;       // VTPass variation_code
  final String name;
  final int price;
  final String validity;

  const DataBundle({
    required this.id,
    required this.name,
    required this.price,
    this.validity = '',
  });

  factory DataBundle.fromJson(Map<String, dynamic> j) => DataBundle(
        id:       j['id']       as String? ?? '',
        name:     j['name']     as String? ?? '',
        price:    (j['price']   as num?)?.toInt() ?? 0,
        validity: j['validity'] as String? ?? '',
      );
}

// ═══════════════════════════════════════════════════════════════════════════════
// FORM STATE + VIEW-MODEL
// ═══════════════════════════════════════════════════════════════════════════════

@immutable
class RechargeFormState {
  final RechargeType type;
  final String? selectedNetwork;
  final String phone;
  final int? amount;
  final DataBundle? selectedBundle;
  final bool isSubmitting;
  final String? errorMessage;

  const RechargeFormState({
    this.type = RechargeType.airtime,
    this.selectedNetwork,
    this.phone = '',
    this.amount,
    this.selectedBundle,
    this.isSubmitting = false,
    this.errorMessage,
  });

  RechargeFormState copyWith({
    RechargeType? type,
    String? selectedNetwork,
    String? phone,
    int? amount,
    DataBundle? selectedBundle,
    bool? isSubmitting,
    String? errorMessage,
    bool clearAmount = false,
    bool clearBundle = false,
    bool clearError  = false,
    bool clearNetwork = false,
  }) =>
      RechargeFormState(
        type:            type            ?? this.type,
        selectedNetwork: clearNetwork ? null : selectedNetwork ?? this.selectedNetwork,
        phone:           phone           ?? this.phone,
        amount:          clearAmount ? null : amount ?? this.amount,
        selectedBundle:  clearBundle ? null : selectedBundle  ?? this.selectedBundle,
        isSubmitting:    isSubmitting    ?? this.isSubmitting,
        errorMessage:    clearError ? null : errorMessage ?? this.errorMessage,
      );

  bool get canProceed {
    if (phone.trim().length < 10) return false;
    if (selectedNetwork == null) return false;
    if (type == RechargeType.airtime) return amount != null && amount! >= 50;
    return selectedBundle != null;
  }

  int? get effectiveAmount =>
      type == RechargeType.airtime ? amount : selectedBundle?.price;
}

// ─── ViewModel ────────────────────────────────────────────────────────────────

class RechargeFormNotifier extends StateNotifier<RechargeFormState> {
  RechargeFormNotifier() : super(const RechargeFormState());

  void setType(RechargeType t) => state = state.copyWith(
        type: t,
        clearAmount: true,
        clearBundle: true,
        clearError:  true,
      );

  void selectNetwork(String code) => state = state.copyWith(
        selectedNetwork: code,
        clearBundle:     true,
        clearError:      true,
      );

  void setPhone(String v) => state = state.copyWith(phone: v, clearError: true);

  void setAmount(int v) => state = state.copyWith(
        amount:      v,
        clearBundle: true,
        clearError:  true,
      );

  void selectBundle(DataBundle b) => state = state.copyWith(
        selectedBundle: b,
        clearAmount:    true,
        clearError:     true,
      );

  void setError(String? msg) => state = state.copyWith(errorMessage: msg);

  void _setSubmitting(bool v) => state = state.copyWith(isSubmitting: v);

  void prefillPhone(String? phone) {
    if (phone != null && phone.isNotEmpty && state.phone.isEmpty) {
      state = state.copyWith(phone: phone);
    }
  }

  // ── Submit ───────────────────────────────────────────────────────────────────

  Future<String?> submit(Dio dio, String? userId) async {
    if (!state.canProceed) return null;
    _setSubmitting(true);
    state = state.copyWith(clearError: true);

    try {
      final body = <String, dynamic>{
        'msisdn':        state.phone.trim(),
        'network':       state.selectedNetwork,
        'amount_kobo':   (state.effectiveAmount ?? 0) * 100,
        'recharge_type': state.type.name,
        if (state.type == RechargeType.data)
          'variation_code': state.selectedBundle!.id,
        if (userId != null) 'user_id': userId,
      };

      final raw = await dio.apiPost<Map<String, dynamic>>(
        '/recharge/initiate',
        data: body,
      );

      final paymentUrl = raw['payment_url'] as String? ?? '';
      final reference  = raw['reference']  as String? ?? '';

      final uri = Uri.tryParse(paymentUrl);
      if (uri != null && await canLaunchUrl(uri)) {
        await launchUrl(uri, mode: LaunchMode.externalApplication);
      }

      return reference;
    } on ApiException catch (e) {
      state = state.copyWith(errorMessage: e.message);
      return null;
    } catch (e) {
      state = state.copyWith(errorMessage: 'Something went wrong. Please try again.');
      return null;
    } finally {
      _setSubmitting(false);
    }
  }
}

// ─── Providers ────────────────────────────────────────────────────────────────

final rechargeFormProvider =
    StateNotifierProvider.autoDispose<RechargeFormNotifier, RechargeFormState>(
  (_) => RechargeFormNotifier(),
);

final rechargeNetworksProvider =
    FutureProvider.autoDispose<List<NetworkOperator>>((ref) async {
  final raw = await ref.read(dioProvider).apiGet<List<dynamic>>('/recharge/networks');
  return raw
      .map((e) => NetworkOperator.fromJson(e as Map<String, dynamic>))
      .toList();
});

final rechargeBundlesProvider =
    FutureProvider.autoDispose.family<List<DataBundle>, String>((ref, networkCode) async {
  final raw = await ref
      .read(dioProvider)
      .apiGet<List<dynamic>>('/recharge/networks/$networkCode/bundles');
  return raw
      .map((e) => DataBundle.fromJson(e as Map<String, dynamic>))
      .toList();
});

// ═══════════════════════════════════════════════════════════════════════════════
// SCREEN
// ═══════════════════════════════════════════════════════════════════════════════

class RechargeScreen extends ConsumerWidget {
  const RechargeScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    // Pre-fill phone for logged-in users — use ref.select to minimise rebuilds
    final phone = ref.watch(authStateProvider.select((s) => s.phoneNumber));
    ref.read(rechargeFormProvider.notifier).prefillPhone(phone);

    final networksAsync = ref.watch(rechargeNetworksProvider);
    final form          = ref.watch(rechargeFormProvider);

    return Scaffold(
      backgroundColor: NexusColors.background,
      body: GestureDetector(
        onTap: () => FocusScope.of(context).unfocus(),
        child: networksAsync.when(
          loading: () => const _LoadingBody(),
          error:   (e, _) => _ErrorBody(onRetry: () => ref.refresh(rechargeNetworksProvider)),
          data:    (networks) => _RechargeBody(networks: networks, form: form),
        ),
      ),
      bottomNavigationBar: networksAsync.maybeWhen(
        data: (networks) => _StickyPayButton(form: form, networks: networks),
        orElse: () => const SizedBox.shrink(),
      ),
    );
  }
}

// ═══════════════════════════════════════════════════════════════════════════════
// BODY
// ═══════════════════════════════════════════════════════════════════════════════

class _RechargeBody extends ConsumerWidget {
  final List<NetworkOperator> networks;
  final RechargeFormState form;

  const _RechargeBody({required this.networks, required this.form});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return CustomScrollView(
      keyboardDismissBehavior: ScrollViewKeyboardDismissBehavior.onDrag,
      slivers: [
        // ── App bar ─────────────────────────────────────────────────────────
        SliverAppBar(
          backgroundColor: NexusColors.surface,
          surfaceTintColor: Colors.transparent,
          elevation: 0,
          pinned: true,
          leading: Navigator.canPop(context)
              ? IconButton(
                  icon: const Icon(Icons.arrow_back_ios_new,
                      color: Colors.white70, size: 20),
                  tooltip: 'Back',
                  onPressed: () => context.pop(),
                )
              : null,
          title: const Text(
            'Recharge',
            style: TextStyle(
              color: Colors.white,
              fontWeight: FontWeight.w800,
              fontSize: 20,
            ),
          ),
          bottom: const PreferredSize(
            preferredSize: Size.fromHeight(34),
            child: _DoublePointsBanner(),
          ),
        ),

        SliverPadding(
          padding: const EdgeInsets.fromLTRB(16, 20, 16, 24),
          sliver: SliverList(
            delegate: SliverChildListDelegate([

              // ── Error message ──
              if (form.errorMessage != null) ...[
                _ErrorBanner(message: form.errorMessage!),
                const SizedBox(height: 16),
              ],

              // ── Type toggle ──
              const _SectionLabel('Recharge Type'),
              const SizedBox(height: 8),
              _TypeToggle(current: form.type),
              const SizedBox(height: 20),

              // ── Phone ──
              const _SectionLabel('Phone Number'),
              const SizedBox(height: 8),
              _PhoneField(value: form.phone),
              const SizedBox(height: 20),

              // ── Network grid ──
              const _SectionLabel('Select Network'),
              const SizedBox(height: 10),
              _NetworkGrid(
                networks: networks,
                selected: form.selectedNetwork,
                type:     form.type,
              ),
              const SizedBox(height: 20),

              // ── Amount or bundles ──
              if (form.type == RechargeType.airtime) ...[
                const _SectionLabel('Amount'),
                const SizedBox(height: 10),
                _AirtimeSection(currentAmount: form.amount),
                const SizedBox(height: 20),
              ] else ...[
                const _SectionLabel('Data Bundles'),
                const SizedBox(height: 10),
                _DataSection(
                  networkCode:    form.selectedNetwork,
                  selectedBundle: form.selectedBundle,
                ),
                const SizedBox(height: 20),
              ],

              // ── Summary ──
              if (form.canProceed) ...[
                _SummaryCard(form: form, networks: networks),
                const SizedBox(height: 20),
              ],

              // ── Guest nudge ──
              const _GuestNudge(),
              const SizedBox(height: 16),
            ]),
          ),
        ),
      ],
    );
  }
}

// ═══════════════════════════════════════════════════════════════════════════════
// SUB-WIDGETS
// ═══════════════════════════════════════════════════════════════════════════════

// ── Loading / Error states ───────────────────────────────────────────────────

class _LoadingBody extends StatelessWidget {
  const _LoadingBody();

  @override
  Widget build(BuildContext context) {
    return const Center(
      child: CircularProgressIndicator(color: NexusColors.gold),
    );
  }
}

class _ErrorBody extends StatelessWidget {
  final VoidCallback onRetry;
  const _ErrorBody({required this.onRetry});

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(32),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            const Icon(Icons.wifi_off_rounded, color: Colors.white38, size: 52),
            const SizedBox(height: 16),
            const Text(
              'Could not load networks',
              style: TextStyle(color: Colors.white54, fontSize: 16),
            ),
            const SizedBox(height: 20),
            FilledButton.icon(
              onPressed: onRetry,
              icon: const Icon(Icons.refresh_rounded),
              label: const Text('Retry'),
              style: FilledButton.styleFrom(
                backgroundColor:  NexusColors.primary,
                foregroundColor:  Colors.white,
                minimumSize:      const Size(140, 48),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

// ── Double-points banner ──────────────────────────────────────────────────────

class _DoublePointsBanner extends StatelessWidget {
  const _DoublePointsBanner();

  @override
  Widget build(BuildContext context) {
    return Container(
      color: NexusColors.goldDim,
      width: double.infinity,
      padding: const EdgeInsets.symmetric(vertical: 7),
      child: const Row(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          Text('⚡', style: TextStyle(fontSize: 14)),
          SizedBox(width: 6),
          Text(
            'EARN DOUBLE POINTS on every recharge!',
            style: TextStyle(
              color:         NexusColors.gold,
              fontSize:      12,
              fontWeight:    FontWeight.w700,
              letterSpacing: 0.5,
            ),
          ),
        ],
      ),
    );
  }
}

// ── Section label ─────────────────────────────────────────────────────────────

class _SectionLabel extends StatelessWidget {
  final String label;
  const _SectionLabel(this.label);

  @override
  Widget build(BuildContext context) {
    return Text(
      label,
      style: const TextStyle(
        color:         Colors.white60,
        fontSize:      12,
        fontWeight:    FontWeight.w700,
        letterSpacing: 0.6,
      ),
    );
  }
}

// ── Error banner ──────────────────────────────────────────────────────────────

class _ErrorBanner extends StatelessWidget {
  final String message;
  const _ErrorBanner({required this.message});

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 10),
      decoration: BoxDecoration(
        color:        const Color(0xFF7F1D1D).withValues(alpha: 0.4),
        borderRadius: BorderRadius.circular(10),
        border:       Border.all(color: const Color(0xFFF87171).withValues(alpha: 0.5)),
      ),
      child: Row(
        children: [
          const Icon(Icons.error_outline_rounded, color: Color(0xFFF87171), size: 18),
          const SizedBox(width: 8),
          Expanded(
            child: Text(
              message,
              style: const TextStyle(color: Color(0xFFF87171), fontSize: 13),
            ),
          ),
        ],
      ),
    );
  }
}

// ── Type toggle ───────────────────────────────────────────────────────────────

class _TypeToggle extends ConsumerWidget {
  final RechargeType current;
  const _TypeToggle({required this.current});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return Container(
      decoration: BoxDecoration(
        color:        NexusColors.surface,
        borderRadius: BorderRadius.circular(12),
      ),
      padding: const EdgeInsets.all(4),
      child: Row(
        children: [
          _TypeChip(
            label:    '📞  Airtime',
            type:     RechargeType.airtime,
            selected: current == RechargeType.airtime,
          ),
          _TypeChip(
            label:    '📶  Data',
            type:     RechargeType.data,
            selected: current == RechargeType.data,
          ),
        ],
      ),
    );
  }
}

class _TypeChip extends ConsumerWidget {
  final String label;
  final RechargeType type;
  final bool selected;
  const _TypeChip({required this.label, required this.type, required this.selected});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return Expanded(
      child: Semantics(
        label: label,
        selected: selected,
        button: true,
        child: AnimatedContainer(
          duration: const Duration(milliseconds: 200),
          curve: Curves.easeInOut,
          decoration: BoxDecoration(
            color:        selected ? NexusColors.primary : Colors.transparent,
            borderRadius: BorderRadius.circular(9),
          ),
          child: Material(
            color:        Colors.transparent,
            borderRadius: BorderRadius.circular(9),
            child: InkWell(
              borderRadius: BorderRadius.circular(9),
              onTap: selected ? null : () {
                HapticFeedback.selectionClick();
                ref.read(rechargeFormProvider.notifier).setType(type);
              },
              child: Padding(
                padding: const EdgeInsets.symmetric(vertical: 12),
                child: Text(
                  label,
                  textAlign: TextAlign.center,
                  style: TextStyle(
                    color:      Colors.white,
                    fontWeight: selected ? FontWeight.w700 : FontWeight.w500,
                    fontSize:   14,
                  ),
                ),
              ),
            ),
          ),
        ),
      ),
    );
  }
}

// ── Phone field ───────────────────────────────────────────────────────────────

class _PhoneField extends ConsumerWidget {
  final String value;
  const _PhoneField({required this.value});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return Semantics(
      label: 'Phone number input',
      textField: true,
      child: TextField(
        controller: TextEditingController.fromValue(
          TextEditingValue(
            text: value,
            selection: TextSelection.collapsed(offset: value.length),
          ),
        ),
        keyboardType:     TextInputType.phone,
        textInputAction:  TextInputAction.done,
        inputFormatters:  [FilteringTextInputFormatter.digitsOnly],
        maxLength:        11,
        style: const TextStyle(color: Colors.white, fontSize: 16),
        decoration: InputDecoration(
          counterText: '',
          hintText:    '080XXXXXXXX',
          hintStyle:   const TextStyle(color: Colors.white38),
          prefixIcon: const Icon(Icons.phone_android_rounded, color: Colors.white54),
          filled:      true,
          fillColor:   NexusColors.surface,
          border:      OutlineInputBorder(
            borderRadius: BorderRadius.circular(12),
            borderSide:   BorderSide.none,
          ),
          focusedBorder: OutlineInputBorder(
            borderRadius: BorderRadius.circular(12),
            borderSide:   const BorderSide(color: NexusColors.primary, width: 1.5),
          ),
          errorBorder: OutlineInputBorder(
            borderRadius: BorderRadius.circular(12),
            borderSide: const BorderSide(color: Color(0xFFF87171), width: 1.5),
          ),
        ),
        onChanged: (v) => ref.read(rechargeFormProvider.notifier).setPhone(v),
      ),
    );
  }
}

// ── Network grid ──────────────────────────────────────────────────────────────

const _networkColors = <String, Color>{
  'MTN':     Color(0xFFFFCC00),
  'GLO':     Color(0xFF4CAF50),
  'AIRTEL':  Color(0xFFFF5722),
  '9MOBILE': Color(0xFF00897B),
};

const _networkEmojis = <String, String>{
  'MTN':     '🟡',
  'GLO':     '🟢',
  'AIRTEL':  '🔴',
  '9MOBILE': '🟦',
};

class _NetworkGrid extends ConsumerWidget {
  final List<NetworkOperator> networks;
  final String? selected;
  final RechargeType type;

  const _NetworkGrid({
    required this.networks,
    required this.selected,
    required this.type,
  });

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return GridView.builder(
      shrinkWrap: true,
      physics: const NeverScrollableScrollPhysics(),
      gridDelegate: const SliverGridDelegateWithFixedCrossAxisCount(
        crossAxisCount:   4,
        crossAxisSpacing: 8,
        mainAxisSpacing:  8,
        childAspectRatio: 1.05,
      ),
      itemCount: networks.length,
      itemBuilder: (_, i) {
        final net     = networks[i];
        final enabled = net.isEnabledFor(type);
        final isSelected = selected == net.code;
        final netColor   = _networkColors[net.code.toUpperCase()] ?? NexusColors.primary;
        final emoji      = _networkEmojis[net.code.toUpperCase()] ?? '📶';

        return Semantics(
          label:    '${net.name} network${enabled ? '' : ', coming soon'}',
          selected: isSelected,
          button:   enabled,
          child: AnimatedContainer(
            duration: const Duration(milliseconds: 200),
            curve: Curves.easeInOut,
            decoration: BoxDecoration(
              color: isSelected
                  ? netColor.withValues(alpha: 0.12)
                  : NexusColors.surface,
              borderRadius: BorderRadius.circular(12),
              border: Border.all(
                color: isSelected
                    ? netColor
                    : (enabled ? Colors.white12 : Colors.white.withValues(alpha: 0.06)),
                width: isSelected ? 2 : 1,
              ),
            ),
            child: Material(
              color:        Colors.transparent,
              borderRadius: BorderRadius.circular(11),
              child: InkWell(
                borderRadius: BorderRadius.circular(11),
                onTap: enabled
                    ? () {
                        HapticFeedback.selectionClick();
                        ref.read(rechargeFormProvider.notifier)
                            .selectNetwork(net.code);
                      }
                    : null,
                child: Opacity(
                  opacity: enabled ? 1.0 : 0.35,
                  child: Column(
                    mainAxisAlignment: MainAxisAlignment.center,
                    children: [
                      Text(emoji, style: const TextStyle(fontSize: 22)),
                      const SizedBox(height: 4),
                      Text(
                        net.name,
                        style: TextStyle(
                          color:      enabled ? Colors.white : Colors.white38,
                          fontSize:   11,
                          fontWeight: isSelected ? FontWeight.w700 : FontWeight.w500,
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
            ),
          ),
        );
      },
    );
  }
}

// ── Airtime amount section ─────────────────────────────────────────────────────

const _presetAmounts = [100, 200, 500, 1000, 2000, 5000];

class _AirtimeSection extends ConsumerStatefulWidget {
  final int? currentAmount;
  const _AirtimeSection({required this.currentAmount});

  @override
  ConsumerState<_AirtimeSection> createState() => _AirtimeSectionState();
}

class _AirtimeSectionState extends ConsumerState<_AirtimeSection> {
  late final TextEditingController _ctrl;

  @override
  void initState() {
    super.initState();
    _ctrl = TextEditingController(
      text: widget.currentAmount != null ? '${widget.currentAmount}' : '',
    );
  }

  @override
  void didUpdateWidget(_AirtimeSection old) {
    super.didUpdateWidget(old);
    if (widget.currentAmount != old.currentAmount &&
        widget.currentAmount != int.tryParse(_ctrl.text)) {
      _ctrl.text = widget.currentAmount != null ? '${widget.currentAmount}' : '';
      _ctrl.selection = TextSelection.collapsed(offset: _ctrl.text.length);
    }
  }

  @override
  void dispose() {
    _ctrl.dispose();
    super.dispose();
  }

  String _fmt(int n) {
    if (n >= 1000 && n % 1000 == 0) return '${n ~/ 1000}k';
    return n.toString();
  }

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        // Presets
        GridView.builder(
          shrinkWrap:  true,
          physics:     const NeverScrollableScrollPhysics(),
          gridDelegate: const SliverGridDelegateWithFixedCrossAxisCount(
            crossAxisCount:   3,
            crossAxisSpacing: 8,
            mainAxisSpacing:  8,
            childAspectRatio: 2.8,
          ),
          itemCount: _presetAmounts.length,
          itemBuilder: (_, i) {
            final val      = _presetAmounts[i];
            final isActive = widget.currentAmount == val;

            return Semantics(
              label:    '₦${_fmt(val)} airtime',
              selected: isActive,
              button:   true,
              child: AnimatedContainer(
                duration: const Duration(milliseconds: 150),
                decoration: BoxDecoration(
                  color: isActive
                      ? NexusColors.primary.withValues(alpha: 0.18)
                      : NexusColors.surface,
                  borderRadius: BorderRadius.circular(10),
                  border: Border.all(
                    color: isActive ? NexusColors.primary : Colors.white12,
                    width: isActive ? 2 : 1,
                  ),
                ),
                child: Material(
                  color:        Colors.transparent,
                  borderRadius: BorderRadius.circular(9),
                  child: InkWell(
                    borderRadius: BorderRadius.circular(9),
                    onTap: () {
                      HapticFeedback.selectionClick();
                      _ctrl.text = val.toString();
                      _ctrl.selection =
                          TextSelection.collapsed(offset: _ctrl.text.length);
                      ref.read(rechargeFormProvider.notifier).setAmount(val);
                    },
                    child: Center(
                      child: Text(
                        '₦${_fmt(val)}',
                        style: TextStyle(
                          color:      isActive ? NexusColors.primary : Colors.white70,
                          fontWeight: FontWeight.w600,
                          fontSize:   13,
                        ),
                      ),
                    ),
                  ),
                ),
              ),
            );
          },
        ),
        const SizedBox(height: 12),

        // Custom amount input
        Semantics(
          label:    'Custom amount input',
          textField: true,
          child: TextField(
            controller:       _ctrl,
            keyboardType:     TextInputType.number,
            textInputAction:  TextInputAction.done,
            inputFormatters:  [FilteringTextInputFormatter.digitsOnly],
            style: const TextStyle(color: Colors.white, fontSize: 16),
            decoration: InputDecoration(
              hintText:  'Enter custom amount',
              hintStyle: const TextStyle(color: Colors.white38),
              prefixText:  '₦  ',
              prefixStyle: const TextStyle(color: Colors.white60, fontSize: 16),
              filled:     true,
              fillColor:  NexusColors.surface,
              border:     OutlineInputBorder(
                borderRadius: BorderRadius.circular(12),
                borderSide:   BorderSide.none,
              ),
              focusedBorder: OutlineInputBorder(
                borderRadius: BorderRadius.circular(12),
                borderSide: const BorderSide(color: NexusColors.primary, width: 1.5),
              ),
            ),
            onChanged: (v) {
              final parsed = int.tryParse(v);
              if (parsed != null) {
                ref.read(rechargeFormProvider.notifier).setAmount(parsed);
              }
            },
          ),
        ),
      ],
    );
  }
}

// ── Data bundles section ──────────────────────────────────────────────────────

class _DataSection extends ConsumerWidget {
  final String? networkCode;
  final DataBundle? selectedBundle;

  const _DataSection({
    required this.networkCode,
    required this.selectedBundle,
  });

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    if (networkCode == null) {
      return const _HintTile(
        icon:  Icons.wifi_find_rounded,
        label: 'Select a network above to see data bundles',
      );
    }

    final bundlesAsync = ref.watch(rechargeBundlesProvider(networkCode!));

    return bundlesAsync.when(
      loading: () => const Padding(
        padding: EdgeInsets.symmetric(vertical: 24),
        child: Center(child: CircularProgressIndicator(color: NexusColors.gold)),
      ),
      error: (_, __) => _HintTile(
        icon:  Icons.error_outline_rounded,
        label: 'Could not load bundles. Tap to retry.',
        onTap: () => ref.refresh(rechargeBundlesProvider(networkCode!)),
      ),
      data: (bundles) {
        if (bundles.isEmpty) {
          return const _HintTile(
            icon:  Icons.signal_cellular_off_rounded,
            label: 'No bundles available for this network.',
          );
        }

        return ListView.separated(
          shrinkWrap: true,
          physics:    const NeverScrollableScrollPhysics(),
          itemCount:  bundles.length,
          separatorBuilder: (_, __) => const SizedBox(height: 8),
          itemBuilder: (_, i) {
            final b          = bundles[i];
            final isSelected = selectedBundle?.id == b.id;

            return Semantics(
              label:    '${b.name}, ₦${b.price}, ${b.validity}',
              selected: isSelected,
              button:   true,
              child: AnimatedContainer(
                duration: const Duration(milliseconds: 150),
                decoration: BoxDecoration(
                  color: isSelected
                      ? NexusColors.primary.withValues(alpha: 0.12)
                      : NexusColors.surface,
                  borderRadius: BorderRadius.circular(12),
                  border: Border.all(
                    color: isSelected ? NexusColors.primary : Colors.white12,
                    width: isSelected ? 2 : 1,
                  ),
                ),
                child: Material(
                  color:        Colors.transparent,
                  borderRadius: BorderRadius.circular(11),
                  child: InkWell(
                    borderRadius: BorderRadius.circular(11),
                    onTap: () {
                      HapticFeedback.selectionClick();
                      ref.read(rechargeFormProvider.notifier).selectBundle(b);
                    },
                    child: Padding(
                      padding: const EdgeInsets.symmetric(
                          horizontal: 16, vertical: 14),
                      child: Row(
                        children: [
                          Icon(
                            Icons.data_usage_rounded,
                            color: isSelected
                                ? NexusColors.primary
                                : Colors.white38,
                            size: 20,
                          ),
                          const SizedBox(width: 12),
                          Expanded(
                            child: Column(
                              crossAxisAlignment: CrossAxisAlignment.start,
                              children: [
                                Text(
                                  b.name,
                                  style: TextStyle(
                                    color: isSelected
                                        ? Colors.white
                                        : Colors.white70,
                                    fontWeight: FontWeight.w600,
                                    fontSize:   14,
                                  ),
                                ),
                                if (b.validity.isNotEmpty) ...[
                                  const SizedBox(height: 2),
                                  Text(
                                    b.validity,
                                    style: const TextStyle(
                                      color:    Colors.white38,
                                      fontSize: 12,
                                    ),
                                  ),
                                ],
                              ],
                            ),
                          ),
                          Text(
                            '₦${b.price}',
                            style: TextStyle(
                              color:      isSelected
                                  ? NexusColors.primary
                                  : NexusColors.gold,
                              fontWeight: FontWeight.w700,
                              fontSize:   15,
                            ),
                          ),
                          const SizedBox(width: 8),
                          Icon(
                            isSelected
                                ? Icons.check_circle_rounded
                                : Icons.radio_button_unchecked_rounded,
                            color: isSelected
                                ? NexusColors.primary
                                : Colors.white24,
                            size: 20,
                          ),
                        ],
                      ),
                    ),
                  ),
                ),
              ),
            );
          },
        );
      },
    );
  }
}

// ── Hint tile ─────────────────────────────────────────────────────────────────

class _HintTile extends StatelessWidget {
  final IconData icon;
  final String label;
  final VoidCallback? onTap;

  const _HintTile({required this.icon, required this.label, this.onTap});

  @override
  Widget build(BuildContext context) {
    return Material(
      color:        NexusColors.surface,
      borderRadius: BorderRadius.circular(12),
      child: InkWell(
        onTap:        onTap,
        borderRadius: BorderRadius.circular(12),
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Column(
            children: [
              Icon(icon, color: Colors.white24, size: 36),
              const SizedBox(height: 10),
              Text(
                label,
                textAlign: TextAlign.center,
                style: const TextStyle(color: Colors.white38, fontSize: 13),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

// ── Summary card ──────────────────────────────────────────────────────────────

class _SummaryCard extends StatelessWidget {
  final RechargeFormState form;
  final List<NetworkOperator> networks;

  const _SummaryCard({required this.form, required this.networks});

  @override
  Widget build(BuildContext context) {
    final net = networks.firstWhere(
      (n) => n.code == form.selectedNetwork,
      orElse: () => const NetworkOperator(
        code: '', name: '—', isActive: false,
        airtimeEnabled: false, dataEnabled: false,
      ),
    );

    final typeLabel  = form.type == RechargeType.airtime ? 'Airtime' : 'Data Bundle';
    final detailLabel = form.type == RechargeType.airtime
        ? 'Airtime'
        : (form.selectedBundle?.name ?? '');
    final amount = form.effectiveAmount;

    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color:        NexusColors.surfaceHigh,
        borderRadius: BorderRadius.circular(16),
        border:       Border.all(color: NexusColors.goldDim),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          _SummaryRow(label: 'Type',    value: typeLabel),
          _SummaryRow(label: 'Network', value: net.name),
          _SummaryRow(label: 'Phone',   value: form.phone),
          if (detailLabel.isNotEmpty)
            _SummaryRow(label: typeLabel, value: detailLabel),
          if (amount != null)
            _SummaryRow(label: 'Amount', value: '₦$amount'),
          const Padding(
            padding: EdgeInsets.only(top: 12),
            child: Divider(color: Colors.white12, height: 1),
          ),
          const SizedBox(height: 12),
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
}

class _SummaryRow extends StatelessWidget {
  final String label;
  final String value;
  const _SummaryRow({required this.label, required this.value});

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 3),
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

// ── Guest nudge ───────────────────────────────────────────────────────────────

class _GuestNudge extends ConsumerWidget {
  const _GuestNudge();

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final isAuth = ref.watch(authStateProvider.select((s) => s.isAuthenticated));
    final isLoading = ref.watch(authStateProvider.select((s) => s.isLoading));
    if (isAuth || isLoading) return const SizedBox.shrink();

    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 12),
      decoration: BoxDecoration(
        color:        NexusColors.surface,
        borderRadius: BorderRadius.circular(12),
        border:       Border.all(color: Colors.white12),
      ),
      child: Row(
        children: [
          const Icon(Icons.info_outline_rounded, color: Colors.white38, size: 18),
          const SizedBox(width: 10),
          const Expanded(
            child: Text(
              'Sign in to earn & track your Pulse Points',
              style: TextStyle(color: Colors.white54, fontSize: 13),
            ),
          ),
          TextButton(
            onPressed: () => context.go('/'),
            style: TextButton.styleFrom(
              foregroundColor: NexusColors.primary,
              minimumSize:     const Size(64, 40),
              padding: const EdgeInsets.symmetric(horizontal: 8),
            ),
            child: const Text(
              'Sign In',
              style: TextStyle(fontSize: 12, fontWeight: FontWeight.w700),
            ),
          ),
        ],
      ),
    );
  }
}

// ── Sticky pay button ─────────────────────────────────────────────────────────

class _StickyPayButton extends ConsumerWidget {
  final RechargeFormState form;
  final List<NetworkOperator> networks;

  const _StickyPayButton({required this.form, required this.networks});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final amount = form.effectiveAmount;
    final label  = form.canProceed && amount != null
        ? 'Pay ₦$amount with Paystack'
        : (form.type == RechargeType.airtime
            ? 'Enter details to continue'
            : 'Select a bundle to continue');

    // Validate network is still active/enabled before allowing submit
    final net = networks.firstWhere(
      (n) => n.code == form.selectedNetwork,
      orElse: () => const NetworkOperator(
        code: '', name: '', isActive: false,
        airtimeEnabled: false, dataEnabled: false,
      ),
    );
    final networkOk = net.isEnabledFor(form.type);
    final canTap    = form.canProceed && networkOk && !form.isSubmitting;

    return SafeArea(
      top: false,
      child: Container(
        padding: const EdgeInsets.fromLTRB(16, 12, 16, 8),
        decoration: BoxDecoration(
          color: NexusColors.surface,
          boxShadow: [
            BoxShadow(
              color:      Colors.black.withValues(alpha: 0.3),
              blurRadius: 8,
              offset:     const Offset(0, -2),
            ),
          ],
        ),
        child: Semantics(
          button: true,
          label:  label,
          child: SizedBox(
            width:  double.infinity,
            height: 52,
            child: FilledButton(
              onPressed: canTap
                  ? () async {
                      FocusScope.of(context).unfocus();
                      final userId = ref.read(
                          authStateProvider.select((s) => s.phoneNumber));
                      final reference = await ref
                          .read(rechargeFormProvider.notifier)
                          .submit(ref.read(dioProvider), userId);
                      if (reference != null && context.mounted) {
                        context.push('/recharge/success', extra: reference);
                      }
                    }
                  : null,
              style: FilledButton.styleFrom(
                backgroundColor:         NexusColors.primary,
                disabledBackgroundColor: Colors.white12,
                foregroundColor:         Colors.white,
                disabledForegroundColor: Colors.white38,
                shape: RoundedRectangleBorder(
                  borderRadius: BorderRadius.circular(14),
                ),
                elevation: 0,
              ),
              child: form.isSubmitting
                  ? const SizedBox(
                      width:  22,
                      height: 22,
                      child: CircularProgressIndicator.adaptive(
                        strokeWidth: 2.5,
                        valueColor: AlwaysStoppedAnimation<Color>(Colors.white),
                      ),
                    )
                  : Text(
                      label,
                      style: const TextStyle(
                        fontWeight: FontWeight.w700,
                        fontSize:   16,
                      ),
                    ),
            ),
          ),
        ),
      ),
    );
  }
}
