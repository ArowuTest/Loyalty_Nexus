-- 090_fix_tool_ui_templates.up.sql
-- Purpose: Correct ui_template assignments that were wrong in earlier migrations.
--
-- Issues fixed:
--   1. nexus-chat / ask-nexus — were set to 'KnowledgeDoc' (a generate template).
--      These are conversational tools and must use the 'chat' template so the
--      frontend routes them to the Chat tab instead of the ToolDrawer.
--   2. translate — was set to 'VoiceStudio' (a TTS template). Translate is a
--      text-in / text-out tool and belongs in 'knowledge-doc'.
--   3. video-premium — was set to 'VideoAnimator' (image-to-video). It is a
--      text-to-video tool and must use 'video-creator'.
--   4. video-jingle — was set to 'VideoCreator' (text-to-video). It is an
--      image/audio-to-video tool and must use 'video-animator'.

-- ── 1. Chat tools → 'chat' template ─────────────────────────────────────────
UPDATE studio_tools
SET    ui_template = 'chat'
WHERE  slug IN ('nexus-chat', 'ask-nexus', 'ai-chat');

-- ── 2. Translate → 'knowledge-doc' template ──────────────────────────────────
UPDATE studio_tools
SET    ui_template = 'knowledge-doc'
WHERE  slug = 'translate';

-- ── 3. video-premium → 'video-creator' (text-to-video) ───────────────────────
UPDATE studio_tools
SET    ui_template = 'video-creator'
WHERE  slug = 'video-premium';

-- ── 4. video-jingle → 'video-animator' (image/audio-to-video) ────────────────
UPDATE studio_tools
SET    ui_template = 'video-animator'
WHERE  slug = 'video-jingle';
