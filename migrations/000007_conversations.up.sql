-- 000007_conversations.up.sql
-- Create conversations table with indexes and trigger

CREATE TABLE conversations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    participant_1 UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    participant_2 UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    last_message_content VARCHAR(100),
    last_message_sender UUID,
    last_message_at TIMESTAMPTZ,
    unread_count_1 INTEGER NOT NULL DEFAULT 0,
    unread_count_2 INTEGER NOT NULL DEFAULT 0,
    initial_post_id UUID REFERENCES posts(id),
    deleted_for_1_at TIMESTAMPTZ,
    deleted_for_2_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (participant_1, participant_2),
    CHECK (participant_1 < participant_2)
);

-- Indexes
CREATE INDEX idx_conversations_participant_1_updated ON conversations(participant_1, updated_at);
CREATE INDEX idx_conversations_participant_2_updated ON conversations(participant_2, updated_at);
CREATE INDEX idx_conversations_updated_at ON conversations(updated_at DESC);

-- Update trigger
CREATE TRIGGER set_conversations_updated_at
    BEFORE UPDATE ON conversations
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
