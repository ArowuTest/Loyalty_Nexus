-- 078_chat_tool_slug.down.sql
ALTER TABLE chat_sessions DROP COLUMN IF EXISTS tool_slug;
ALTER TABLE session_summaries DROP COLUMN IF EXISTS tool_slug;
DROP INDEX IF EXISTS idx_chat_sessions_user_tool;
DROP INDEX IF EXISTS idx_session_summaries_user_tool;
DROP INDEX IF EXISTS idx_chat_messages_session_created;
