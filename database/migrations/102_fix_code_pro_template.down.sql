-- Migration 102 DOWN: Revert Nexus Code Pro UI template to VisionAsk
-- ============================================================

UPDATE studio_tools
SET
    ui_template = 'VisionAsk',
    updated_at  = NOW()
WHERE slug = 'code-pro';
