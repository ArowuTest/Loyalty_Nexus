-- Migration 086: Add Image Composer tool (Whisk-style multi-reference composition)
-- Uses FAL.ai fal-ai/flux-pro/v1.1-ultra with subject image_url + composition prompt
-- Supports optional scene and style reference images, and 1-4 output variations
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
    'Image Composer',
    'image-compose',
    'Upload a subject, scene, and style reference — AI composes them into a stunning new image. Inspired by Google Whisk.',
    'Create',
    150,
    'fal',
    'fal-ai/flux-pro/v1.1-ultra',
    true,
    false,
    '🖼️',
    46,
    50,
    'image-compose',
    '{
        "prompt_placeholder": "Describe how to compose the images — e.g. Place the subject in the scene with cinematic lighting",
        "show_style_tags": false,
        "show_negative_prompt": false,
        "show_quality_toggle": false,
        "output_hint": "AI will compose your reference images into a new image using Flux Pro 1.1 Ultra"
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
