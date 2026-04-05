-- Rollback 094 (partial — cannot un-delete prize pool rows or un-normalise phones)
ALTER TABLE wallets DROP COLUMN IF EXISTS draw_entries_today;
ALTER TABLE wallets DROP COLUMN IF EXISTS draw_entries_date;
UPDATE regional_wars SET status = 'ACTIVE', updated_at = NOW() WHERE period = '2026-03';
