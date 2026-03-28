import 'package:flutter/material.dart';
import 'template_types.dart';

// ─── Translation language list (mirrors webapp) ───────────────────────────────
const _translateLanguages = [
  {'code': 'en',  'label': 'English'},
  {'code': 'fr',  'label': 'French'},
  {'code': 'es',  'label': 'Spanish'},
  {'code': 'pt',  'label': 'Portuguese'},
  {'code': 'de',  'label': 'German'},
  {'code': 'ar',  'label': 'Arabic'},
  {'code': 'zh',  'label': 'Chinese'},
  {'code': 'sw',  'label': 'Swahili'},
  {'code': 'yo',  'label': 'Yoruba'},
  {'code': 'ha',  'label': 'Hausa'},
  {'code': 'ig',  'label': 'Igbo'},
  {'code': 'pcm', 'label': 'Nigerian Pidgin'},
  {'code': 'af',  'label': 'Afrikaans'},
];

const _quickTargets = ['English', 'French', 'Yoruba', 'Hausa', 'Igbo', 'Spanish', 'Arabic'];

// ─── Translate sub-layout ─────────────────────────────────────────────────────
class _TranslateLayout extends StatefulWidget {
  final TemplateProps props;
  const _TranslateLayout({required this.props});

  @override
  State<_TranslateLayout> createState() => _TranslateLayoutState();
}

class _TranslateLayoutState extends State<_TranslateLayout> {
  final _textCtrl    = TextEditingController();
  String _sourceLang = 'auto';
  String _targetLang = 'en';

  TemplateProps get p => widget.props;

  List<Map<String, String>> get _languages {
    final raw = p.uiConfig['translate_languages'];
    if (raw is List) return raw.cast<Map<String, String>>();
    return List<Map<String, String>>.from(_translateLanguages);
  }

  bool get _isValid => _textCtrl.text.trim().length >= 3 && _targetLang.isNotEmpty;

  String _labelFor(String code) =>
      _languages.firstWhere((l) => l['code'] == code,
          orElse: () => {'code': code, 'label': code})['label']!;

  void _handleSubmit() {
    if (!_isValid || p.isLoading || !p.canAfford) return;
    final srcLabel = _sourceLang == 'auto' ? 'Auto-detect' : _labelFor(_sourceLang);
    final tgtLabel = _labelFor(_targetLang);
    p.onSubmit(GeneratePayload(
      prompt:   'Translate the following text from $srcLabel to $tgtLabel:\n\n${_textCtrl.text.trim()}',
      language: _targetLang,
      extraParams: {
        'source_language': _sourceLang,
        'target_language': _targetLang,
        'original_text':   _textCtrl.text.trim(),
      },
    ));
  }

  @override
  void dispose() {
    _textCtrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [

        // ── Language pair ──
        Row(
          children: [
            Expanded(child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                buildSectionLabel('From'),
                _langDropdown(
                  value: _sourceLang,
                  items: [
                    const DropdownMenuItem(value: 'auto', child: Text('Auto-detect')),
                    ..._languages.map((l) => DropdownMenuItem(value: l['code'], child: Text(l['label']!))),
                  ],
                  onChanged: (v) => setState(() => _sourceLang = v!),
                ),
              ],
            )),
            Padding(
              padding: const EdgeInsets.only(top: 20, left: 10, right: 10),
              child: Icon(Icons.arrow_forward, size: 16, color: Colors.white.withOpacity(0.25)),
            ),
            Expanded(child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                buildSectionLabel('To'),
                _langDropdown(
                  value: _targetLang,
                  items: _languages.map((l) =>
                      DropdownMenuItem(value: l['code'], child: Text(l['label']!))).toList(),
                  onChanged: (v) => setState(() => _targetLang = v!),
                ),
              ],
            )),
          ],
        ),
        const SizedBox(height: kTemplateSpacing),

        // ── Quick target chips ──
        buildSectionLabel('Quick select target'),
        Wrap(
          spacing: 6, runSpacing: 6,
          children: _quickTargets.map((label) {
            final lang = _languages.where((l) => l['label'] == label).firstOrNull;
            if (lang == null) return const SizedBox.shrink();
            return buildChip(
              label: label,
              selected: _targetLang == lang['code'],
              onTap: () => setState(() => _targetLang = lang['code']!),
            );
          }).toList(),
        ),
        const SizedBox(height: kTemplateSpacing),

        // ── Text to translate ──
        Row(
          mainAxisAlignment: MainAxisAlignment.spaceBetween,
          children: [
            Text('TEXT TO TRANSLATE', style: kLabelStyle),
            ValueListenableBuilder(
              valueListenable: _textCtrl,
              builder: (_, __, ___) =>
                  Text('${_textCtrl.text.length}/5000', style: kHintStyle),
            ),
          ],
        ),
        const SizedBox(height: 6),
        buildTextArea(
          controller: _textCtrl,
          placeholder: p.uiConfig['prompt_placeholder']?.toString() ??
              'Paste or type the text you want to translate…',
          maxLines: 5,
          autoFocus: true,
          maxLength: 5000,
        ),
        const SizedBox(height: kTemplateSpacing),

        ValueListenableBuilder(
          valueListenable: _textCtrl,
          builder: (_, __, ___) => buildGenerateButton(
            label: 'Translate',
            enabled: _isValid && p.canAfford,
            isLoading: p.isLoading,
            onTap: _handleSubmit,
            icon: Icons.translate_rounded,
          ),
        ),
      ],
    );
  }

  Widget _langDropdown({
    required String value,
    required List<DropdownMenuItem<String>> items,
    required ValueChanged<String?> onChanged,
  }) {
    return DropdownButtonFormField<String>(
      value: value,
      onChanged: onChanged,
      items: items,
      style: const TextStyle(color: Colors.white, fontSize: 13),
      dropdownColor: const Color(0xFF1A1A2E),
      decoration: InputDecoration(
        filled: true,
        fillColor: Colors.white.withOpacity(0.04),
        border: OutlineInputBorder(
          borderRadius: BorderRadius.circular(10),
          borderSide: BorderSide(color: Colors.white.withOpacity(0.1)),
        ),
        enabledBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(10),
          borderSide: BorderSide(color: Colors.white.withOpacity(0.1)),
        ),
        focusedBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(10),
          borderSide: const BorderSide(color: Color(0xFF7C3AED), width: 1.5),
        ),
        contentPadding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
      ),
    );
  }
}

// ─── Main KnowledgeDoc template ───────────────────────────────────────────────
class KnowledgeDocTemplate extends StatefulWidget {
  final TemplateProps props;
  const KnowledgeDocTemplate({super.key, required this.props});

  @override
  State<KnowledgeDocTemplate> createState() => _KnowledgeDocTemplateState();
}

class _KnowledgeDocTemplateState extends State<KnowledgeDocTemplate> {
  late final List<Map<String, dynamic>> _fields;
  late final Map<String, TextEditingController> _controllers;

  TemplateProps get p => widget.props;

  @override
  void initState() {
    super.initState();
    final rawFields = p.uiConfig['fields'];
    _fields = rawFields is List
        ? rawFields.cast<Map<String, dynamic>>()
        : [
            {
              'key': 'prompt',
              'label': 'Describe what you want',
              'type': 'textarea',
              'required': true,
              'placeholder': 'Provide details about what you\'d like to generate…',
              'rows': 5,
              'default': '',
            }
          ];

    _controllers = {
      for (final f in _fields)
        f['key'] as String: TextEditingController(text: f['default']?.toString() ?? ''),
    };
  }

  bool _isValid() {
    return _fields.every((f) {
      if (f['required'] != true) return true;
      return (_controllers[f['key']]?.text.trim().length ?? 0) >= 3;
    });
  }

  void _handleSubmit() {
    if (!_isValid() || p.isLoading || !p.canAfford) return;
    final parts = _fields
        .where((f) => _controllers[f['key']]?.text.trim().isNotEmpty == true)
        .map((f) => '${f['label']}: ${_controllers[f['key']]!.text.trim()}')
        .toList();
    p.onSubmit(GeneratePayload(
      prompt: parts.join('\n'),
      extraParams: {
        for (final f in _fields)
          f['key'] as String: _controllers[f['key']]!.text,
        if (p.uiConfig['output_format'] != null)
          'output_format': p.uiConfig['output_format'],
      },
    ));
  }

  @override
  void dispose() {
    for (final c in _controllers.values) c.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    // Translate tool gets its own dedicated layout
    if (p.slug == 'translate') {
      return _TranslateLayout(props: p);
    }

    final outputFormat = p.uiConfig['output_format']?.toString();
    final btnLabel = outputFormat == 'document'
        ? 'Generate Document'
        : outputFormat == 'audio'
            ? 'Generate Audio'
            : 'Generate';

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [

        // ── Dynamic fields ──
        for (int i = 0; i < _fields.length; i++) ...[
          _buildField(_fields[i], i == 0),
          const SizedBox(height: kTemplateSpacing),
        ],

        // ── Output format badge ──
        if (outputFormat != null) ...[
          Row(children: [
            Text('Output: ', style: kHintStyle),
            Container(
              padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
              decoration: BoxDecoration(
                color: outputFormat == 'document'
                    ? const Color(0xFF2563EB).withOpacity(0.2)
                    : outputFormat == 'audio'
                        ? const Color(0xFF059669).withOpacity(0.2)
                        : Colors.white.withOpacity(0.08),
                borderRadius: BorderRadius.circular(999),
                border: Border.all(
                  color: outputFormat == 'document'
                      ? const Color(0xFF2563EB).withOpacity(0.3)
                      : outputFormat == 'audio'
                          ? const Color(0xFF059669).withOpacity(0.3)
                          : Colors.white.withOpacity(0.15),
                ),
              ),
              child: Text(
                outputFormat == 'document' ? '📄 Document'
                    : outputFormat == 'audio' ? '🎙 Audio'
                    : '📝 Text',
                style: TextStyle(
                  fontSize: 11,
                  fontWeight: FontWeight.w700,
                  color: outputFormat == 'document'
                      ? const Color(0xFF93C5FD)
                      : outputFormat == 'audio'
                          ? const Color(0xFF6EE7B7)
                          : Colors.white.withOpacity(0.5),
                ),
              ),
            ),
          ]),
          const SizedBox(height: 10),
        ],

        // ── Output hint ──
        if (p.uiConfig['output_hint'] != null) ...[
          Text(p.uiConfig['output_hint'].toString(), style: kHintStyle),
          const SizedBox(height: 12),
        ],

        // ── Generate button — listens to ALL controllers ──
        ListenableBuilder(
          listenable: Listenable.merge(_controllers.values.toList()),
          builder: (_, __) => buildGenerateButton(
            label: btnLabel,
            enabled: _isValid() && p.canAfford,
            isLoading: p.isLoading,
            onTap: _handleSubmit,
          ),
        ),
      ],
    );
  }

  Widget _buildField(Map<String, dynamic> field, bool autoFocus) {
    final key         = field['key'] as String;
    final label       = field['label'] as String;
    final type        = field['type'] as String? ?? 'textarea';
    final required    = field['required'] == true;
    final placeholder = field['placeholder'] as String? ?? '';
    final rows        = (field['rows'] as int?) ?? 4;
    final options     = (field['options'] as List?)?.cast<String>() ?? [];
    final ctrl        = _controllers[key]!;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(children: [
          Text(label.toUpperCase(), style: kLabelStyle),
          if (required)
            const Text(' *', style: TextStyle(color: Colors.redAccent, fontSize: 12)),
        ]),
        const SizedBox(height: 6),
        if (type == 'textarea') ...[
          buildTextArea(
            controller: ctrl,
            placeholder: placeholder,
            maxLines: rows,
            autoFocus: autoFocus,
          ),
          const SizedBox(height: 4),
          ValueListenableBuilder(
            valueListenable: ctrl,
            builder: (_, __, ___) =>
                Text('${ctrl.text.length}/1000 characters', style: kHintStyle),
          ),
        ] else if (type == 'select') ...[
          DropdownButtonFormField<String>(
            value: ctrl.text.isNotEmpty ? ctrl.text : null,
            onChanged: (v) => ctrl.text = v ?? '',
            items: options.map((o) => DropdownMenuItem(value: o, child: Text(o))).toList(),
            style: const TextStyle(color: Colors.white, fontSize: 13),
            dropdownColor: const Color(0xFF1A1A2E),
            hint: Text(required ? 'Choose an option…' : 'Optional…',
                style: TextStyle(color: Colors.white.withOpacity(0.3), fontSize: 12)),
            decoration: InputDecoration(
              filled: true,
              fillColor: Colors.white.withOpacity(0.04),
              border: OutlineInputBorder(
                borderRadius: BorderRadius.circular(12),
                borderSide: BorderSide(color: Colors.white.withOpacity(0.1)),
              ),
              enabledBorder: OutlineInputBorder(
                borderRadius: BorderRadius.circular(12),
                borderSide: BorderSide(color: Colors.white.withOpacity(0.1)),
              ),
              focusedBorder: OutlineInputBorder(
                borderRadius: BorderRadius.circular(12),
                borderSide: const BorderSide(color: Color(0xFF7C3AED), width: 1.5),
              ),
              contentPadding: const EdgeInsets.symmetric(horizontal: 14, vertical: 12),
            ),
          ),
        ] else ...[
          TextField(
            controller: ctrl,
            autofocus: autoFocus,
            style: const TextStyle(color: Colors.white, fontSize: 14),
            decoration: InputDecoration(
              hintText: placeholder,
              hintStyle: TextStyle(color: Colors.white.withOpacity(0.3), fontSize: 13),
              filled: true,
              fillColor: Colors.white.withOpacity(0.04),
              border: OutlineInputBorder(
                borderRadius: BorderRadius.circular(12),
                borderSide: BorderSide(color: Colors.white.withOpacity(0.1)),
              ),
              enabledBorder: OutlineInputBorder(
                borderRadius: BorderRadius.circular(12),
                borderSide: BorderSide(color: Colors.white.withOpacity(0.1)),
              ),
              focusedBorder: OutlineInputBorder(
                borderRadius: BorderRadius.circular(12),
                borderSide: const BorderSide(color: Color(0xFF7C3AED), width: 1.5),
              ),
              contentPadding: const EdgeInsets.symmetric(horizontal: 14, vertical: 12),
            ),
          ),
        ],
      ],
    );
  }
}
