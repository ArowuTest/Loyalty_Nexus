-- Rollback migration 099: Revert bg-remover template back to image-compose (as set by migration 096)
UPDATE studio_tools
SET
  ui_template = 'image-compose',
  updated_at  = NOW()
WHERE slug IN ('bg-remover', 'background-remover');
