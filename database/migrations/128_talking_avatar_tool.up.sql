-- Migration 128: Talking Avatar tool (photo + script → lip-synced talking-head video)
--
-- HeyGen-Avatar-5-style digital human. Built on FAL.ai now, with a dormant HeyGen
-- provider so admins can switch later with NO deploy (see ai_provider_configs below).
--
-- Providers are registered by INPUT SHAPE, not vendor:
--   fal-avatar-text  → {image_url, text_input, voice}  (model does TTS internally)
--   fal-avatar-audio → {image_url, audio_url}          (audio-driven lip-sync)
--   heygen           → HeyGen Avatar IV bespoke API    (dormant until a key exists)
--
-- Cost note: fal-ai/ai-avatar/single-text ≈ $0.20/sec; veed/fabric-1.0 ≈ $0.08-0.15/sec.
-- point_cost below is a PLACEHOLDER — the owner sets the real PulsePoints value in the
-- admin panel after mapping USD→PulsePoints. max_script_chars bounds worst-case spend.

-- ── Tool row ──────────────────────────────────────────────────────────────────
INSERT INTO studio_tools (
    id, name, slug, description, category, point_cost, provider, provider_tool,
    is_active, is_free, icon, sort_order, entry_point_cost, ui_template, ui_config,
    created_at, updated_at
) VALUES (
    gen_random_uuid(),
    'Talking Avatar',
    'talking-avatar',
    'Turn a single photo into a talking video. Upload a face, type what they should say, pick a voice — Nexus generates a lip-synced talking-head clip.',
    'Create',
    100,                       -- PLACEHOLDER PulsePoints — tune in admin
    'fal.ai',
    'fal-ai/ai-avatar/single-text',
    true,
    false,
    '🗣️',
    70,
    120,                       -- entry_point_cost (min balance to open the tool)
    'talking-avatar',
    '{
        "prompt_placeholder": "Type the script the avatar should say — e.g. Welcome to Loyalty Nexus, recharge and win amazing prizes!",
        "max_script_chars": 300,
        "avatar_voices": ["Bill", "Cherry", "Ethan", "Sarah"],
        "aspect_ratios": [
            { "label": "Portrait", "value": "9:16", "w": 9, "h": 16 },
            { "label": "Square",   "value": "1:1",  "w": 1, "h": 1  },
            { "label": "Landscape","value": "16:9", "w": 16,"h": 9  }
        ],
        "allow_audio_upload": true,
        "generation_warning": "Keep scripts short (under 300 characters) — longer clips cost more and take longer to render.",
        "output_hint": "Photorealistic talking avatar. First render can take 1-3 minutes."
    }'::jsonb,
    NOW(), NOW()
)
ON CONFLICT (slug) DO UPDATE SET
    name             = EXCLUDED.name,
    description      = EXCLUDED.description,
    category         = EXCLUDED.category,
    point_cost       = EXCLUDED.point_cost,
    provider         = EXCLUDED.provider,
    provider_tool    = EXCLUDED.provider_tool,
    is_active        = EXCLUDED.is_active,
    icon             = EXCLUDED.icon,
    sort_order       = EXCLUDED.sort_order,
    entry_point_cost = EXCLUDED.entry_point_cost,
    ui_template      = EXCLUDED.ui_template,
    ui_config        = EXCLUDED.ui_config,
    updated_at       = NOW();

-- ── Provider registry rows (avatar category) ─────────────────────────────────
-- Priority order = fallback order. Admin can reorder / enable / add rows with NO deploy.
INSERT INTO ai_provider_configs
    (name, slug, category, template, env_key, model_id, priority, is_primary, is_active, cost_micros, pulse_pts, notes)
VALUES
    ('FAL Avatar (Single Text)', 'fal-avatar-text',   'avatar', 'fal-avatar-text',  'FAL_API_KEY',    'fal-ai/ai-avatar/single-text', 1, true,  true,  40000, 0, 'Photo + script → talking video; internal TTS. ~$0.20/sec.'),
    ('FAL Avatar (VEED Fabric)', 'fal-avatar-fabric', 'avatar', 'fal-avatar-audio', 'FAL_API_KEY',    'veed/fabric-1.0',              2, false, true,  20000, 0, 'Photo + audio → talking video; cheaper (~$0.08-0.15/sec), TTS-first.'),
    ('HeyGen Avatar IV',         'heygen',            'avatar', 'heygen',           'HEYGEN_API_KEY', '',                             3, false, false, 0,     0, 'DORMANT — max realism. Set HEYGEN_API_KEY + flip is_active in admin to enable.')
ON CONFLICT (slug) DO NOTHING;
