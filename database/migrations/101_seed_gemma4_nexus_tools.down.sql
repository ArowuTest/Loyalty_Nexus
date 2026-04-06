-- Migration 101 rollback: Remove Gemma 4 / Nexus AI Tools
-- Deletes the four new tools and their associated generations.

DELETE FROM ai_generations
WHERE tool_id IN (
    SELECT id FROM studio_tools
    WHERE slug IN ('code-pro', 'doc-analyzer', 'localize-ui', 'nexus-agent')
);

DELETE FROM studio_tools
WHERE slug IN ('code-pro', 'doc-analyzer', 'localize-ui', 'nexus-agent');
