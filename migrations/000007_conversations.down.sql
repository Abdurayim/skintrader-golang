-- 000007_conversations.down.sql
-- Drop conversations table

DROP TRIGGER IF EXISTS set_conversations_updated_at ON conversations;
DROP TABLE IF EXISTS conversations;
