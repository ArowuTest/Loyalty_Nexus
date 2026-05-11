-- Migration 107 down: restore removed tool slugs (rollback only)
INSERT INTO studio_tools (id, name, slug, description, category, point_cost, is_enabled, created_at, updated_at)
VALUES
    (gen_random_uuid(), 'Quiz Me',      'quiz-gen',    'Generate quiz questions on any topic.',  'Learn', 2, false, NOW(), NOW()),
    (gen_random_uuid(), 'Mind Map',     'mindmap-gen', 'Create a visual mind map from any concept.', 'Learn', 2, false, NOW(), NOW()),
    (gen_random_uuid(), 'Summarise',    'summarise',   'Summarise any text or document.',        'Create', 0, false, NOW(), NOW()),
    (gen_random_uuid(), 'Essay Writer', 'essay',       'Write essays on any topic.',             'Create', 0, false, NOW(), NOW()),
    (gen_random_uuid(), 'Email Writer', 'email-writer','Write professional emails.',             'Create', 0, false, NOW(), NOW()),
    (gen_random_uuid(), 'CV Writer',    'cv-writer',   'Create a professional CV/resume.',       'Create', 0, false, NOW(), NOW())
ON CONFLICT (slug) DO NOTHING;
