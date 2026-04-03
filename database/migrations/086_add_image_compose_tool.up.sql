-- Migration 086: Add Image Compose tool (Whisk-style multi-reference composition)
-- Uses FAL.ai fal-ai/flux-pro/v1.1-ultra with subject image_url + composition prompt
-- Supports optional scene and style reference images, and 1-4 output variations
-- Tier: Silver and above

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
    'image-compose',
    'Image Composer',
    'Upload a subject, scene, and style reference — AI composes them into a stunning new image. Inspired by Google Whisk.',
    'image',
    'Layers',
    'image-compose',
    150,
    'silver',
    true,
    22,
    '{
        "prompt_placeholder": "Describe how to compose the images — e.g. Place the subject in the scene with cinematic lighting",
        "show_style_tags": false,
        "show_negative_prompt": false,
        "show_quality_toggle": false,
        "output_hint": "AI will compose your reference images into a new image using Flux Pro 1.1 Ultra"
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
