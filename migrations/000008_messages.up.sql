-- 000008_messages.up.sql
-- Create messages enum, table, and indexes

-- Enum
CREATE TYPE message_status AS ENUM ('sent', 'delivered', 'read');

-- Messages table
CREATE TABLE messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    sender_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content VARCHAR(1000) NOT NULL,
    status message_status NOT NULL DEFAULT 'sent',
    read_at TIMESTAMPTZ,
    post_id UUID REFERENCES posts(id),
    deleted_at TIMESTAMPTZ,
    deleted_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_messages_conversation_created ON messages(conversation_id, created_at DESC);
CREATE INDEX idx_messages_sender_created ON messages(sender_id, created_at);
CREATE INDEX idx_messages_unread ON messages(conversation_id, status) WHERE status != 'read' AND deleted_at IS NULL;
