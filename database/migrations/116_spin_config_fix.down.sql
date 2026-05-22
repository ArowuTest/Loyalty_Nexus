-- Migration 116 rollback: Restore removed duplicate spin limit config key

INSERT INTO network_configs (key, value, description, created_at, updated_at)
SELECT
    'spin_max_per_user_per_day',
    '3',
    'Legacy per-user daily spin limit key restored by rollback',
    NOW(),
    NOW()
WHERE NOT EXISTS (
    SELECT 1 FROM network_configs WHERE key = 'spin_max_per_user_per_day'
);
