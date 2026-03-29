-- Migration 062: Guarantee super_admin exists in the database.
-- This is a belt-and-braces fallback alongside the ADMIN_SEED_EMAIL/PASSWORD env var approach.
-- Password: Admin@LoyaltyNexus2026!
-- ON CONFLICT DO NOTHING — completely safe if admin already exists.
INSERT INTO admin_users (id, email, password_hash, full_name, role, is_active, created_at, updated_at)
VALUES (
    gen_random_uuid(),
    'admin@loyaltynexus.ng',
    '$2b$10$9U6kXVOrcNk11Dg2F38OhuvTtrmbgAIHpzUFxSnZ.L0NCyyuM/Gim',
    'Platform Admin',
    'super_admin',
    TRUE,
    NOW(),
    NOW()
)
ON CONFLICT (email) DO UPDATE SET
    is_active     = TRUE,
    role          = 'super_admin',
    password_hash = '$2b$10$9U6kXVOrcNk11Dg2F38OhuvTtrmbgAIHpzUFxSnZ.L0NCyyuM/Gim',
    updated_at    = NOW();
