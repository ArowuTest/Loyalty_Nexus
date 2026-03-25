import 'dart:async';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../../core/api/api_client.dart';
import '../../../core/theme/nexus_theme.dart';

final _toolsProvider = FutureProvider.autoDispose<List<dynamic>>((ref) async {
  return ref.read(studioApiProvider).listTools();
});

final _galleryProvider = FutureProvider.autoDispose<List<dynamic>>((ref) async {
  return ref.read(studioApiProvider).getGallery();
});

class StudioScreen extends ConsumerStatefulWidget {
  const StudioScreen({super.key});
  @override ConsumerState<StudioScreen> createState() => _State();
}

class _State extends ConsumerState<StudioScreen>
    with SingleTickerProviderStateMixin {
  late TabController _tabs;
  int _selectedTool = 0;
  final _promptCtrl = TextEditingController();
  bool _generating = false;
  String? _activeGenId;
  String? _genStatus;
  Timer? _pollTimer;

  @override
  void initState() {
    super.initState();
    _tabs = TabController(length: 2, vsync: this);
  }

  @override
  void dispose() {
    _tabs.dispose();
    _promptCtrl.dispose();
    _pollTimer?.cancel();
    super.dispose();
  }

  Future<void> _generate(List<dynamic> tools) async {
    if (_generating || _promptCtrl.text.trim().isEmpty) return;
    final tool = tools[_selectedTool] as Map;
    setState(() { _generating = true; _genStatus = 'Generating…'; });
    try {
      final res = await ref.read(studioApiProvider).startGeneration(
          tool['slug']?.toString() ?? 'image_gen',
          {'prompt': _promptCtrl.text.trim()});
      _activeGenId = res['generation_id']?.toString();
      _pollStatus();
    } on ApiException catch (e) {
      setState(() { _genStatus = 'Error: ${e.message}'; _generating = false; });
    }
  }

  void _pollStatus() {
    if (_activeGenId == null) return;
    _pollTimer?.cancel();
    _pollTimer = Timer.periodic(const Duration(seconds: 3), (_) async {
      if (!mounted) return;
      try {
        final s = await ref.read(studioApiProvider).getGenerationStatus(_activeGenId!);
        final status = s['status']?.toString() ?? 'PENDING';
        setState(() => _genStatus = status);
        if (status == 'COMPLETED' || status == 'FAILED') {
          _pollTimer?.cancel();
          setState(() => _generating = false);
          if (status == 'COMPLETED') {
            ref.invalidate(_galleryProvider);
            _promptCtrl.clear();
          }
        }
      } catch (_) {}
    });
  }

  @override
  Widget build(BuildContext ctx) {
    final toolsAsync = ref.watch(_toolsProvider);
    final galleryAsync = ref.watch(_galleryProvider);

    return Scaffold(
      appBar: AppBar(
        title: const Text('AI Studio ✨'),
        bottom: TabBar(controller: _tabs, tabs: const [
          Tab(text: 'Create'),
          Tab(text: 'Gallery'),
        ]),
      ),
      body: TabBarView(controller: _tabs, children: [
        // ── Create tab ──
        toolsAsync.when(
          loading: () => const Center(child: CircularProgressIndicator(color: NexusColors.primary)),
          error: (e, _) => Center(child: Text('Could not load tools', style: TextStyle(color: NexusColors.textSecondary))),
          data: (tools) => SingleChildScrollView(
            padding: const EdgeInsets.all(20),
            child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
              // Tool picker
              Text('Choose Tool', style: Theme.of(ctx).textTheme.titleMedium),
              const SizedBox(height: 12),
              SizedBox(
                height: 44,
                child: ListView.separated(
                  scrollDirection: Axis.horizontal,
                  itemCount: tools.length,
                  separatorBuilder: (_, __) => const SizedBox(width: 8),
                  itemBuilder: (_, i) {
                    final t = tools[i] as Map;
                    final sel = i == _selectedTool;
                    return GestureDetector(
                      onTap: () => setState(() => _selectedTool = i),
                      child: Container(
                        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
                        decoration: BoxDecoration(
                          color: sel ? NexusColors.primary : NexusColors.surface,
                          borderRadius: BorderRadius.circular(22),
                          border: Border.all(
                              color: sel ? NexusColors.primary : NexusColors.border)),
                        child: Text(t['name']?.toString() ?? '—',
                            style: TextStyle(
                                color: sel ? Colors.white : NexusColors.textPrimary,
                                fontSize: 13)),
                      ),
                    );
                  },
                ),
              ),
              const SizedBox(height: 20),
              // Prompt input
              Text('Prompt', style: Theme.of(ctx).textTheme.titleMedium),
              const SizedBox(height: 8),
              TextField(
                controller: _promptCtrl,
                maxLines: 4,
                decoration: InputDecoration(
                  hintText: 'Describe what you want to create…',
                  filled: true,
                  fillColor: NexusColors.surface,
                  border: OutlineInputBorder(borderRadius: BorderRadius.circular(12),
                      borderSide: const BorderSide(color: NexusColors.border)),
                ),
                style: const TextStyle(color: NexusColors.textPrimary),
              ),
              const SizedBox(height: 16),
              if (_genStatus != null)
                Container(
                  width: double.infinity,
                  padding: const EdgeInsets.all(12),
                  decoration: BoxDecoration(
                    color: NexusColors.surface,
                    borderRadius: BorderRadius.circular(10),
                    border: Border.all(color: NexusColors.border)),
                  child: Row(children: [
                    if (_generating) ...[
                      const SizedBox(width: 20, height: 20,
                        child: CircularProgressIndicator(color: NexusColors.primary, strokeWidth: 2)),
                      const SizedBox(width: 12),
                    ],
                    Text(_genStatus!, style: const TextStyle(color: NexusColors.textSecondary)),
                  ]),
                ),
              const SizedBox(height: 16),
              SizedBox(
                width: double.infinity,
                child: ElevatedButton.icon(
                  onPressed: _generating ? null : () => _generate(tools),
                  icon: const Icon(Icons.auto_awesome),
                  label: Text(_generating ? 'Generating…' : 'Generate'),
                ),
              ),
            ]),
          ),
        ),

        // ── Gallery tab ──
        galleryAsync.when(
          loading: () => const Center(child: CircularProgressIndicator(color: NexusColors.primary)),
          error: (_, __) => const Center(child: Text('Gallery unavailable')),
          data: (items) => items.isEmpty
            ? Center(child: Column(mainAxisAlignment: MainAxisAlignment.center, children: [
                const Icon(Icons.photo_library_outlined, size: 64, color: NexusColors.textSecondary),
                const SizedBox(height: 12),
                Text('No generations yet', style: TextStyle(color: NexusColors.textSecondary)),
              ]))
            : GridView.builder(
                padding: const EdgeInsets.all(16),
                gridDelegate: const SliverGridDelegateWithFixedCrossAxisCount(
                    crossAxisCount: 2, crossAxisSpacing: 12, mainAxisSpacing: 12),
                itemCount: items.length,
                itemBuilder: (_, i) {
                  final g = items[i] as Map;
                  return Container(
                    decoration: BoxDecoration(
                      color: NexusColors.surface,
                      borderRadius: BorderRadius.circular(12),
                      border: Border.all(color: NexusColors.border),
                    ),
                    child: Column(children: [
                      Expanded(child: ClipRRect(
                        borderRadius: const BorderRadius.vertical(top: Radius.circular(12)),
                        child: g['output_url'] != null
                          ? Image.network(g['output_url'].toString(),
                              fit: BoxFit.cover,
                              errorBuilder: (_, __, ___) => const Icon(
                                Icons.broken_image, color: NexusColors.textSecondary))
                          : const Center(child: Icon(Icons.image_outlined,
                              color: NexusColors.textSecondary)),
                      )),
                      Padding(
                        padding: const EdgeInsets.all(8),
                        child: Text(g['tool_name']?.toString() ?? '—',
                            style: const TextStyle(
                                color: NexusColors.textSecondary, fontSize: 11),
                            maxLines: 1, overflow: TextOverflow.ellipsis),
                      ),
                    ]),
                  );
                },
              ),
        ),
      ]),
    );
  }
}
