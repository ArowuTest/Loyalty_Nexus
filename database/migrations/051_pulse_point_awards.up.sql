-- Migration 051: Bonus Pulse Point Awards
-- ─────────────────────────────────────────────────────────────────────────────
-- Super-admins can award bonus Pulse Points to individual users as part of
-- campaigns or incentive programmes.  Every award is recorded here for a full
-- audit trail (who awarded, to whom, how many, why, when).
--
-- The corresponding wallet credit and immutable ledger entry are written by the
-- application service in the same DB transaction, so the three records are
-- always consistent.
-- ─────────────────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS pulse_point_awards (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Recipient
    user_id         UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    phone_number    TEXT        NOT NULL,           -- denormalised for fast audit queries

    -- Award details
    points          BIGINT      NOT NULL CHECK (points > 0),
    campaign        TEXT        NOT NULL DEFAULT '', -- e.g. "Ramadan 2025", "Beta Tester"
    note            TEXT        NOT NULL DEFAULT '', -- free-text reason

    -- Who did it
    awarded_by      UUID        NOT NULL,           -- admin user_id (FK not enforced — admins may be deleted)
    awarded_by_name TEXT        NOT NULL DEFAULT '', -- denormalised display name at time of award

    -- Immutable back-reference to the ledger entry
    transaction_id  UUID        NOT NULL,           -- FK to transactions.id

    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Fast lookups by recipient
CREATE INDEX IF NOT EXISTS idx_ppa_user_id
    ON pulse_point_awards (user_id);

-- Fast lookups by phone (admin audit search)
CREATE INDEX IF NOT EXISTS idx_ppa_phone
    ON pulse_point_awards (phone_number);

-- Fast lookups by campaign
CREATE INDEX IF NOT EXISTS idx_ppa_campaign
    ON pulse_point_awards (campaign)
    WHERE campaign <> '';

-- Fast lookups by awarding admin
CREATE INDEX IF NOT EXISTS idx_ppa_awarded_by
    ON pulse_point_awards (awarded_by);

COMMENT ON TABLE pulse_point_awards IS
    'Immutable audit log of every bonus Pulse Point award made by a super-admin.';
COMMENT ON COLUMN pulse_point_awards.campaign IS
    'Optional campaign or incentive programme name, e.g. "Ramadan 2025".';
COMMENT ON COLUMN pulse_point_awards.awarded_by IS
    'UUID of the admin user who made the award (from JWT claims at request time).';
COMMENT ON COLUMN pulse_point_awards.transaction_id IS
    'Back-reference to the bonus ledger entry in the transactions table.';
