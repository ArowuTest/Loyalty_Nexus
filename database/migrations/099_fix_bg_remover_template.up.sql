-- Migration 099: Fix bg-remover ui_template
--
-- Migration 096 incorrectly changed bg-remover from 'image-editor' to 'image-compose'.
-- The image-compose template is the Whisk-style 3-slot Subject/Scene/Style composer
-- which is completely wrong for a background remover.
--
-- bg-remover should use 'image-editor' with show_edit_prompt=false so the user
-- just uploads an image and clicks Generate (no text instruction needed).

UPDATE studio_tools
SET
  ui_template = 'image-editor',
  ui_config   = '{
    "upload_label": "Upload the image to remove background from",
    "upload_accept": ["image/png","image/jpeg","image/webp"],
    "show_edit_prompt": false,
    "prompt_optional": true,
    "max_file_mb": 10,
    "output_note": "Output is a transparent PNG with the background removed."
  }'::jsonb,
  updated_at  = NOW()
WHERE slug IN ('bg-remover', 'background-remover');
