-- Migration 039: Cost corrections and point price fixes
-- Audit date: 2026-03-27
--
-- ai-photo-dream uses seedream5 model (Pollinations).
-- Cost analysis: seedream5 = $0.01/image = 10,000 micros.
-- Pulse Point exchange: ~1pt ≈ $0.00126 (₦1,000 / 500pts / 1.59 USD/NGN).
-- Cost in pts: $0.01 / $0.00126 ≈ 7.9 pts. Current price: 8 pts (breakeven).
-- Raise to 12 pts to ensure 50% margin, aligning with platform profitability model.
--
-- Comparison: ai-photo-pro (gptimage, $0.02) = 10 pts → 59% margin. 
-- We bring ai-photo-dream in line at similar margin.

UPDATE studio_tools
SET    point_cost = 12,
       updated_at = NOW()
WHERE  slug = 'ai-photo-dream';

-- Log the audit trail
DO $$
BEGIN
  RAISE NOTICE 'Migration 039: ai-photo-dream point_cost raised from 8 → 12 (seedream5 cost coverage + margin)';
END;
$$;
