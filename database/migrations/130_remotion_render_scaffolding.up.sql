-- Migration 130: Remotion "Video Templates" scaffolding (Track A, Phase 1 groundwork).
--
-- Ships the first templated-video tool + provider row INACTIVE. The tool is
-- is_active=false, so ListActiveTools hides it from users — nothing is surfaced
-- until the self-hosted render-service exists and RENDER_SERVICE_URL is set.
-- Flip is_active=true (admin) once the render-service is live to launch it.
--
-- Render target is host-portable behind RENDER_SERVICE_URL: self-hosted on
-- Render now, the same container on GCP Cloud Run later — no code change.

-- ── Tool row (INACTIVE / hidden) ──────────────────────────────────────────────
INSERT INTO studio_tools (
    id, name, slug, description, category, point_cost, provider, provider_tool,
    is_active, is_free, icon, sort_order, entry_point_cost, ui_template, ui_config,
    created_at, updated_at
) VALUES (
    gen_random_uuid(),
    'Video Slideshow',
    'video-slideshow',
    'Turn your AI images into a polished video montage with transitions, captions and music — cheap, templated video.',
    'Create',
    15,                        -- PLACEHOLDER PulsePoints — tune in admin
    'remotion',
    'video-slideshow',
    false,                     -- INACTIVE: hidden until the render-service is live
    false,
    '🎞️',
    72,
    0,
    'video-slideshow',
    '{
        "prompt_placeholder": "Optional caption or theme for the montage",
        "max_images": 6,
        "min_images": 3,
        "aspect_ratios": [
            { "label": "Portrait", "value": "9:16" },
            { "label": "Square",   "value": "1:1"  },
            { "label": "Landscape","value": "16:9" }
        ],
        "coming_soon_note": "Video Templates launching soon."
    }'::jsonb,
    NOW(), NOW()
)
ON CONFLICT (slug) DO UPDATE SET
    ui_template = EXCLUDED.ui_template,
    ui_config   = EXCLUDED.ui_config,
    updated_at  = NOW();

-- ── Provider registry row (INACTIVE) — for when the DB-driven render path is wired ──
INSERT INTO ai_provider_configs
    (name, slug, category, template, env_key, model_id, priority, is_primary, is_active, cost_micros, pulse_pts, notes)
VALUES
    ('Remotion (self-hosted)', 'remotion-selfhosted', 'render', 'remotion', 'RENDER_SERVICE_URL', 'video-slideshow', 1, true, false, 1000, 0, 'DORMANT — self-hosted Remotion render-service (Render now, GCP Cloud Run later). Flip active when the service is live.')
ON CONFLICT (slug) DO NOTHING;
