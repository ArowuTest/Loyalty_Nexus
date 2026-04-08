// ══════════════════════════════════════════════════════════════════════════════
// WebsiteBuilderScreen — 5-step wizard matching WebsiteBuilderWizard.tsx
// ══════════════════════════════════════════════════════════════════════════════

import 'dart:async';
import 'dart:convert';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:image_picker/image_picker.dart';
import 'package:url_launcher/url_launcher.dart';
import 'package:share_plus/share_plus.dart';
import 'package:webview_flutter/webview_flutter.dart';
import '../../../core/api/api_client.dart';
import '../../../core/theme/nexus_theme.dart';

// ─── Site Type model ──────────────────────────────────────────────────────────

class _SiteType {
  final String key, label, description, icon;
  final Color color;
  final List<_FieldDef> fields;
  const _SiteType({
    required this.key, required this.label, required this.description,
    required this.icon, required this.color, required this.fields,
  });
}

class _FieldDef {
  final String key, label, hint;
  final bool required;
  const _FieldDef({required this.key, required this.label, required this.hint, this.required = false});
}

const _kSiteTypes = <_SiteType>[
  _SiteType(key: 'shop', label: 'Online Shop', icon: '🛒',
    description: 'Sell products online', color: Color(0xFF34d399),
    fields: [
      _FieldDef(key: 'business_name', label: 'Shop Name', hint: 'e.g. Adeola\'s Fashion Store', required: true),
      _FieldDef(key: 'tagline', label: 'Tagline', hint: 'Short catchy slogan'),
      _FieldDef(key: 'phone', label: 'WhatsApp / Phone', hint: '+234…'),
      _FieldDef(key: 'location', label: 'Location', hint: 'Lagos, Nigeria'),
      _FieldDef(key: 'products', label: 'Products (comma-separated)', hint: 'Dresses, Bags, Shoes'),
    ]),
  _SiteType(key: 'corporate', label: 'Corporate', icon: '🏢',
    description: 'Professional company site', color: Color(0xFF38bdf8),
    fields: [
      _FieldDef(key: 'business_name', label: 'Company Name', hint: 'Acme Ltd.', required: true),
      _FieldDef(key: 'tagline', label: 'Tagline', hint: 'What you do in one line'),
      _FieldDef(key: 'services', label: 'Services', hint: 'Consulting, Training, Research'),
      _FieldDef(key: 'phone', label: 'Phone', hint: '+234…'),
      _FieldDef(key: 'email', label: 'Email', hint: 'info@company.com'),
    ]),
  _SiteType(key: 'professional', label: 'Professional', icon: '👔',
    description: 'Personal professional profile', color: Color(0xFFa78bfa),
    fields: [
      _FieldDef(key: 'business_name', label: 'Your Name', hint: 'Dr. Amaka Obi', required: true),
      _FieldDef(key: 'profession', label: 'Profession', hint: 'Software Engineer / Lawyer'),
      _FieldDef(key: 'bio', label: 'Short Bio', hint: '2-3 sentences about yourself'),
      _FieldDef(key: 'skills', label: 'Skills', hint: 'Flutter, Python, Public Speaking'),
    ]),
  _SiteType(key: 'restaurant', label: 'Restaurant', icon: '🍽️',
    description: 'Food & dining website', color: Color(0xFFf97316),
    fields: [
      _FieldDef(key: 'business_name', label: 'Restaurant Name', hint: 'Buka Palace', required: true),
      _FieldDef(key: 'cuisine', label: 'Cuisine Type', hint: 'Nigerian, Continental, Shawarma'),
      _FieldDef(key: 'address', label: 'Address', hint: '5 Allen Ave, Ikeja'),
      _FieldDef(key: 'phone', label: 'Phone / WhatsApp', hint: '+234…'),
      _FieldDef(key: 'hours', label: 'Opening Hours', hint: 'Mon–Sun 8am–10pm'),
    ]),
  _SiteType(key: 'portfolio', label: 'Portfolio', icon: '🎨',
    description: 'Showcase your work', color: Color(0xFFe879f9),
    fields: [
      _FieldDef(key: 'business_name', label: 'Your Name / Brand', hint: 'Chidi Creative', required: true),
      _FieldDef(key: 'specialty', label: 'Specialty', hint: 'Photography, Design, Art'),
      _FieldDef(key: 'bio', label: 'Bio', hint: 'Tell your story'),
      _FieldDef(key: 'contact', label: 'Contact Email', hint: 'hello@you.com'),
    ]),
  _SiteType(key: 'events', label: 'Events', icon: '🎉',
    description: 'Event or conference page', color: Color(0xFFfbbf24),
    fields: [
      _FieldDef(key: 'business_name', label: 'Event Name', hint: 'TechFest Lagos 2026', required: true),
      _FieldDef(key: 'date', label: 'Event Date', hint: 'April 20, 2026'),
      _FieldDef(key: 'location', label: 'Venue', hint: 'Eko Hotel, Victoria Island'),
      _FieldDef(key: 'description', label: 'Description', hint: 'What is this event about?'),
    ]),
  _SiteType(key: 'church', label: 'Church', icon: '⛪',
    description: 'Church / ministry website', color: Color(0xFF67e8f9),
    fields: [
      _FieldDef(key: 'business_name', label: 'Church Name', hint: 'Living Word Chapel', required: true),
      _FieldDef(key: 'pastor', label: 'Pastor\'s Name', hint: 'Pastor John Eze'),
      _FieldDef(key: 'address', label: 'Address', hint: '12 Grace Road, Abuja'),
      _FieldDef(key: 'service_times', label: 'Service Times', hint: 'Sunday 8am & 10:30am'),
    ]),
  _SiteType(key: 'education', label: 'Education', icon: '🎓',
    description: 'School or tutoring site', color: Color(0xFF6ee7b7),
    fields: [
      _FieldDef(key: 'business_name', label: 'School / Institution Name', hint: 'Nexus Academy', required: true),
      _FieldDef(key: 'tagline', label: 'Tagline', hint: 'Shaping Tomorrow\'s Leaders'),
      _FieldDef(key: 'subjects', label: 'Subjects / Courses', hint: 'Maths, English, Coding'),
      _FieldDef(key: 'phone', label: 'Contact Phone', hint: '+234…'),
    ]),
];

// ─── Photo model ──────────────────────────────────────────────────────────────

class _Photo {
  final String path;
  final String caption;
  _Photo({required this.path, this.caption = ''});
  _Photo copyWith({String? caption}) => _Photo(path: path, caption: caption ?? this.caption);
}

// ══════════════════════════════════════════════════════════════════════════════
// WebsiteBuilderScreen
// ══════════════════════════════════════════════════════════════════════════════

class WebsiteBuilderScreen extends ConsumerStatefulWidget {
  const WebsiteBuilderScreen({super.key});

  @override
  ConsumerState<WebsiteBuilderScreen> createState() => _WebsiteBuilderScreenState();
}

class _WebsiteBuilderScreenState extends ConsumerState<WebsiteBuilderScreen> {
  final _pageCtrl = PageController();
  int _step = 0;

  // Step 1
  _SiteType? _siteType;

  // Step 2
  final _fieldCtrls = <String, TextEditingController>{};
  final _slugCtrl = TextEditingController();
  bool _slugAvail = false;
  bool _slugLoading = false;
  Timer? _slugTimer;
  String _slugStatus = ''; // '', 'free', 'taken'

  // Step 3
  final _photos = <_Photo>[];

  // Step 4
  bool _generating = false;
  String? _genError;

  // Step 5
  String? _generationId;
  String? _publicUrl;
  String? _vanitySlug;
  String _genStatus = 'pending';
  Timer? _pollTimer;
  WebViewController? _webCtrl;

  @override
  void dispose() {
    _pageCtrl.dispose();
    _slugCtrl.dispose();
    _slugTimer?.cancel();
    _pollTimer?.cancel();
    for (final c in _fieldCtrls.values) c.dispose();
    super.dispose();
  }

  void _initFields() {
    if (_siteType == null) return;
    for (final c in _fieldCtrls.values) c.dispose();
    _fieldCtrls.clear();
    for (final f in _siteType!.fields) {
      final ctrl = TextEditingController();
      _fieldCtrls[f.key] = ctrl;
      if (f.key == 'business_name') {
        ctrl.addListener(_autoSlug);
      }
    }
  }

  void _autoSlug() {
    final name = _fieldCtrls['business_name']?.text ?? '';
    final slug = name.toLowerCase()
        .replaceAll(RegExp(r'[^a-z0-9\s]'), '')
        .trim()
        .replaceAll(RegExp(r'\s+'), '-');
    if (slug != _slugCtrl.text) {
      _slugCtrl.text = slug;
      _checkSlug(slug);
    }
  }

  void _onSlugChanged(String val) {
    _slugTimer?.cancel();
    _slugTimer = Timer(const Duration(milliseconds: 500), () => _checkSlug(val));
  }

  Future<void> _checkSlug(String slug) async {
    if (slug.isEmpty) { setState(() => _slugStatus = ''); return; }
    setState(() { _slugLoading = true; _slugStatus = ''; });
    try {
      final api = ref.read(studioApiProvider);
      final res = await api.checkSlug(slug);
      final avail = res['available'] as bool? ?? true;
      setState(() {
        _slugAvail = avail;
        _slugStatus = avail ? 'free' : 'taken';
        _slugLoading = false;
      });
    } catch (_) {
      setState(() { _slugLoading = false; _slugStatus = ''; });
    }
  }

  void _goNext() {
    if (_step < 4) {
      setState(() => _step++);
      _pageCtrl.animateToPage(_step,
        duration: const Duration(milliseconds: 300), curve: Curves.easeInOut);
    }
  }

  void _goBack() {
    if (_step > 0) {
      setState(() => _step--);
      _pageCtrl.animateToPage(_step,
        duration: const Duration(milliseconds: 300), curve: Curves.easeInOut);
    } else {
      Navigator.of(context).pop();
    }
  }

  Future<void> _generate() async {
    setState(() { _generating = true; _genError = null; });
    try {
      final fields = <String, String>{};
      _fieldCtrls.forEach((k, v) { if (v.text.isNotEmpty) fields[k] = v.text; });

      final photos = <Map<String, String>>[];
      for (final p in _photos) {
        photos.add({'path': p.path, 'caption': p.caption});
      }

      final api = ref.read(studioApiProvider);
      final res = await api.createWebsite(
        siteType: _siteType!.key,
        vanitySlug: _slugCtrl.text.isNotEmpty ? _slugCtrl.text : null,
        fields: fields,
        photos: photos,
      );
      _generationId = res['generation_id']?.toString();
      _publicUrl = res['public_url']?.toString();
      _vanitySlug = res['vanity_slug']?.toString();
      _genStatus = res['status']?.toString() ?? 'pending';

      setState(() { _generating = false; });
      _goNext();
      _startPolling();
    } catch (e) {
      setState(() {
        _generating = false;
        _genError = e.toString();
      });
    }
  }

  void _startPolling() {
    _pollTimer?.cancel();
    _pollTimer = Timer.periodic(const Duration(seconds: 3), (_) async {
      if (_generationId == null) return;
      try {
        final api = ref.read(studioApiProvider);
        final res = await api.getGeneration(_generationId!);
        final status = res['status']?.toString() ?? 'pending';
        setState(() => _genStatus = status);
        if (status == 'completed') {
          _pollTimer?.cancel();
          if (_publicUrl != null) {
            final ctrl = WebViewController()
              ..setJavaScriptMode(JavaScriptMode.unrestricted)
              ..loadRequest(Uri.parse(_publicUrl!));
            setState(() => _webCtrl = ctrl);
          }
        } else if (status == 'failed') {
          _pollTimer?.cancel();
        }
      } catch (_) {}
    });
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: NexusColors.background,
      appBar: AppBar(
        backgroundColor: NexusColors.background,
        leading: IconButton(
          icon: const Icon(Icons.arrow_back_rounded),
          onPressed: _goBack,
        ),
        title: Text('Website Builder — Step ${_step + 1} of 5',
          style: const TextStyle(fontSize: 15, fontWeight: FontWeight.w700)),
        bottom: PreferredSize(
          preferredSize: const Size.fromHeight(4),
          child: LinearProgressIndicator(
            value: (_step + 1) / 5,
            backgroundColor: NexusColors.surface,
            color: NexusColors.gold,
            minHeight: 3,
          ),
        ),
      ),
      body: PageView(
        controller: _pageCtrl,
        physics: const NeverScrollableScrollPhysics(),
        children: [
          _buildStep1(),
          _buildStep2(),
          _buildStep3(),
          _buildStep4(),
          _buildStep5(),
        ],
      ),
    );
  }

  // ── Step 1: Site type selection ───────────────────────────────────────────

  Widget _buildStep1() {
    return SingleChildScrollView(
      padding: const EdgeInsets.all(20),
      child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
        const Text('What type of website?',
          style: TextStyle(fontSize: 22, fontWeight: FontWeight.w800,
            color: NexusColors.textPrimary)),
        const SizedBox(height: 6),
        const Text('Choose the template that fits your needs',
          style: TextStyle(color: NexusColors.textSecondary, fontSize: 14)),
        const SizedBox(height: 24),
        GridView.count(
          shrinkWrap: true,
          physics: const NeverScrollableScrollPhysics(),
          crossAxisCount: 2,
          mainAxisSpacing: 12,
          crossAxisSpacing: 12,
          childAspectRatio: 1.3,
          children: _kSiteTypes.map((t) => _SiteTypeCard(
            type: t,
            selected: _siteType?.key == t.key,
            onTap: () => setState(() { _siteType = t; _initFields(); }),
          )).toList(),
        ),
        const SizedBox(height: 28),
        ElevatedButton(
          onPressed: _siteType != null ? _goNext : null,
          style: ElevatedButton.styleFrom(
            backgroundColor: NexusColors.gold,
            foregroundColor: Colors.black,
            minimumSize: const Size(double.infinity, 52),
          ),
          child: const Text('Continue', style: TextStyle(fontWeight: FontWeight.w700, fontSize: 15)),
        ),
      ]),
    );
  }

  // ── Step 2: Details + slug ────────────────────────────────────────────────

  Widget _buildStep2() {
    final type = _siteType;
    if (type == null) return const SizedBox.shrink();
    return SingleChildScrollView(
      padding: const EdgeInsets.all(20),
      child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
        Text('${type.icon} ${type.label} Details',
          style: const TextStyle(fontSize: 20, fontWeight: FontWeight.w800,
            color: NexusColors.textPrimary)),
        const SizedBox(height: 20),
        ...type.fields.map((f) {
          final ctrl = _fieldCtrls[f.key] ?? TextEditingController();
          return Padding(
            padding: const EdgeInsets.only(bottom: 14),
            child: TextField(
              controller: ctrl,
              decoration: InputDecoration(
                labelText: f.label + (f.required ? ' *' : ''),
                hintText: f.hint,
              ),
              style: const TextStyle(color: NexusColors.textPrimary),
            ),
          );
        }),
        const SizedBox(height: 8),
        const Divider(color: NexusColors.border),
        const SizedBox(height: 12),
        const Text('Your Website URL',
          style: TextStyle(fontSize: 14, fontWeight: FontWeight.w700,
            color: NexusColors.textSecondary)),
        const SizedBox(height: 8),
        Row(children: [
          const Text('nexus.app/s/',
            style: TextStyle(color: NexusColors.textMuted, fontSize: 14)),
          Expanded(
            child: TextField(
              controller: _slugCtrl,
              onChanged: _onSlugChanged,
              style: const TextStyle(color: NexusColors.textPrimary),
              decoration: InputDecoration(
                hintText: 'your-slug',
                border: const UnderlineInputBorder(),
                enabledBorder: const UnderlineInputBorder(
                  borderSide: BorderSide(color: NexusColors.border)),
                focusedBorder: UnderlineInputBorder(
                  borderSide: BorderSide(color: NexusColors.primary)),
                suffixIcon: _slugLoading
                  ? const SizedBox(width: 16, height: 16,
                      child: Padding(padding: EdgeInsets.all(12),
                        child: CircularProgressIndicator(strokeWidth: 2)))
                  : _slugStatus == 'free'
                    ? const Icon(Icons.check_circle, color: NexusColors.green, size: 18)
                    : _slugStatus == 'taken'
                      ? const Icon(Icons.cancel, color: NexusColors.red, size: 18)
                      : null,
              ),
            ),
          ),
        ]),
        if (_slugStatus == 'free')
          const Padding(
            padding: EdgeInsets.only(top: 4),
            child: Text('✓ This slug is available!',
              style: TextStyle(color: NexusColors.green, fontSize: 12))),
        if (_slugStatus == 'taken')
          const Padding(
            padding: EdgeInsets.only(top: 4),
            child: Text('✗ Already taken — try a different slug',
              style: TextStyle(color: NexusColors.red, fontSize: 12))),
        const SizedBox(height: 28),
        ElevatedButton(
          onPressed: _goNext,
          style: ElevatedButton.styleFrom(
            backgroundColor: NexusColors.gold, foregroundColor: Colors.black,
            minimumSize: const Size(double.infinity, 52)),
          child: const Text('Continue', style: TextStyle(fontWeight: FontWeight.w700, fontSize: 15)),
        ),
      ]),
    );
  }

  // ── Step 3: Photo upload ──────────────────────────────────────────────────

  Widget _buildStep3() {
    return SingleChildScrollView(
      padding: const EdgeInsets.all(20),
      child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
        const Text('Add Photos',
          style: TextStyle(fontSize: 20, fontWeight: FontWeight.w800,
            color: NexusColors.textPrimary)),
        const SizedBox(height: 6),
        const Text('Upload up to 6 photos for your website (optional)',
          style: TextStyle(color: NexusColors.textSecondary, fontSize: 14)),
        const SizedBox(height: 20),
        if (_photos.isNotEmpty)
          ...List.generate(_photos.length, (i) => _PhotoRow(
            photo: _photos[i],
            onRemove: () => setState(() => _photos.removeAt(i)),
            onCaption: (c) => setState(() => _photos[i] = _photos[i].copyWith(caption: c)),
          )),
        if (_photos.length < 6)
          GestureDetector(
            onTap: _addPhoto,
            child: Container(
              margin: const EdgeInsets.only(top: 8),
              height: 80,
              decoration: BoxDecoration(
                color: NexusColors.surface,
                borderRadius: BorderRadius.circular(12),
                border: Border.all(color: NexusColors.border, style: BorderStyle.solid),
              ),
              child: const Row(mainAxisAlignment: MainAxisAlignment.center, children: [
                Icon(Icons.add_photo_alternate_outlined, color: NexusColors.textSecondary),
                SizedBox(width: 8),
                Text('Add Photo', style: TextStyle(color: NexusColors.textSecondary)),
              ]),
            ),
          ),
        const SizedBox(height: 32),
        ElevatedButton(
          onPressed: _goNext,
          style: ElevatedButton.styleFrom(
            backgroundColor: NexusColors.gold, foregroundColor: Colors.black,
            minimumSize: const Size(double.infinity, 52)),
          child: Text(_photos.isEmpty ? 'Skip & Continue' : 'Continue',
            style: const TextStyle(fontWeight: FontWeight.w700, fontSize: 15)),
        ),
      ]),
    );
  }

  Future<void> _addPhoto() async {
    final picker = ImagePicker();
    final img = await picker.pickImage(source: ImageSource.gallery, imageQuality: 75);
    if (img != null) setState(() => _photos.add(_Photo(path: img.path)));
  }

  // ── Step 4: Review ────────────────────────────────────────────────────────

  Widget _buildStep4() {
    final fields = <String, String>{};
    _fieldCtrls.forEach((k, v) { if (v.text.isNotEmpty) fields[k] = v.text; });

    return SingleChildScrollView(
      padding: const EdgeInsets.all(20),
      child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
        const Text('Review & Generate',
          style: TextStyle(fontSize: 20, fontWeight: FontWeight.w800,
            color: NexusColors.textPrimary)),
        const SizedBox(height: 20),
        _ReviewRow('Site Type', '${_siteType?.icon ?? ''} ${_siteType?.label ?? ''}'),
        if (_slugCtrl.text.isNotEmpty) _ReviewRow('URL', 'nexus.app/s/${_slugCtrl.text}'),
        _ReviewRow('Fields filled', '${fields.length}'),
        _ReviewRow('Photos', '${_photos.length}'),
        _ReviewRow('Cost', '25 Pulse Points'),
        if (_genError != null) ...[
          const SizedBox(height: 12),
          Container(
            padding: const EdgeInsets.all(12),
            decoration: BoxDecoration(
              color: NexusColors.redDim,
              borderRadius: BorderRadius.circular(10),
              border: Border.all(color: NexusColors.red.withOpacity(0.3)),
            ),
            child: Text(_genError!,
              style: const TextStyle(color: NexusColors.red, fontSize: 13)),
          ),
        ],
        const SizedBox(height: 32),
        ElevatedButton(
          onPressed: _generating ? null : _generate,
          style: ElevatedButton.styleFrom(
            backgroundColor: NexusColors.gold, foregroundColor: Colors.black,
            minimumSize: const Size(double.infinity, 52)),
          child: _generating
            ? const SizedBox(width: 20, height: 20,
                child: CircularProgressIndicator(color: Colors.black, strokeWidth: 2))
            : const Text('⚡ Generate Website',
                style: TextStyle(fontWeight: FontWeight.w700, fontSize: 15)),
        ),
      ]),
    );
  }

  // ── Step 5: Result ────────────────────────────────────────────────────────

  Widget _buildStep5() {
    final completed = _genStatus == 'completed';
    final failed = _genStatus == 'failed';
    final url = _publicUrl ?? '';

    return Column(children: [
      if (!completed && !failed) ...[
        const Expanded(child: Center(child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            CircularProgressIndicator(color: NexusColors.gold),
            SizedBox(height: 20),
            Text('Building your website…',
              style: TextStyle(color: NexusColors.textPrimary, fontSize: 16,
                fontWeight: FontWeight.w600)),
            SizedBox(height: 8),
            Text('This usually takes 15–30 seconds',
              style: TextStyle(color: NexusColors.textSecondary, fontSize: 13)),
          ],
        ))),
      ] else if (failed) ...[
        const Expanded(child: Center(child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(Icons.error_outline, color: NexusColors.red, size: 48),
            SizedBox(height: 16),
            Text('Generation failed',
              style: TextStyle(color: NexusColors.red, fontSize: 16, fontWeight: FontWeight.w700)),
            SizedBox(height: 8),
            Text('Please try again',
              style: TextStyle(color: NexusColors.textSecondary)),
          ],
        ))),
      ] else ...[
        Padding(
          padding: const EdgeInsets.all(16),
          child: Row(children: [
            Expanded(
              child: Text('🎉 Your website is live!',
                style: const TextStyle(fontSize: 16, fontWeight: FontWeight.w700,
                  color: NexusColors.textPrimary)),
            ),
            IconButton(
              icon: const Icon(Icons.share_rounded, color: NexusColors.gold),
              onPressed: () => Share.share(url),
            ),
            IconButton(
              icon: const Icon(Icons.copy_rounded, color: NexusColors.textSecondary),
              onPressed: () {
                Clipboard.setData(ClipboardData(text: url));
                ScaffoldMessenger.of(context).showSnackBar(
                  const SnackBar(content: Text('Link copied!')));
              },
            ),
            IconButton(
              icon: const Icon(Icons.open_in_browser_rounded, color: NexusColors.textSecondary),
              onPressed: () => launchUrl(Uri.parse(url)),
            ),
          ]),
        ),
        Container(
          margin: const EdgeInsets.symmetric(horizontal: 16),
          padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
          decoration: BoxDecoration(
            color: NexusColors.surface,
            borderRadius: BorderRadius.circular(8),
            border: Border.all(color: NexusColors.border),
          ),
          child: Text(url,
            style: const TextStyle(color: NexusColors.primary, fontSize: 12),
            overflow: TextOverflow.ellipsis),
        ),
        const SizedBox(height: 8),
        if (_webCtrl != null)
          Expanded(child: ClipRRect(
            borderRadius: const BorderRadius.all(Radius.circular(12)),
            child: Padding(
              padding: const EdgeInsets.all(12),
              child: WebViewWidget(controller: _webCtrl!),
            ),
          )),
      ],
    ]);
  }
}

// ─── Supporting widgets ───────────────────────────────────────────────────────

class _SiteTypeCard extends StatelessWidget {
  final _SiteType type;
  final bool selected;
  final VoidCallback onTap;
  const _SiteTypeCard({required this.type, required this.selected, required this.onTap});

  @override
  Widget build(BuildContext context) {
    return GestureDetector(
      onTap: onTap,
      child: AnimatedContainer(
        duration: const Duration(milliseconds: 150),
        padding: const EdgeInsets.all(12),
        decoration: BoxDecoration(
          color: selected ? type.color.withOpacity(0.12) : NexusColors.surface,
          borderRadius: BorderRadius.circular(14),
          border: Border.all(
            color: selected ? type.color : NexusColors.border,
            width: selected ? 1.5 : 1,
          ),
        ),
        child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
          Text(type.icon, style: const TextStyle(fontSize: 24)),
          const SizedBox(height: 6),
          Text(type.label,
            style: TextStyle(
              fontSize: 13, fontWeight: FontWeight.w700,
              color: selected ? type.color : NexusColors.textPrimary)),
          const SizedBox(height: 2),
          Text(type.description,
            maxLines: 2, overflow: TextOverflow.ellipsis,
            style: const TextStyle(fontSize: 11, color: NexusColors.textSecondary)),
        ]),
      ),
    );
  }
}

class _ReviewRow extends StatelessWidget {
  final String label, value;
  const _ReviewRow(this.label, this.value);

  @override
  Widget build(BuildContext context) => Padding(
    padding: const EdgeInsets.symmetric(vertical: 8),
    child: Row(children: [
      SizedBox(width: 120,
        child: Text(label, style: const TextStyle(
          color: NexusColors.textSecondary, fontSize: 13))),
      Expanded(child: Text(value, style: const TextStyle(
        color: NexusColors.textPrimary, fontSize: 13, fontWeight: FontWeight.w600))),
    ]),
  );
}

class _PhotoRow extends StatelessWidget {
  final _Photo photo;
  final VoidCallback onRemove;
  final ValueChanged<String> onCaption;
  const _PhotoRow({required this.photo, required this.onRemove, required this.onCaption});

  @override
  Widget build(BuildContext context) {
    return Container(
      margin: const EdgeInsets.only(bottom: 10),
      padding: const EdgeInsets.all(10),
      decoration: BoxDecoration(
        color: NexusColors.surface,
        borderRadius: BorderRadius.circular(10),
        border: Border.all(color: NexusColors.border),
      ),
      child: Row(children: [
        ClipRRect(
          borderRadius: BorderRadius.circular(6),
          child: Image.asset(photo.path, width: 56, height: 56, fit: BoxFit.cover,
            errorBuilder: (_, __, ___) => Container(
              width: 56, height: 56,
              color: NexusColors.surfaceHigh,
              child: const Icon(Icons.image, color: NexusColors.textMuted)),
          ),
        ),
        const SizedBox(width: 10),
        Expanded(
          child: TextField(
            onChanged: onCaption,
            style: const TextStyle(color: NexusColors.textPrimary, fontSize: 13),
            decoration: const InputDecoration(
              hintText: 'Add caption…',
              hintStyle: TextStyle(fontSize: 12),
              border: UnderlineInputBorder(),
              enabledBorder: UnderlineInputBorder(
                borderSide: BorderSide(color: NexusColors.border)),
              contentPadding: EdgeInsets.symmetric(vertical: 4),
            ),
          ),
        ),
        IconButton(
          icon: const Icon(Icons.close_rounded, size: 18, color: NexusColors.textSecondary),
          onPressed: onRemove,
        ),
      ]),
    );
  }
}
