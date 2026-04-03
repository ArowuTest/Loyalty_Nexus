-- Migration 085: Add Video Story Builder (multi-scene image-to-video) tool
-- Uses FAL.ai fal-ai/kling-video/v1.6/standard/multi-image-to-video
-- Supports up to 4 images, each with its own scene description
-- Tier: Gold and above (same as video-premium)

-- Insert the new tool
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
    'video-story',
    'Video Story Builder',
    'Upload up to 4 images and describe each scene — AI weaves them into a single cinematic video with smooth transitions.',
    'video',
    'Film',
    'video-multi-scene',
    800,
    'gold',
    true,
    55,
    '{
        "show_multi_image": true,
        "max_images": 4,
        "show_duration": true,
        "duration_options": [5, 10],
        "show_aspect_ratio": true,
        "aspect_ratio_options": ["16:9", "9:16", "1:1"],
        "show_motion_style": false,
        "show_motion_intensity": false,
        "show_audio_toggle": false,
        "show_end_image": false,
        "placeholder_prompt": "Describe the overall story or mood of the video...",
        "submit_label": "Build Story Video →"
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
