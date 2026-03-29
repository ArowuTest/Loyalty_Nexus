-- 061 down: remove seeded test data only (columns are safe to keep)
DELETE FROM admin_users WHERE email = 'admin@loyaltynexus.ng';
DELETE FROM users WHERE phone_number IN (
    '+2348027000000', '+2348020000000', '+2348023000000',
    '+2348025000000', '+2348029000000'
);
