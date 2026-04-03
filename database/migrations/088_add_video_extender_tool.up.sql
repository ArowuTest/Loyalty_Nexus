-- Migration 088: Add Video Extender tool (Grok Imagine video extension)
-- Uses xAI Grok Imagine video extension API: upload a video (2-15s), Grok continues it
-- seamlessly from the last frame, adding 2-10 more seconds of AI-generated content
-- Tier: Gold and above (Grok API cost: $0.05/sec × extension duration)

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
    'video-extend',
    'Video Extender',
    'Upload a video and Grok seamlessly continues it from the last frame — add up to 10 more seconds of AI-generated content with no visible cuts.',
    'video',
    'ArrowRight',
    'video-extender',
    200,
    'gold',
    true,
    32,
    '{
        "prompt_placeholder": "Describe what happens next — or leave blank for a natural continuation",
        "max_file_mb": 100,
        "default_duration": 6,
        "duration_options": [2, 4, 6, 8, 10],
        "generation_warning": "Input video must be 2–15 seconds. Extension adds 2–10 more seconds seamlessly.",
        "output_hint": "Powered by Grok Imagine — seamless video continuation from the last frame"
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
