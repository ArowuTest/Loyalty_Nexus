-- Migration 133: make the AI attempt ledger DURABLE and APPEND-ONLY.
--
-- WHY
-- 131 declared ai_generation_attempts.generation_id ... ON DELETE CASCADE, so the
-- routine retention sweep (lifecycle_worker asset-expiry → DeleteGeneration)
-- cascade-erased the per-attempt cost/outcome/latency history that the V2
-- architecture mandates as "immutable per-attempt telemetry for audit, health
-- and cost attribution" (review B2). Every OTHER foreign key on the table is
-- ON DELETE SET NULL, and the row denormalizes provider_slug / model_id /
-- cost_micros precisely so attribution survives upstream deletion — this brings
-- generation_id in line with that design.
--
-- It also enforces append-only at the DB level (review M2): once an attempt is
-- terminal (any outcome but STARTED) it can no longer be UPDATEd, regardless of
-- caller. The only permitted transition is STARTED → terminal — the router's
-- finalize, and the lifecycle worker's stranded-attempt reconciliation.
-- Deletion is deliberately NOT blocked: retention is an explicit policy decision.
--
-- Idempotent: a no-op-equivalent on a fresh install (131 just ran) and a fix on
-- any database that already ran 131 with CASCADE.

-- ── 1. generation_id: CASCADE → SET NULL ─────────────────────────────────────
ALTER TABLE ai_generation_attempts
    DROP CONSTRAINT IF EXISTS ai_generation_attempts_generation_id_fkey;
ALTER TABLE ai_generation_attempts
    ADD CONSTRAINT ai_generation_attempts_generation_id_fkey
    FOREIGN KEY (generation_id) REFERENCES ai_generations(id) ON DELETE SET NULL;

-- ── 2. Append-only guard: terminal attempts are immutable ────────────────────
-- One transition is exempt: Postgres implements the ON DELETE SET NULL above as
-- an UPDATE of the referencing row, so an unlink that changes generation_id to
-- NULL and NOTHING else must pass or every generation delete would be blocked.
-- The exemption is exactly that shape — an unlink bundled with any other change
-- is still rejected.
CREATE OR REPLACE FUNCTION ai_generation_attempts_append_only() RETURNS trigger AS $$
BEGIN
    IF OLD.outcome <> 'STARTED' THEN
        IF NEW.generation_id IS NULL
           AND OLD.generation_id IS NOT NULL
           AND (to_jsonb(NEW) - 'generation_id') = (to_jsonb(OLD) - 'generation_id') THEN
            RETURN NEW;  -- FK ON DELETE SET NULL from ai_generations
        END IF;
        RAISE EXCEPTION
            'ai_generation_attempts is append-only: terminal attempt % (%) cannot be modified',
            OLD.id, OLD.outcome;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_ai_generation_attempts_append_only ON ai_generation_attempts;
CREATE TRIGGER trg_ai_generation_attempts_append_only
    BEFORE UPDATE ON ai_generation_attempts
    FOR EACH ROW EXECUTE FUNCTION ai_generation_attempts_append_only();
