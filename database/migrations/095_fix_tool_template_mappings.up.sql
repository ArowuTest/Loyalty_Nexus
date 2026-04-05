-- Migration 095: Fix 9 incorrect ui_template assignments
--
-- Verified via live API audit on 2026-04-05.
-- 41/50 tools were already correct. These 9 had wrong templates causing
-- the wrong input UI to render for users.
--
-- Fix summary:
--   1. animate-my-photo   KnowledgeDoc  → video-animator  (image-to-video, needs image upload)
--   2. background-remover KnowledgeDoc  → image-compose   (bg removal, needs image upload)
--   3. bg-remover         ImageEditor   → image-compose   (bg removal, not editing)
--   4. my-ai-photo        KnowledgeDoc  → image-creator   (text-to-image, needs prompt + style)
--   5. my-marketing-jingle KnowledgeDoc → music-composer  (jingle generation, needs music controls)
--   6. my-podcast         KnowledgeDoc  → music-composer  (podcast audio, needs music controls)
--   7. my-video-story     KnowledgeDoc  → video-animator  (image-to-video, needs image upload)
--   8. text-to-speech     KnowledgeDoc  → voice-studio    (TTS, needs voice picker + speed)
--   9. video-cinematic    video-animator → video-creator  (text-to-video, not image-to-video)

-- ── 1. animate-my-photo → video-animator ─────────────────────────────────────
UPDATE studio_tools
SET    ui_template = 'video-animator',
       updated_at  = NOW()
WHERE  slug = 'animate-my-photo';

-- ── 2. background-remover → image-compose ────────────────────────────────────
UPDATE studio_tools
SET    ui_template = 'image-compose',
       updated_at  = NOW()
WHERE  slug = 'background-remover';

-- ── 3. bg-remover → image-compose ────────────────────────────────────────────
UPDATE studio_tools
SET    ui_template = 'image-compose',
       updated_at  = NOW()
WHERE  slug = 'bg-remover';

-- ── 4. my-ai-photo → image-creator ───────────────────────────────────────────
UPDATE studio_tools
SET    ui_template = 'image-creator',
       updated_at  = NOW()
WHERE  slug = 'my-ai-photo';

-- ── 5. my-marketing-jingle → music-composer ──────────────────────────────────
UPDATE studio_tools
SET    ui_template = 'music-composer',
       updated_at  = NOW()
WHERE  slug = 'my-marketing-jingle';

-- ── 6. my-podcast → music-composer ───────────────────────────────────────────
UPDATE studio_tools
SET    ui_template = 'music-composer',
       updated_at  = NOW()
WHERE  slug = 'my-podcast';

-- ── 7. my-video-story → video-animator ───────────────────────────────────────
UPDATE studio_tools
SET    ui_template = 'video-animator',
       updated_at  = NOW()
WHERE  slug = 'my-video-story';

-- ── 8. text-to-speech → voice-studio ─────────────────────────────────────────
UPDATE studio_tools
SET    ui_template = 'voice-studio',
       updated_at  = NOW()
WHERE  slug = 'text-to-speech';

-- ── 9. video-cinematic → video-creator ───────────────────────────────────────
UPDATE studio_tools
SET    ui_template = 'video-creator',
       updated_at  = NOW()
WHERE  slug = 'video-cinematic';
