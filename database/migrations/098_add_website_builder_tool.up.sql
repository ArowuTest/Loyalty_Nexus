-- Migration 098: Add website-builder tool
INSERT INTO studio_tools (
  id, name, slug, description, category,
  point_cost, provider, provider_tool,
  is_active, is_free, icon, sort_order, entry_point_cost,
  ui_template, ui_config, created_at, updated_at
) VALUES (
  gen_random_uuid(),
  'Website Builder',
  'website-builder',
  'Create a stunning, professional website for your business in minutes. Choose your type, fill in your details, add photos — get a real shareable link.',
  'Build',
  25,
  'gemini',
  'gemini-2.5-flash',
  true, false, '🌐', 1, 0,
  'website-builder',
  '{"max_photos": 6, "supported_types": ["shop","corporate","professional","restaurant","portfolio","events","church","education"]}',
  NOW(), NOW()
) ON CONFLICT (slug) DO NOTHING;
