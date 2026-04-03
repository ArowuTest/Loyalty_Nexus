-- Migration 087: Add Video Editor tool (Grok Imagine natural language video editing)
-- Uses xAI Grok Imagine video editing API: upload a video, describe the edit in plain English
-- Grok rewrites the video preserving original duration and aspect ratio
-- Tier: Gold and above (Grok API cost: $0.05/sec × ~6s = ~$0.30 per generation)

INSERT INTO studio_tools (
    slug,
    name,
    description,
    category,
    icon,
    ui_template,
    point_cost,
    min_tier,
    is_active,
    sort_order,
    ui_config
) VALUES (
    'video-edit',
    'Video Editor',
    'Upload a video and describe your edit in plain English — Grok rewrites it instantly. Change backgrounds, add accessories, adjust lighting, and more.',
    'video',
    'Wand2',
    'video-editor',
    250,
    'gold',
    true,
    31,
    '{
        "prompt_placeholder": "Describe your edit — e.g. Give her a gold necklace, change the background to a beach",
        "max_file_mb": 100,
        "generation_warning": "Input video must be 8.7 seconds or shorter. Output keeps the same duration and aspect ratio.",
        "output_hint": "Powered by Grok Imagine — the most advanced AI video editing model available"
    }'::jsonb
)
ON CONFLICT (slug) DO UPDATE SET
    name        = EXCLUDED.name,
    description = EXCLUDED.description,
    ui_template = EXCLUDED.ui_template,
    point_cost  = EXCLUDED.point_cost,
    min_tier    = EXCLUDED.min_tier,
    is_active   = EXCLUDED.is_active,
    ui_config   = EXCLUDED.ui_config;
