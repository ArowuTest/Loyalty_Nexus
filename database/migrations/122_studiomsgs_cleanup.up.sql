-- Migration 122: Clean up legacy SSE-prefixed chat_messages rows (STU-002)
--
-- Before the STU-002 fix, the streaming handler stored the raw SSE wire
-- format ("data: {…}") into the content column instead of the parsed JSON.
-- Those rows cause JSON parse errors in the frontend on page load:
--   Uncaught SyntaxError: Unexpected non-whitespace character after JSON at position 4
--
-- This migration strips the leading "data: " prefix so existing rows parse
-- correctly.  Idempotent: only rows still carrying the prefix are affected.
--
-- Fix: table is chat_messages (not studio_messages); chat_messages has no
-- updated_at column (only created_at) so that SET clause is removed.

-- Strip 6-char "data: " prefix (data + colon + space)
UPDATE chat_messages
SET    content = SUBSTR(content, 7)
WHERE  content LIKE 'data: {%';

-- Strip 5-char "data:" prefix (no space variant)
UPDATE chat_messages
SET    content = SUBSTR(content, 6)
WHERE  content LIKE 'data:{%';
