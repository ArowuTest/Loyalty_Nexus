import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import '../../../core/api/api_client.dart';
import '../../../core/auth/auth_provider.dart';
import '../../../core/theme/nexus_theme.dart';

// ─── Nigerian states ──────────────────────────────────────────────────────────

const _states = [
  'Abia','Adamawa','Akwa Ibom','Anambra','Bauchi','Bayelsa','Benue','Borno',
  'Cross River','Delta','Ebonyi','Edo','Ekiti','Enugu','FCT Abuja','Gombe',
  'Imo','Jigawa','Kaduna','Kano','Katsina','Kebbi','Kogi','Kwara','Lagos',
  'Nasarawa','Niger','Ogun','Ondo','Osun','Oyo','Plateau','Rivers','Sokoto',
  'Taraba','Yobe','Zamfara',
];

// ─── Screen ───────────────────────────────────────────────────────────────────

class RegisterScreen extends ConsumerStatefulWidget {
  const RegisterScreen({super.key});
  @override ConsumerState<RegisterScreen> createState() => _RegisterScreenState();
}

class _RegisterScreenState extends ConsumerState<RegisterScreen> {
  final _nameCtrl    = TextEditingController();
  String? _state;
  bool    _saving    = false;
  String  _error     = '';
  int     _step      = 0; // 0 = name, 1 = state

  @override
  void dispose() { _nameCtrl.dispose(); super.dispose(); }

  Future<void> _submit() async {
    // Step 0 → validate name
    if (_step == 0) {
      if (_nameCtrl.text.trim().length < 2) {
        setState(() => _error = 'Enter your full name (at least 2 characters)');
        return;
      }
      setState(() { _step = 1; _error = ''; });
      return;
    }

    // Step 1 → validate state + save
    if (_state == null) {
      setState(() => _error = 'Please select your state');
      return;
    }
    setState(() { _saving = true; _error = ''; });
    try {
      await ref.read(userApiProvider).updateProfile(
        fullName: _nameCtrl.text.trim(),
        state:    _state,
      );
      ref.read(authStateProvider.notifier).markOnboarded();
      if (mounted) context.go('/dashboard');
    } on ApiException catch (e) {
      setState(() => _error = e.message);
    } finally {
      if (mounted) setState(() => _saving = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: NexusColors.background,
      body: SafeArea(
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [

            // ── Progress dots ──
            Row(children: [
              _ProgDot(active: _step == 0, done: _step > 0),
              Expanded(child: Container(height: 2, color: _step > 0
                  ? NexusColors.primary : NexusColors.border)),
              _ProgDot(active: _step == 1, done: false),
            ]),
            const SizedBox(height: 36),

            // ── Logo ──
            Row(children: [
              Container(
                width: 44, height: 44,
                decoration: BoxDecoration(
                    gradient: NexusColors.gradientBrand,
                    borderRadius: NexusRadius.md),
                child: const Icon(Icons.bolt_rounded, color: Colors.white, size: 26),
              ),
              const SizedBox(width: 12),
              const Text('Welcome!', style: TextStyle(
                  color: NexusColors.textPrimary, fontSize: 22,
                  fontWeight: FontWeight.w900)),
            ]),
            const SizedBox(height: 8),
            const Text('A few quick details to set up your account.',
                style: TextStyle(color: NexusColors.textSecondary, fontSize: 14)),
            const SizedBox(height: 36),

            // ── Step content ──
            Expanded(child: AnimatedSwitcher(
              duration: const Duration(milliseconds: 280),
              transitionBuilder: (child, anim) => FadeTransition(
                opacity: anim,
                child: SlideTransition(
                  position: Tween<Offset>(
                    begin: const Offset(0.08, 0), end: Offset.zero).animate(anim),
                  child: child,
                ),
              ),
              child: _step == 0
                  ? _NameStep(key: const ValueKey('name'), ctrl: _nameCtrl, error: _error)
                  : _StateStep(
                      key: const ValueKey('state'),
                      selected: _state,
                      error: _error,
                      onSelect: (s) => setState(() { _state = s; _error = ''; }),
                    ),
            )),

            // ── Actions ──
            if (_step == 1) ...[
              OutlinedButton.icon(
                onPressed: _saving ? null : () => setState(() { _step = 0; _error = ''; }),
                icon: const Icon(Icons.arrow_back_rounded, size: 16),
                label: const Text('Back'),
              ),
              const SizedBox(height: 10),
            ],
            ElevatedButton(
              onPressed: _saving ? null : _submit,
              child: _saving
                  ? const SizedBox(width: 22, height: 22,
                      child: CircularProgressIndicator(color: Colors.white, strokeWidth: 2.5))
                  : Text(_step == 0 ? 'Continue →' : 'Finish Setup →',
                      style: const TextStyle(fontSize: 16, fontWeight: FontWeight.w800)),
            ),
            const SizedBox(height: 12),

            // Skip — always available (profile editable from settings)
            Center(child: TextButton(
              onPressed: _saving ? null : () {
                ref.read(authStateProvider.notifier).markOnboarded();
                context.go('/dashboard');
              },
              child: const Text('Skip for now',
                  style: TextStyle(color: NexusColors.textSecondary, fontSize: 13)),
            )),
          ]),
        ),
      ),
    );
  }
}

// ─── Step 1 — Name ────────────────────────────────────────────────────────────

class _NameStep extends StatelessWidget {
  final TextEditingController ctrl;
  final String error;
  const _NameStep({super.key, required this.ctrl, required this.error});

  @override
  Widget build(BuildContext context) => Column(
    crossAxisAlignment: CrossAxisAlignment.start, children: [
    const Text('What\'s your name?', style: TextStyle(
        color: NexusColors.textPrimary, fontSize: 20, fontWeight: FontWeight.w800)),
    const SizedBox(height: 6),
    const Text('This will appear on your Digital Passport.',
        style: TextStyle(color: NexusColors.textSecondary, fontSize: 13)),
    const SizedBox(height: 24),
    TextField(
      controller: ctrl,
      autofocus:  true,
      textCapitalization: TextCapitalization.words,
      style: const TextStyle(color: NexusColors.textPrimary, fontSize: 16),
      decoration: InputDecoration(
        hintText:   'Full name',
        prefixIcon: const Icon(Icons.person_outline_rounded),
        errorText:  error.isNotEmpty ? error : null,
      ),
    ),
  ]);
}

// ─── Step 2 — State ───────────────────────────────────────────────────────────

class _StateStep extends StatefulWidget {
  final String? selected;
  final String  error;
  final ValueChanged<String> onSelect;
  const _StateStep({super.key, required this.selected, required this.error, required this.onSelect});
  @override State<_StateStep> createState() => _StateStepState();
}

class _StateStepState extends State<_StateStep> {
  String _filter = '';
  final _searchCtrl = TextEditingController();

  @override
  void dispose() { _searchCtrl.dispose(); super.dispose(); }

  List<String> get _filtered => _filter.isEmpty
      ? _states
      : _states.where((s) => s.toLowerCase().contains(_filter)).toList();

  @override
  Widget build(BuildContext context) => Column(
    crossAxisAlignment: CrossAxisAlignment.start, children: [
    const Text('Which state are you in?', style: TextStyle(
        color: NexusColors.textPrimary, fontSize: 20, fontWeight: FontWeight.w800)),
    const SizedBox(height: 6),
    const Text('Used for Regional Wars team assignment.',
        style: TextStyle(color: NexusColors.textSecondary, fontSize: 13)),
    const SizedBox(height: 16),
    TextField(
      controller: _searchCtrl,
      onChanged:  (v) => setState(() => _filter = v.toLowerCase()),
      decoration: const InputDecoration(
        hintText:   'Search state…',
        prefixIcon: Icon(Icons.search_rounded),
      ),
    ),
    if (widget.error.isNotEmpty) ...[
      const SizedBox(height: 8),
      Text(widget.error, style: const TextStyle(color: NexusColors.red, fontSize: 12)),
    ],
    const SizedBox(height: 12),
    Expanded(child: ListView.separated(
      itemCount: _filtered.length,
      separatorBuilder: (_, __) => const Divider(color: NexusColors.border, height: 1),
      itemBuilder: (_, i) {
        final s = _filtered[i];
        final selected = s == widget.selected;
        return ListTile(
          dense: true,
          contentPadding: const EdgeInsets.symmetric(horizontal: 8),
          title: Text(s, style: TextStyle(
              color: selected ? NexusColors.primary : NexusColors.textPrimary,
              fontWeight: selected ? FontWeight.w700 : FontWeight.w400,
              fontSize: 14)),
          trailing: selected
              ? const Icon(Icons.check_circle_rounded, color: NexusColors.primary, size: 18)
              : null,
          onTap: () => widget.onSelect(s),
        );
      },
    )),
  ]);
}

// ─── Progress dot ─────────────────────────────────────────────────────────────

class _ProgDot extends StatelessWidget {
  final bool active, done;
  const _ProgDot({required this.active, required this.done});
  @override
  Widget build(BuildContext context) {
    final Color bg = done ? NexusColors.green
        : active ? NexusColors.primary : NexusColors.border;
    return AnimatedContainer(
      duration: const Duration(milliseconds: 300),
      width: 28, height: 28,
      decoration: BoxDecoration(color: bg, shape: BoxShape.circle,
          boxShadow: active ? NexusShadows.glow : null),
      child: Center(child: done
          ? const Icon(Icons.check_rounded, color: Colors.white, size: 14)
          : Icon(active ? Icons.circle : Icons.circle_outlined,
              size: 10, color: active ? Colors.white : NexusColors.textMuted)),
    );
  }
}
