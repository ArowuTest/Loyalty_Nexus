-- Migration 107: Remove dead/orphan studio tool slugs
-- These slugs were inserted by early migrations (NotebookLM era) and have
-- never had a working backend implementation. Each has a functioning
-- replacement already live in the current system.
--
-- Removed slugs and their replacements:
--   quiz-gen     → quiz (Quiz Generator, fully functional)
--   mindmap-gen  → mindmap (Mind Map, fully functional)
--   summarise    → ask-nexus / web-search-ai (chat-based, no dedicated tool needed)
--   essay        → ask-nexus (general writing assistant)
--   email-writer → ask-nexus (email writing via chat)
--   cv-writer    → ask-nexus (CV/resume writing via chat)

DELETE FROM studio_tools WHERE slug IN (
    'quiz-gen',
    'mindmap-gen',
    'summarise',
    'essay',
    'email-writer',
    'cv-writer'
);
