-- Migration 088: Add Video Extender tool (Grok Imagine video extension)
-- Uses xAI Grok Imagine video extension API: upload a video (2-15s), Grok continues it
-- seamlessly from the last frame, adding 2-10 more seconds of AI-generated content
-- Grok API cost: $0.05/sec x extension duration
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
    'Video Extender',
    'video-extend',
    'Upload a video and Grok seamlessly continues it from the last frame — add up to 10 more seconds of AI-generated content with no visible cuts.',
    'Create',
    200,
    'xai',
    'grok-imagine-video',
    true,
    false,
    '▶️',
    66,
    60,
    'video-extender',
    '{
        "prompt_placeholder": "Describe what happens next — or leave blank for a natural continuation",
        "max_file_mb": 100,
        "default_duration": 6,
        "duration_options": [2, 4, 6, 8, 10],
        "generation_warning": "Input video must be 2-15 seconds. Extension adds 2-10 more seconds seamlessly.",
        "output_hint": "Powered by Grok Imagine — seamless video continuation from the last frame"
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
