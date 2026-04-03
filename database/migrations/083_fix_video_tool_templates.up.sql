-- Migration 083: Fix video tool template assignments
--
-- video-cinematic: requires an image upload (image-to-video), so must use
--   video-animator template (which has the image upload zone), NOT video-creator
--   (which is text-to-video only).
--
-- video-jingle: composite tool — generates a video clip + jingle music track.
--   Uses video-creator template with a music_style field shown.
--   The backend now routes it through dispatchComposite → assembleVideoJingle.

UPDATE studio_tools
SET ui_template = 'video-animator',
    ui_config = ui_config || '{"show_motion_prompt": true, "motion_prompt_label": "Cinematic motion", "motion_prompt_placeholder": "Describe the camera movement and animation: slow zoom in, dramatic pan, particles floating...", "show_aspect_ratio": true, "show_duration": false}'::jsonb
WHERE slug = 'video-cinematic';

UPDATE studio_tools
SET ui_template = 'video-creator',
    ui_config = ui_config || '{"show_music_style": true, "show_image_upload": true, "image_upload_optional": true, "image_upload_label": "Reference image (optional)", "image_upload_hint": "Upload a brand image or scene to animate alongside the jingle", "show_aspect_ratio": true, "show_duration": false, "show_camera_movement": false, "show_scenes": false, "show_negative_prompt": false, "prompt_label": "Jingle concept", "prompt_placeholder": "Describe your brand, product, or the mood you want: energetic startup launch, warm family brand, tech product reveal..."}'::jsonb
WHERE slug = 'video-jingle';
