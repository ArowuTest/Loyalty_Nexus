-- 062 down: remove seeded admin (only if it was created by this migration)
DELETE FROM admin_users WHERE email = 'admin@loyaltynexus.ng';
