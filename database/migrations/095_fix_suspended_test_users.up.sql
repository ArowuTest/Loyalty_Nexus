-- Migration 095: Fix suspended test user + verify test data integrity

-- Unsuspend test users that were incorrectly suspended
UPDATE users SET is_active = true WHERE phone_number IN (
  '2348025000000', '+2348025000000',
  '2348029000000', '+2348029000000',
  '2348020000000', '+2348020000000',
  '2348023000000', '+2348023000000',
  '2348027000000', '+2348027000000'
);

-- Ensure wallets exist for all test users (create if missing)
INSERT INTO wallets (user_id, pulse_points, spin_credits, lifetime_points,
  recharge_counter, spin_draw_counter, spin_counter, draw_counter,
  pulse_counter, daily_recharge_kobo, daily_spins_awarded, updated_at)
SELECT u.id, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, NOW()
FROM users u
WHERE u.phone_number IN (
  '2348025000000','2348029000000','2348020000000','2348023000000','2348027000000'
)
ON CONFLICT (user_id) DO NOTHING;

-- Sync wallet pulse_points and lifetime_points with users.total_points for test users
UPDATE wallets w
SET pulse_points     = u.total_points,
    lifetime_points  = u.total_points,
    updated_at       = NOW()
FROM users u
WHERE w.user_id = u.id
  AND u.phone_number IN (
    '2348027000000','2348020000000','2348023000000','2348025000000','2348029000000'
  );
