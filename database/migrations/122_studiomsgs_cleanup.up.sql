-- Migration 122: Clean up legacy SSE-prefixed studio_messages rows (STU-002)
--
-- Before the STU-002 fix, the streaming handler stored the raw SSE wire
-- format ("data: {…}") into the content column instead of the parsed JSON.
-- Those rows cause JSON parse errors in the frontend on page load:
--   Uncaught SyntaxError: Unexpected non-whitespace character after JSON at position 4
--
-- This migration strips the leading "data: " prefix so existing rows parse
-- correctly.  Idempotent: only rows still carrying the prefix are affected.

-- Strip 6-char "data: " prefix (data + colon + space)
UPDATE studio_messages
SET    content    = SUBSTR(content, 7),
       updated_at = NOW()
WHERE  content LIKE 'data: {%';

-- Strip 5-char "data:" prefix (no space variant)
UPDATE studio_messages
SET    content    = SUBSTR(content, 6),
       updated_at = NOW()
WHERE  content LIKE 'data:{%';
