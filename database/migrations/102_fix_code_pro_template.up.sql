-- Migration 102: Fix Nexus Code Pro UI template
-- Changes ui_template from 'VisionAsk' to 'CodePro' for the code-pro slug.
--
-- Rationale:
--   The VisionAsk template requires an image to be uploaded before the
--   question textarea appears. For a code assistant, the text question must
--   always be visible and the image upload should be optional (collapsible).
--   The new CodePro template provides this correct UX:
--     1. Code question textarea (always visible, required)
--     2. Example question chips
--     3. Optional screenshot upload (collapsible panel)
--     4. Generate button (violet/purple theme to distinguish from other tools)
--
-- Idempotent — safe to run multiple times.
-- ============================================================

UPDATE studio_tools
SET
    ui_template = 'CodePro',
    updated_at  = NOW()
WHERE slug = 'code-pro';
