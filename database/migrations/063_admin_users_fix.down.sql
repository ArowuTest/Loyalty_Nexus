-- 063 down: revert admin email (keep table structure changes)
UPDATE admin_users SET email = NULL WHERE email = 'admin@loyaltynexus.ng';
