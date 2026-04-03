-- Migration 087: Add Video Editor tool (Grok Imagine natural language video editing)
-- Uses xAI Grok Imagine API: upload a video, describe the edit in plain English
-- Grok rewrites the video preserving original duration and aspect ratio
-- Grok API cost: $0.05/sec x ~6s = ~$0.30 per generation
INSERT INTO studio_tools (
    id,
    name,
    slug,
    description,
    category,
    point_cost,
    provider,
    provider_tool,
    is_active,
    is_free,
    icon,
    sort_order,
    entry_point_cost,
    ui_template,
    ui_config,
    created_at,
    updated_at
) VALUES (
    gen_random_uuid(),
    'Video Editor',
    'video-edit',
    'Upload a video and describe your edit in plain English — Grok rewrites it instantly. Change backgrounds, add accessories, adjust lighting, and more.',
    'Create',
    250,
    'xai',
    'grok-imagine-video',
    true,
    false,
    '✏️',
    65,
    75,
    'video-editor',
    '{
        "prompt_placeholder": "Describe your edit — e.g. Give her a gold necklace, change the background to a beach",
        "max_file_mb": 100,
        "generation_warning": "Input video must be 8.7 seconds or shorter. Output keeps the same duration and aspect ratio.",
        "output_hint": "Powered by Grok Imagine — the most advanced AI video editing model available"
    }'::jsonb,
    NOW(),
    NOW()
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
