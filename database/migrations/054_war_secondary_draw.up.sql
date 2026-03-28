-- Migration 054: Regional Wars — Secondary Draw tables
-- After a war is resolved, admin can run one secondary draw per winning state.
-- All participants are users in that state who were active during the war window.
-- Winners are selected via CSPRNG Fisher-Yates (same engine as main draw — SEC-009).
-- Prizes are paid via MoMo Cash by admin manually after draw execution.

BEGIN;

-- ── war_secondary_draws ───────────────────────────────────────────────────────
-- One row per secondary draw execution (admin can run at most once per state per war).

CREATE TABLE IF NOT EXISTS war_secondary_draws (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    war_id          UUID        NOT NULL REFERENCES regional_wars(id) ON DELETE CASCADE,
    state           TEXT        NOT NULL,
    winner_count    INT         NOT NULL DEFAULT 1 CHECK (winner_count BETWEEN 1 AND 10),
    prize_per_winner_kobo BIGINT NOT NULL DEFAULT 0,  -- e.g. 50000 = ₦500
    total_pool_kobo BIGINT      NOT NULL DEFAULT 0,   -- winner_count * prize_per_winner_kobo
    participant_count INT       NOT NULL DEFAULT 0,   -- eligible users at time of draw
    status          TEXT        NOT NULL DEFAULT 'PENDING'
                                CHECK (status IN ('PENDING','COMPLETED','CANCELLED')),
    triggered_by    UUID        REFERENCES admin_users(id),
    executed_at     TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- Only one draw per (war, state) — admin cannot re-run
    UNIQUE (war_id, state)
);

CREATE INDEX IF NOT EXISTS idx_war_sec_draws_war_id ON war_secondary_draws(war_id);
CREATE INDEX IF NOT EXISTS idx_war_sec_draws_state  ON war_secondary_draws(state);

-- ── war_secondary_draw_winners ────────────────────────────────────────────────
-- One row per winner selected in the secondary draw.

CREATE TABLE IF NOT EXISTS war_secondary_draw_winners (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    secondary_draw_id UUID      NOT NULL REFERENCES war_secondary_draws(id) ON DELETE CASCADE,
    war_id          UUID        NOT NULL,
    state           TEXT        NOT NULL,
    user_id         UUID        NOT NULL REFERENCES users(id),
    phone_number    TEXT        NOT NULL,
    position        INT         NOT NULL,            -- 1 = first winner
    prize_kobo      BIGINT      NOT NULL DEFAULT 0,
    momo_number     TEXT,                            -- filled when admin pays
    payment_status  TEXT        NOT NULL DEFAULT 'PENDING_PAYMENT'
                                CHECK (payment_status IN ('PENDING_PAYMENT','PAID','FAILED')),
    paid_at         TIMESTAMPTZ,
    paid_by         UUID        REFERENCES admin_users(id),
    notes           TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_war_sec_winners_draw_id ON war_secondary_draw_winners(secondary_draw_id);
CREATE INDEX IF NOT EXISTS idx_war_sec_winners_user_id ON war_secondary_draw_winners(user_id);

-- ── Triggers for updated_at ───────────────────────────────────────────────────

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_war_sec_draws_updated_at ON war_secondary_draws;
CREATE TRIGGER trg_war_sec_draws_updated_at
    BEFORE UPDATE ON war_secondary_draws
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS trg_war_sec_winners_updated_at ON war_secondary_draw_winners;
CREATE TRIGGER trg_war_sec_winners_updated_at
    BEFORE UPDATE ON war_secondary_draw_winners
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ── Config keys ───────────────────────────────────────────────────────────────

INSERT INTO network_configs (key, value, description) VALUES
    ('wars_secondary_draw_default_winners',    '3',       'Default number of winners per state secondary draw (1-10)'),
    ('wars_secondary_draw_default_prize_kobo', '50000',   'Default prize per winner in kobo (50000 = ₦500)')
ON CONFLICT (key) DO NOTHING;

COMMIT;
