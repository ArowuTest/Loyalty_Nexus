-- Migration 034 — Fix code-helper category + web-search-ai chat routing
-- code-helper uses qwen-coder via Pollinations chat API — it is a chat-mode
-- tool and belongs in 'Chat' alongside web-search-ai and ai-chat.
-- Keeping it in 'Build' hides it from the Chat tab and confuses users.

UPDATE studio_tools
SET    category   = 'Chat',
       sort_order = 22,
       updated_at = NOW()
WHERE  slug = 'code-helper';

-- Also confirm web-search-ai stays in Chat (idempotent)
UPDATE studio_tools
SET    category   = 'Chat',
       sort_order = 18,
       updated_at = NOW()
WHERE  slug = 'web-search-ai';

-- Result: Chat tab now contains 3 tools:
--   ai-chat        (sort 17) — general assistant
--   web-search-ai  (sort 18) — live internet answers
--   code-helper    (sort 22) — Qwen Coder via Pollinations
