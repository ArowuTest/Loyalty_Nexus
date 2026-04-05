// ─── VideoScript Template ──────────────────────────────────────────────────────
// Mirrors webapp VideoScript.tsx exactly.
// Supports: video-script
// Workflow:
//  1. Define characters (name + appearance + voice note)
//  2. Write scene-by-scene script (background image + dialogue + direction)
//  3. Choose visual style (cinematic, anime, cartoon, realistic, 3D, storybook)
//  4. Set duration and aspect ratio
//  5. Generate — compiles script into structured prompt + image list for Kling v2.6
//
// Payload:
//  - prompt: compiled story synopsis + character descriptions + style
//  - extra_params.image_urls: array of scene background CDN URLs
//  - extra_params.scene_N_caption: per-scene compiled dialogue + direction
//  - extra_params.visual_style, generate_audio: true
//  - duration, aspect_ratio: standard fields

import 'dart:io';
import 'package:flutter/material.dart';
import 'package:image_picker/image_picker.dart';
import 'template_types.dart';
import '../../../core/api/api_client.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

// ── Constants ──────────────────────────────────────────────────────────────────

const _kNarratorId = '__narrator__';

const _visualStyles = [
  {'value': 'cinematic',  'label': 'Cinematic',  'desc': 'Realistic, film-quality',   'emoji': '🎬'},
  {'value': 'anime',      'label': 'Anime',      'desc': 'Japanese animation style',  'emoji': '🌸'},
  {'value': 'cartoon',    'label': 'Cartoon',    'desc': 'Colourful, expressive',      'emoji': '🎨'},
  {'value': 'realistic',  'label': 'Realistic',  'desc': 'Photorealistic rendering',  'emoji': '📷'},
  {'value': '3d',         'label': '3D Render',  'desc': 'CGI / Pixar-style',         'emoji': '🎭'},
  {'value': 'storybook',  'label': 'Storybook',  'desc': 'Illustrated, painterly',    'emoji': '📖'},
];

const _aspectRatios = [
  {'value': '16:9', 'label': 'Landscape', 'icon': '🖥️'},
  {'value': '9:16', 'label': 'Portrait',  'icon': '📱'},
  {'value': '1:1',  'label': 'Square',    'icon': '⬜'},
];

const _durationOptions = [5, 8, 10, 15];

const _exampleCharacters = [
  {
    'id': 'c1',
    'name': 'Amara',
    'appearance': 'A confident young Nigerian woman in a bright yellow dress, natural hair',
    'voiceNote': 'Speaks warmly and with authority',
  },
  {
    'id': 'c2',
    'name': 'Emeka',
    'appearance': 'A tall man in a traditional agbada, mid-30s, friendly smile',
    'voiceNote': 'Calm and thoughtful',
  },
];

// ── Data models ────────────────────────────────────────────────────────────────

class _Character {
  String id;
  String name;
  String appearance;
  String voiceNote;

  _Character({
    required this.id,
    this.name = '',
    this.appearance = '',
    this.voiceNote = '',
  });

  _Character copyWith({String? name, String? appearance, String? voiceNote}) =>
      _Character(
        id: id,
        name: name ?? this.name,
        appearance: appearance ?? this.appearance,
        voiceNote: voiceNote ?? this.voiceNote,
      );
}

class _DialogueLine {
  String characterId;
  String text;

  _DialogueLine(this.characterId, this.text);

  /// Convenience factory: narrator line with empty text (default state).
  factory _DialogueLine.empty() => _DialogueLine(_kNarratorId, '');
}

class _SceneSlot {
  final String id;
  File?   imageFile;
  String? imageUrl;     // pasted URL
  String  uploadedUrl;  // CDN URL after upload
  String  direction;
  List<_DialogueLine> dialogue;

  _SceneSlot({
    required this.id,
    required this.imageFile,
    required this.imageUrl,
    required this.uploadedUrl,
    required this.direction,
    required this.dialogue,
  });

  /// Convenience factory: empty scene with default values.
  factory _SceneSlot.empty(String id) => _SceneSlot(
    id: id,
    imageFile: null,
    imageUrl: null,
    uploadedUrl: '',
    direction: '',
    dialogue: [_DialogueLine.empty()],
  );

  bool get hasImage => imageFile != null || (imageUrl?.isNotEmpty ?? false) || uploadedUrl.isNotEmpty;
  bool get hasContent => hasImage || direction.isNotEmpty || dialogue.any((l) => l.text.isNotEmpty);
}

// _SceneSlot.empty(id) is the canonical way to create an empty scene.

// ── Widget ─────────────────────────────────────────────────────────────────────

class VideoScriptTemplate extends ConsumerStatefulWidget {
  final TemplateProps props;
  const VideoScriptTemplate({super.key, required this.props});

  @override
  ConsumerState<VideoScriptTemplate> createState() => _VideoScriptTemplateState();
}

class _VideoScriptTemplateState extends ConsumerState<VideoScriptTemplate> {
  final _synopsisCtrl = TextEditingController();

  // Story-level
  String _visualStyle = 'cinematic';
  String _aspectRatio = '16:9';
  int    _duration    = 10;

  // Characters
  final List<_Character> _characters = [];
  bool _showChars = true;
  bool _usedExamples = false;

  // Scenes
  late List<_SceneSlot> _scenes;
  String? _expandedSceneId;
  bool    _uploading = false;

  @override
  void initState() {
    super.initState();
    _scenes = [_SceneSlot.empty('s1'), _SceneSlot.empty('s2')];
    _expandedSceneId = 's1';
  }

  @override
  void dispose() {
    _synopsisCtrl.dispose();
    super.dispose();
  }

  // ── Character helpers ────────────────────────────────────────────────────────

  void _addCharacter() {
    if (_characters.length >= 5) return;
    setState(() => _characters.add(_Character(id: DateTime.now().millisecondsSinceEpoch.toString())));
  }

  void _updateCharacter(String id, {String? name, String? appearance, String? voiceNote}) {
    setState(() {
      final idx = _characters.indexWhere((c) => c.id == id);
      if (idx >= 0) {
        _characters[idx] = _characters[idx].copyWith(
          name: name, appearance: appearance, voiceNote: voiceNote,
        );
      }
    });
  }

  void _removeCharacter(String id) {
    setState(() {
      _characters.removeWhere((c) => c.id == id);
      // Replace removed character's lines with narrator
      for (final scene in _scenes) {
        for (final line in scene.dialogue) {
          if (line.characterId == id) line.characterId = _kNarratorId;
        }
      }
    });
  }

  void _loadExamples() {
    setState(() {
      _characters.clear();
      for (final ex in _exampleCharacters) {
        _characters.add(_Character(
          id: ex['id']!,
          name: ex['name']!,
          appearance: ex['appearance']!,
          voiceNote: ex['voiceNote']!,
        ));
      }
      _usedExamples = true;
    });
  }

  // ── Scene helpers ────────────────────────────────────────────────────────────

  void _addScene() {
    if (_scenes.length >= 6) return;
    final id = 's${DateTime.now().millisecondsSinceEpoch}';
    setState(() {
      _scenes.add(_SceneSlot.empty(id));
      _expandedSceneId = id;
    });
  }

  void _removeScene(String id) {
    if (_scenes.length <= 2) return;
    setState(() => _scenes.removeWhere((s) => s.id == id));
  }

  // ── Dialogue helpers ─────────────────────────────────────────────────────────

  void _addDialogueLine(String sceneId) {
    setState(() {
      final scene = _scenes.firstWhere((s) => s.id == sceneId);
      scene.dialogue.add(_DialogueLine.empty());
    });
  }

  void _removeDialogueLine(String sceneId, int lineIdx) {
    setState(() {
      final scene = _scenes.firstWhere((s) => s.id == sceneId);
      if (scene.dialogue.length > 1) scene.dialogue.removeAt(lineIdx);
    });
  }

  // ── Image upload ─────────────────────────────────────────────────────────────

  Future<void> _pickSceneImage(String sceneId) async {
    final picker = ImagePicker();
    final picked = await picker.pickImage(source: ImageSource.gallery, imageQuality: 85);
    if (picked == null) return;
    setState(() {
      final scene = _scenes.firstWhere((s) => s.id == sceneId);
      scene.imageFile = File(picked.path);
      scene.imageUrl  = null;
      scene.uploadedUrl = '';
    });
  }

  void _clearSceneImage(String sceneId) {
    setState(() {
      final scene = _scenes.firstWhere((s) => s.id == sceneId);
      scene.imageFile   = null;
      scene.imageUrl    = null;
      scene.uploadedUrl = '';
    });
  }

  // ── Prompt compilation ───────────────────────────────────────────────────────

  String _compileSceneCaption(_SceneSlot scene) {
    final parts = <String>[];
    if (scene.direction.isNotEmpty) parts.add('[Setting: ${scene.direction}]');
    for (final line in scene.dialogue) {
      if (line.text.trim().isEmpty) continue;
      if (line.characterId == _kNarratorId) {
        parts.add(line.text.trim());
      } else {
        final char = _characters.firstWhere(
          (c) => c.id == line.characterId,
          orElse: () => _Character(id: '', name: 'Character'),
        );
        parts.add('${char.name}: "${line.text.trim()}"');
      }
    }
    return parts.join(' ');
  }

  String _compileFullPrompt() {
    final styleLabel = _visualStyles
        .firstWhere((s) => s['value'] == _visualStyle, orElse: () => {'label': _visualStyle})['label']!;
    final charDescs = _characters.map((c) => '${c.name} (${c.appearance})').join(', ');
    final parts = <String>[];
    final synopsis = _synopsisCtrl.text.trim();
    if (synopsis.isNotEmpty) parts.add(synopsis);
    if (charDescs.isNotEmpty) parts.add('Characters: $charDescs');
    parts.add('Visual style: $styleLabel animation');
    return parts.join('. ');
  }

  // ── Submit ───────────────────────────────────────────────────────────────────

  Future<void> _handleSubmit() async {
    final p = widget.props;
    if (!p.canAfford || p.isLoading || _uploading) return;
    final filledScenes = _scenes.where((s) => s.hasImage).toList();
    if (filledScenes.length < 2) return;

    setState(() => _uploading = true);
    final imageUrls   = <String>[];
    final extraParams = <String, dynamic>{};

    try {
      final studioApi = ref.read(studioApiProvider);
      int sceneIdx = 0;
      for (final scene in _scenes) {
        if (!scene.hasImage) continue;
        String url = scene.uploadedUrl.isNotEmpty
            ? scene.uploadedUrl
            : (scene.imageUrl ?? '');
        if (scene.imageFile != null && scene.uploadedUrl.isEmpty) {
          url = await studioApi.uploadAsset(scene.imageFile!);
          setState(() => scene.uploadedUrl = url);
        }
        imageUrls.add(url);
        final caption = _compileSceneCaption(scene);
        if (caption.isNotEmpty) {
          extraParams['scene_${sceneIdx + 1}_caption'] = caption;
        }
        sceneIdx++;
      }
    } catch (_) {
      setState(() => _uploading = false);
      return;
    }
    setState(() => _uploading = false);

    p.onSubmit(GeneratePayload(
      prompt:      _compileFullPrompt(),
      duration:    _duration,
      aspectRatio: _aspectRatio,
      extraParams: {
        'image_urls':     imageUrls,
        'visual_style':   _visualStyle,
        'generate_audio': true,
        ...extraParams,
      },
    ));
  }

  // ── Build ────────────────────────────────────────────────────────────────────

  @override
  Widget build(BuildContext context) {
    final p = widget.props;
    final filledScenes = _scenes.where((s) => s.hasImage).length;
    final isValid = filledScenes >= 2 && !_uploading;

    final allCharacters = [
      {'id': _kNarratorId, 'name': 'Narrator / Direction'},
      ..._characters.map((c) => {'id': c.id, 'name': c.name.isEmpty ? 'Character' : c.name}),
    ];

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        // ── Provider badge ──
        const ProviderBadge(
          label: 'Kling v2.6 Pro',
          description: 'Script-driven story animation',
          color: Color(0xFF7C3AED),
          icon: Icons.movie_creation_rounded,
        ),
        const SizedBox(height: 16),

        // ── How it works banner ──
        Container(
          padding: const EdgeInsets.all(12),
          decoration: BoxDecoration(
            color: const Color(0xFF7C3AED).withValues(alpha: 0.08),
            borderRadius: BorderRadius.circular(12),
            border: Border.all(color: const Color(0xFF7C3AED).withValues(alpha: 0.2)),
          ),
          child: Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              const Icon(Icons.movie_filter_rounded, size: 14, color: Color(0xFFA78BFA)),
              const SizedBox(width: 10),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    const Text('Script-Driven Animation',
                      style: TextStyle(color: Colors.white, fontSize: 13, fontWeight: FontWeight.w600)),
                    const SizedBox(height: 2),
                    Text(
                      'Define your characters, write a scene-by-scene script with dialogue, upload background images, and the AI animates your story into a video.',
                      style: TextStyle(color: Colors.white.withValues(alpha: 0.5), fontSize: 11, height: 1.4),
                    ),
                  ],
                ),
              ),
            ],
          ),
        ),
        const SizedBox(height: 16),

        // ── Story Synopsis ──
        buildSectionLabel('Story Synopsis (optional)'),
        buildTextArea(
          controller: _synopsisCtrl,
          placeholder: 'A heartwarming story about two childhood friends who reunite after 10 years in Lagos…',
          maxLines: 2,
          maxLength: 500,
        ),
        const SizedBox(height: 16),

        // ── Characters section ──
        _buildCharactersSection(allCharacters),
        const SizedBox(height: 16),

        // ── Scenes section ──
        _buildScenesSection(allCharacters),
        const SizedBox(height: 16),

        // ── Visual Style ──
        buildSectionLabel('Visual Style'),
        GridView.count(
          crossAxisCount: 3,
          shrinkWrap: true,
          physics: const NeverScrollableScrollPhysics(),
          crossAxisSpacing: 8,
          mainAxisSpacing: 8,
          childAspectRatio: 1.3,
          children: _visualStyles.map((style) {
            final selected = _visualStyle == style['value'];
            return GestureDetector(
              onTap: () => setState(() => _visualStyle = style['value']!),
              child: Container(
                padding: const EdgeInsets.symmetric(vertical: 8, horizontal: 6),
                decoration: BoxDecoration(
                  color: selected
                      ? const Color(0xFF7C3AED).withValues(alpha: 0.12)
                      : Colors.white.withValues(alpha: 0.02),
                  borderRadius: BorderRadius.circular(12),
                  border: Border.all(
                    color: selected
                        ? const Color(0xFF7C3AED).withValues(alpha: 0.5)
                        : Colors.white.withValues(alpha: 0.08),
                  ),
                ),
                child: Column(
                  mainAxisAlignment: MainAxisAlignment.center,
                  children: [
                    Text(style['emoji']!, style: const TextStyle(fontSize: 18)),
                    const SizedBox(height: 2),
                    Text(style['label']!,
                      style: TextStyle(
                        color: selected ? const Color(0xFFC4B5FD) : Colors.white.withValues(alpha: 0.5),
                        fontSize: 11, fontWeight: FontWeight.w600,
                      ),
                      textAlign: TextAlign.center,
                    ),
                    Text(style['desc']!,
                      style: TextStyle(color: Colors.white.withValues(alpha: 0.25), fontSize: 9),
                      textAlign: TextAlign.center,
                      maxLines: 1, overflow: TextOverflow.ellipsis,
                    ),
                  ],
                ),
              ),
            );
          }).toList(),
        ),
        const SizedBox(height: 16),

        // ── Aspect Ratio ──
        buildSectionLabel('Aspect Ratio'),
        Row(
          children: _aspectRatios.map((ar) {
            final selected = _aspectRatio == ar['value'];
            return Expanded(
              child: Padding(
                padding: const EdgeInsets.only(right: 8),
                child: buildChip(
                  label: '${ar['icon']} ${ar['label']}',
                  selected: selected,
                  onTap: () => setState(() => _aspectRatio = ar['value']!),
                  activeColor: const Color(0xFF7C3AED),
                ),
              ),
            );
          }).toList(),
        ),
        const SizedBox(height: 16),

        // ── Duration ──
        buildSectionLabel('Duration'),
        Row(
          children: _durationOptions.map((d) => Padding(
            padding: const EdgeInsets.only(right: 8),
            child: buildChip(
              label: '${d}s',
              selected: _duration == d,
              onTap: () => setState(() => _duration = d),
              activeColor: const Color(0xFF7C3AED),
            ),
          )).toList(),
        ),
        const SizedBox(height: 24),

        // ── Validation hint ──
        if (!isValid && !_uploading) ...[
          Center(
            child: Text(
              'Add background images to at least 2 scenes to generate',
              style: TextStyle(color: Colors.amber.withValues(alpha: 0.7), fontSize: 12),
              textAlign: TextAlign.center,
            ),
          ),
          const SizedBox(height: 12),
        ],

        // ── Generate button ──
        buildGenerateButton(
          label: _uploading
              ? 'Uploading scenes…'
              : p.isLoading
                  ? 'Animating your story…'
                  : 'Animate Story${p.pointCost > 0 ? ' · ${p.pointCost} pts' : ''}',
          enabled: isValid && p.canAfford,
          isLoading: p.isLoading || _uploading,
          onTap: _handleSubmit,
          gradientColors: const [Color(0xFF6D28D9), Color(0xFF7C3AED)],
          icon: Icons.auto_awesome_rounded,
        ),

        if (!p.canAfford) ...[
          const SizedBox(height: 8),
          Center(
            child: Text(
              'You need ${p.pointCost} Pulse Points to use this tool',
              style: TextStyle(color: Colors.red.withValues(alpha: 0.7), fontSize: 12),
            ),
          ),
        ],
      ],
    );
  }

  // ── Characters section widget ────────────────────────────────────────────────

  Widget _buildCharactersSection(List<Map<String, String>> allCharacters) {
    return Container(
      decoration: BoxDecoration(
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: Colors.white.withValues(alpha: 0.1)),
      ),
      child: Column(
        children: [
          // Header
          GestureDetector(
            onTap: () => setState(() => _showChars = !_showChars),
            child: Container(
              padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 12),
              child: Row(
                children: [
                  const Icon(Icons.people_outline_rounded, size: 14, color: Color(0xFFA78BFA)),
                  const SizedBox(width: 8),
                  Expanded(
                    child: Row(
                      children: [
                        Text('Characters',
                          style: TextStyle(color: Colors.white.withValues(alpha: 0.7), fontSize: 13, fontWeight: FontWeight.w500)),
                        if (_characters.isNotEmpty) ...[
                          const SizedBox(width: 8),
                          Text('${_characters.length} defined',
                            style: const TextStyle(color: Color(0xFFA78BFA), fontSize: 11)),
                        ],
                      ],
                    ),
                  ),
                  Icon(_showChars ? Icons.keyboard_arrow_up_rounded : Icons.keyboard_arrow_down_rounded,
                    size: 16, color: Colors.white.withValues(alpha: 0.3)),
                ],
              ),
            ),
          ),

          // Body
          if (_showChars) ...[
            Divider(height: 1, color: Colors.white.withValues(alpha: 0.08)),
            Padding(
              padding: const EdgeInsets.all(12),
              child: Column(
                children: [
                  if (_characters.isEmpty) ...[
                    Padding(
                      padding: const EdgeInsets.symmetric(vertical: 8),
                      child: Column(
                        children: [
                          Text('No characters yet. Add your own or load examples.',
                            style: TextStyle(color: Colors.white.withValues(alpha: 0.3), fontSize: 12),
                            textAlign: TextAlign.center),
                          const SizedBox(height: 12),
                          Row(
                            mainAxisAlignment: MainAxisAlignment.center,
                            children: [
                              _SmallButton(
                                label: 'Add Character',
                                icon: Icons.add_rounded,
                                color: const Color(0xFF7C3AED),
                                onTap: _addCharacter,
                              ),
                              const SizedBox(width: 8),
                              if (!_usedExamples)
                                _SmallButton(
                                  label: 'Load Examples',
                                  icon: Icons.auto_awesome_rounded,
                                  color: Colors.white.withValues(alpha: 0.3),
                                  onTap: _loadExamples,
                                ),
                            ],
                          ),
                        ],
                      ),
                    ),
                  ] else ...[
                    ..._characters.asMap().entries.map((entry) {
                      final idx  = entry.key;
                      final char = entry.value;
                      return Padding(
                        padding: const EdgeInsets.only(bottom: 10),
                        child: _CharacterCard(
                          index: idx,
                          character: char,
                          onUpdate: (name, appearance, voiceNote) =>
                              _updateCharacter(char.id, name: name, appearance: appearance, voiceNote: voiceNote),
                          onRemove: () => _removeCharacter(char.id),
                        ),
                      );
                    }),
                    if (_characters.length < 5)
                      GestureDetector(
                        onTap: _addCharacter,
                        child: Container(
                          width: double.infinity,
                          padding: const EdgeInsets.symmetric(vertical: 10),
                          decoration: BoxDecoration(
                            borderRadius: BorderRadius.circular(10),
                            border: Border.all(
                              color: Colors.white.withValues(alpha: 0.12),
                              style: BorderStyle.solid,
                            ),
                          ),
                          child: Row(
                            mainAxisAlignment: MainAxisAlignment.center,
                            children: [
                              Icon(Icons.add_rounded, size: 14, color: Colors.white.withValues(alpha: 0.35)),
                              const SizedBox(width: 6),
                              Text('Add Another Character',
                                style: TextStyle(color: Colors.white.withValues(alpha: 0.35), fontSize: 12)),
                            ],
                          ),
                        ),
                      ),
                  ],
                ],
              ),
            ),
          ],
        ],
      ),
    );
  }

  // ── Scenes section widget ────────────────────────────────────────────────────

  Widget _buildScenesSection(List<Map<String, String>> allCharacters) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          mainAxisAlignment: MainAxisAlignment.spaceBetween,
          children: [
            buildSectionLabel('Scenes (${_scenes.length}/6)'),
            Text('Upload a background image per scene',
              style: TextStyle(color: Colors.white.withValues(alpha: 0.25), fontSize: 10)),
          ],
        ),
        const SizedBox(height: 8),
        ..._scenes.asMap().entries.map((entry) {
          final idx   = entry.key;
          final scene = entry.value;
          final isExpanded = _expandedSceneId == scene.id;
          return Padding(
            padding: const EdgeInsets.only(bottom: 8),
            child: _SceneCard(
              scene: scene,
              sceneIndex: idx,
              isExpanded: isExpanded,
              canRemove: _scenes.length > 2,
              allCharacters: allCharacters,
              onToggle: () => setState(() =>
                _expandedSceneId = isExpanded ? null : scene.id),
              onRemove: () => _removeScene(scene.id),
              onPickImage: () => _pickSceneImage(scene.id),
              onClearImage: () => _clearSceneImage(scene.id),
              onUpdateDirection: (v) => setState(() => scene.direction = v),
              onUpdateImageUrl: (v) => setState(() => scene.imageUrl = v),
              onAddDialogueLine: () => _addDialogueLine(scene.id),
              onRemoveDialogueLine: (i) => _removeDialogueLine(scene.id, i),
              onUpdateDialogueLine: (i, charId, text) => setState(() {
                scene.dialogue[i].characterId = charId ?? scene.dialogue[i].characterId;
                scene.dialogue[i].text = text ?? scene.dialogue[i].text;
              }),
            ),
          );
        }),
        if (_scenes.length < 6)
          GestureDetector(
            onTap: _addScene,
            child: Container(
              width: double.infinity,
              padding: const EdgeInsets.symmetric(vertical: 14),
              decoration: BoxDecoration(
                borderRadius: BorderRadius.circular(12),
                border: Border.all(
                  color: Colors.white.withValues(alpha: 0.1),
                  style: BorderStyle.solid,
                ),
              ),
              child: Row(
                mainAxisAlignment: MainAxisAlignment.center,
                children: [
                  Icon(Icons.add_rounded, size: 16, color: Colors.white.withValues(alpha: 0.35)),
                  const SizedBox(width: 6),
                  Text('Add Scene',
                    style: TextStyle(color: Colors.white.withValues(alpha: 0.35), fontSize: 13)),
                ],
              ),
            ),
          ),
      ],
    );
  }
}

// ── _CharacterCard ─────────────────────────────────────────────────────────────

class _CharacterCard extends StatelessWidget {
  final int _index;
  final _Character _character;
  final void Function(String? name, String? appearance, String? voiceNote) _onUpdate;
  final VoidCallback _onRemove;

  const _CharacterCard({
    required int index,
    required _Character character,
    required void Function(String?, String?, String?) onUpdate,
    required VoidCallback onRemove,
  }) : _index = index, _character = character, _onUpdate = onUpdate, _onRemove = onRemove;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(10),
      decoration: BoxDecoration(
        color: Colors.white.withValues(alpha: 0.03),
        borderRadius: BorderRadius.circular(10),
        border: Border.all(color: Colors.white.withValues(alpha: 0.08)),
      ),
      child: Column(
        children: [
          Row(
            children: [
              Container(
                width: 24, height: 24,
                decoration: BoxDecoration(
                  color: const Color(0xFF7C3AED).withValues(alpha: 0.3),
                  shape: BoxShape.circle,
                ),
                alignment: Alignment.center,
                child: Text('${_index + 1}',
                  style: const TextStyle(color: Color(0xFFC4B5FD), fontSize: 10, fontWeight: FontWeight.bold)),
              ),
              const SizedBox(width: 8),
              Expanded(
                child: TextFormField(
                  initialValue: _character.name,
                  onChanged: (v) => _onUpdate(v, null, null),
                  style: const TextStyle(color: Colors.white, fontSize: 12),
                  decoration: InputDecoration(
                    hintText: 'Character name (e.g. Amara)',
                    hintStyle: TextStyle(color: Colors.white.withValues(alpha: 0.25), fontSize: 12),
                    isDense: true,
                    contentPadding: const EdgeInsets.symmetric(horizontal: 10, vertical: 8),
                    filled: true,
                    fillColor: Colors.white.withValues(alpha: 0.05),
                    border: OutlineInputBorder(
                      borderRadius: BorderRadius.circular(8),
                      borderSide: BorderSide(color: Colors.white.withValues(alpha: 0.1)),
                    ),
                    enabledBorder: OutlineInputBorder(
                      borderRadius: BorderRadius.circular(8),
                      borderSide: BorderSide(color: Colors.white.withValues(alpha: 0.1)),
                    ),
                    focusedBorder: OutlineInputBorder(
                      borderRadius: BorderRadius.circular(8),
                      borderSide: const BorderSide(color: Color(0xFF7C3AED)),
                    ),
                  ),
                ),
              ),
              const SizedBox(width: 6),
              GestureDetector(
                onTap: _onRemove,
                child: Icon(Icons.delete_outline_rounded, size: 16, color: Colors.white.withValues(alpha: 0.25)),
              ),
            ],
          ),
          const SizedBox(height: 8),
          _buildField(
            value: _character.appearance,
            hint: 'Appearance: e.g. A tall woman in a yellow dress with natural hair',
            onChanged: (v) => _onUpdate(null, v, null),
          ),
          const SizedBox(height: 6),
          _buildField(
            value: _character.voiceNote,
            hint: 'Voice / personality note (optional): e.g. Speaks warmly',
            onChanged: (v) => _onUpdate(null, null, v),
          ),
        ],
      ),
    );
  }

  Widget _buildField({required String value, required String hint, required ValueChanged<String> onChanged}) {
    return TextFormField(
      initialValue: value,
      onChanged: onChanged,
      style: const TextStyle(color: Colors.white, fontSize: 11),
      decoration: InputDecoration(
        hintText: hint,
        hintStyle: TextStyle(color: Colors.white.withValues(alpha: 0.25), fontSize: 11),
        isDense: true,
        contentPadding: const EdgeInsets.symmetric(horizontal: 10, vertical: 8),
        filled: true,
        fillColor: Colors.white.withValues(alpha: 0.05),
        border: OutlineInputBorder(
          borderRadius: BorderRadius.circular(8),
          borderSide: BorderSide(color: Colors.white.withValues(alpha: 0.1)),
        ),
        enabledBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(8),
          borderSide: BorderSide(color: Colors.white.withValues(alpha: 0.1)),
        ),
        focusedBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(8),
          borderSide: const BorderSide(color: Color(0xFF7C3AED)),
        ),
      ),
    );
  }
}

// ── _SceneCard ─────────────────────────────────────────────────────────────────

class _SceneCard extends StatelessWidget {
  final _SceneSlot scene;
  final int sceneIndex;
  final bool isExpanded;
  final bool canRemove;
  final List<Map<String, String>> allCharacters;
  final VoidCallback onToggle;
  final VoidCallback onRemove;
  final VoidCallback onPickImage;
  final VoidCallback onClearImage;
  final ValueChanged<String> onUpdateDirection;
  final ValueChanged<String> onUpdateImageUrl;
  final VoidCallback onAddDialogueLine;
  final ValueChanged<int> onRemoveDialogueLine;
  final void Function(int idx, String? charId, String? text) onUpdateDialogueLine;

  const _SceneCard({
    required this.scene,
    required this.sceneIndex,
    required this.isExpanded,
    required this.canRemove,
    required this.allCharacters,
    required this.onToggle,
    required this.onRemove,
    required this.onPickImage,
    required this.onClearImage,
    required this.onUpdateDirection,
    required this.onUpdateImageUrl,
    required this.onAddDialogueLine,
    required this.onRemoveDialogueLine,
    required this.onUpdateDialogueLine,
  });

  @override
  Widget build(BuildContext context) {
    return Container(
      decoration: BoxDecoration(
        color: scene.hasContent
            ? const Color(0xFF7C3AED).withValues(alpha: 0.04)
            : Colors.white.withValues(alpha: 0.02),
        borderRadius: BorderRadius.circular(12),
        border: Border.all(
          color: scene.hasContent
              ? const Color(0xFF7C3AED).withValues(alpha: 0.25)
              : Colors.white.withValues(alpha: 0.1),
        ),
      ),
      child: Column(
        children: [
          // ── Scene header ──
          GestureDetector(
            onTap: onToggle,
            child: Padding(
              padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
              child: Row(
                children: [
                  Container(
                    width: 20, height: 20,
                    decoration: BoxDecoration(
                      color: scene.hasContent
                          ? const Color(0xFF7C3AED).withValues(alpha: 0.4)
                          : Colors.white.withValues(alpha: 0.08),
                      shape: BoxShape.circle,
                    ),
                    alignment: Alignment.center,
                    child: Text('${sceneIndex + 1}',
                      style: TextStyle(
                        color: scene.hasContent ? const Color(0xFFC4B5FD) : Colors.white.withValues(alpha: 0.4),
                        fontSize: 9, fontWeight: FontWeight.bold,
                      )),
                  ),
                  const SizedBox(width: 8),
                  Expanded(
                    child: Text(
                      scene.direction.isNotEmpty
                          ? (scene.direction.length > 40
                              ? '${scene.direction.substring(0, 40)}…'
                              : scene.direction)
                          : scene.hasImage
                              ? 'Scene ${sceneIndex + 1}'
                              : 'Scene ${sceneIndex + 1} — add image & script',
                      style: TextStyle(color: Colors.white.withValues(alpha: 0.5), fontSize: 12),
                      overflow: TextOverflow.ellipsis,
                    ),
                  ),
                  Icon(
                    isExpanded ? Icons.keyboard_arrow_up_rounded : Icons.keyboard_arrow_down_rounded,
                    size: 16, color: Colors.white.withValues(alpha: 0.3),
                  ),
                  if (canRemove) ...[
                    const SizedBox(width: 4),
                    GestureDetector(
                      onTap: onRemove,
                      child: Icon(Icons.close_rounded, size: 14, color: Colors.white.withValues(alpha: 0.25)),
                    ),
                  ],
                ],
              ),
            ),
          ),

          // ── Scene body ──
          if (isExpanded) ...[
            Divider(height: 1, color: Colors.white.withValues(alpha: 0.06)),
            Padding(
              padding: const EdgeInsets.all(12),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  // Background image
                  _buildImageLabel(),
                  const SizedBox(height: 6),
                  if (!scene.hasImage) ...[
                    // Upload zone
                    GestureDetector(
                      onTap: onPickImage,
                      child: Container(
                        width: double.infinity,
                        height: 80,
                        decoration: BoxDecoration(
                          color: Colors.white.withValues(alpha: 0.02),
                          borderRadius: BorderRadius.circular(10),
                          border: Border.all(
                            color: Colors.white.withValues(alpha: 0.12),
                            style: BorderStyle.solid,
                          ),
                        ),
                        child: Column(
                          mainAxisAlignment: MainAxisAlignment.center,
                          children: [
                            Icon(Icons.upload_file_rounded, size: 18, color: Colors.white.withValues(alpha: 0.3)),
                            const SizedBox(height: 4),
                            Text('Tap to upload background image',
                              style: TextStyle(color: Colors.white.withValues(alpha: 0.4), fontSize: 11)),
                          ],
                        ),
                      ),
                    ),
                    const SizedBox(height: 8),
                    // URL input
                    TextFormField(
                      initialValue: scene.imageUrl ?? '',
                      onChanged: onUpdateImageUrl,
                      style: const TextStyle(color: Colors.white, fontSize: 11),
                      decoration: InputDecoration(
                        hintText: 'Or paste image URL: https://example.com/bg.jpg',
                        hintStyle: TextStyle(color: Colors.white.withValues(alpha: 0.25), fontSize: 11),
                        isDense: true,
                        contentPadding: const EdgeInsets.symmetric(horizontal: 10, vertical: 8),
                        filled: true,
                        fillColor: Colors.white.withValues(alpha: 0.05),
                        border: OutlineInputBorder(
                          borderRadius: BorderRadius.circular(8),
                          borderSide: BorderSide(color: Colors.white.withValues(alpha: 0.1)),
                        ),
                        enabledBorder: OutlineInputBorder(
                          borderRadius: BorderRadius.circular(8),
                          borderSide: BorderSide(color: Colors.white.withValues(alpha: 0.1)),
                        ),
                        focusedBorder: OutlineInputBorder(
                          borderRadius: BorderRadius.circular(8),
                          borderSide: const BorderSide(color: Color(0xFF7C3AED)),
                        ),
                      ),
                    ),
                  ] else ...[
                    // Image preview
                    Stack(
                      children: [
                        ClipRRect(
                          borderRadius: BorderRadius.circular(10),
                          child: scene.imageFile != null
                              ? Image.file(scene.imageFile!,
                                  width: double.infinity, height: 120, fit: BoxFit.cover)
                              : Image.network(scene.imageUrl ?? scene.uploadedUrl,
                                  width: double.infinity, height: 120, fit: BoxFit.cover),
                        ),
                        Positioned(
                          top: 6, right: 6,
                          child: GestureDetector(
                            onTap: onClearImage,
                            child: Container(
                              padding: const EdgeInsets.all(4),
                              decoration: BoxDecoration(
                                color: Colors.black.withValues(alpha: 0.7),
                                shape: BoxShape.circle,
                              ),
                              child: const Icon(Icons.close_rounded, size: 12, color: Colors.white),
                            ),
                          ),
                        ),
                      ],
                    ),
                  ],
                  const SizedBox(height: 12),

                  // Scene direction
                  Row(
                    children: [
                      const Icon(Icons.movie_rounded, size: 10, color: Colors.white38),
                      const SizedBox(width: 4),
                      Text('SCENE DIRECTION',
                        style: TextStyle(color: Colors.white.withValues(alpha: 0.3), fontSize: 9, letterSpacing: 0.8, fontWeight: FontWeight.w600)),
                    ],
                  ),
                  const SizedBox(height: 6),
                  TextFormField(
                    initialValue: scene.direction,
                    onChanged: onUpdateDirection,
                    style: const TextStyle(color: Colors.white, fontSize: 12),
                    decoration: InputDecoration(
                      hintText: 'e.g. A busy Lagos market at sunset. Amara walks through the crowd.',
                      hintStyle: TextStyle(color: Colors.white.withValues(alpha: 0.25), fontSize: 12),
                      isDense: true,
                      contentPadding: const EdgeInsets.symmetric(horizontal: 10, vertical: 8),
                      filled: true,
                      fillColor: Colors.white.withValues(alpha: 0.05),
                      border: OutlineInputBorder(
                        borderRadius: BorderRadius.circular(8),
                        borderSide: BorderSide(color: Colors.white.withValues(alpha: 0.1)),
                      ),
                      enabledBorder: OutlineInputBorder(
                        borderRadius: BorderRadius.circular(8),
                        borderSide: BorderSide(color: Colors.white.withValues(alpha: 0.1)),
                      ),
                      focusedBorder: OutlineInputBorder(
                        borderRadius: BorderRadius.circular(8),
                        borderSide: const BorderSide(color: Color(0xFF7C3AED)),
                      ),
                    ),
                  ),
                  const SizedBox(height: 12),

                  // Dialogue
                  Row(
                    children: [
                      const Icon(Icons.chat_bubble_outline_rounded, size: 10, color: Colors.white38),
                      const SizedBox(width: 4),
                      Text('DIALOGUE & NARRATION',
                        style: TextStyle(color: Colors.white.withValues(alpha: 0.3), fontSize: 9, letterSpacing: 0.8, fontWeight: FontWeight.w600)),
                    ],
                  ),
                  const SizedBox(height: 6),
                  ...scene.dialogue.asMap().entries.map((entry) {
                    final lineIdx = entry.key;
                    final line    = entry.value;
                    return Padding(
                      padding: const EdgeInsets.only(bottom: 6),
                      child: Row(
                        children: [
                          // Character dropdown
                          Container(
                            width: 110,
                            padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
                            decoration: BoxDecoration(
                              color: Colors.white.withValues(alpha: 0.05),
                              borderRadius: BorderRadius.circular(8),
                              border: Border.all(color: Colors.white.withValues(alpha: 0.1)),
                            ),
                            child: DropdownButtonHideUnderline(
                              child: DropdownButton<String>(
                                value: line.characterId,
                                isExpanded: true,
                                dropdownColor: const Color(0xFF1A1A2E),
                                style: const TextStyle(color: Colors.white, fontSize: 11),
                                items: allCharacters.map((c) => DropdownMenuItem(
                                  value: c['id'],
                                  child: Text(c['name']!, overflow: TextOverflow.ellipsis,
                                    style: const TextStyle(fontSize: 11)),
                                )).toList(),
                                onChanged: (v) => onUpdateDialogueLine(lineIdx, v, null),
                              ),
                            ),
                          ),
                          const SizedBox(width: 6),
                          // Line text
                          Expanded(
                            child: TextFormField(
                              initialValue: line.text,
                              onChanged: (v) => onUpdateDialogueLine(lineIdx, null, v),
                              style: const TextStyle(color: Colors.white, fontSize: 11),
                              decoration: InputDecoration(
                                hintText: line.characterId == _kNarratorId
                                    ? 'Narration or stage direction…'
                                    : 'Character says…',
                                hintStyle: TextStyle(color: Colors.white.withValues(alpha: 0.25), fontSize: 11),
                                isDense: true,
                                contentPadding: const EdgeInsets.symmetric(horizontal: 10, vertical: 8),
                                filled: true,
                                fillColor: Colors.white.withValues(alpha: 0.05),
                                border: OutlineInputBorder(
                                  borderRadius: BorderRadius.circular(8),
                                  borderSide: BorderSide(color: Colors.white.withValues(alpha: 0.1)),
                                ),
                                enabledBorder: OutlineInputBorder(
                                  borderRadius: BorderRadius.circular(8),
                                  borderSide: BorderSide(color: Colors.white.withValues(alpha: 0.1)),
                                ),
                                focusedBorder: OutlineInputBorder(
                                  borderRadius: BorderRadius.circular(8),
                                  borderSide: const BorderSide(color: Color(0xFF7C3AED)),
                                ),
                              ),
                            ),
                          ),
                          if (scene.dialogue.length > 1) ...[
                            const SizedBox(width: 4),
                            GestureDetector(
                              onTap: () => onRemoveDialogueLine(lineIdx),
                              child: Icon(Icons.close_rounded, size: 14, color: Colors.white.withValues(alpha: 0.2)),
                            ),
                          ],
                        ],
                      ),
                    );
                  }),
                  GestureDetector(
                    onTap: onAddDialogueLine,
                    child: Row(
                      children: [
                        Icon(Icons.add_rounded, size: 12, color: Colors.white.withValues(alpha: 0.3)),
                        const SizedBox(width: 4),
                        Text('Add line',
                          style: TextStyle(color: Colors.white.withValues(alpha: 0.3), fontSize: 11)),
                      ],
                    ),
                  ),
                ],
              ),
            ),
          ],
        ],
      ),
    );
  }

  Widget _buildImageLabel() {
    return Row(
      children: [
        const Icon(Icons.image_outlined, size: 10, color: Colors.white38),
        const SizedBox(width: 4),
        Text('BACKGROUND IMAGE',
          style: TextStyle(color: Colors.white.withValues(alpha: 0.3), fontSize: 9, letterSpacing: 0.8, fontWeight: FontWeight.w600)),
      ],
    );
  }
}

// ── _SmallButton ───────────────────────────────────────────────────────────────

class _SmallButton extends StatelessWidget {
  final String label;
  final IconData icon;
  final Color color;
  final VoidCallback onTap;

  const _SmallButton({
    required this.label,
    required this.icon,
    required this.color,
    required this.onTap,
  });

  @override
  Widget build(BuildContext context) {
    return GestureDetector(
      onTap: onTap,
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 7),
        decoration: BoxDecoration(
          color: color.withValues(alpha: 0.15),
          borderRadius: BorderRadius.circular(8),
          border: Border.all(color: color.withValues(alpha: 0.3)),
        ),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(icon, size: 12, color: color),
            const SizedBox(width: 5),
            Text(label, style: TextStyle(color: color, fontSize: 11, fontWeight: FontWeight.w500)),
          ],
        ),
      ),
    );
  }
}
