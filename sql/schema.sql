-- sql/schema.sql
-- Drop tables in a specific order to avoid foreign key constraints issues
DROP TABLE IF EXISTS otp_codes;
DROP TABLE IF EXISTS message_status;
DROP TABLE IF EXISTS messages;
DROP TABLE IF EXISTS participants;
DROP TABLE IF EXISTS conversations;
DROP TABLE IF EXISTS users;

-- USERS
CREATE TABLE IF NOT EXISTS users(
    id BIGSERIAL PRIMARY KEY,
    username TEXT NOT NULL UNIQUE,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    profile_pic TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    last_active TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- OTP CODES
CREATE TABLE IF NOT EXISTS otp_codes (
    id BIGSERIAL PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    code TEXT NOT NULL,
    purpose TEXT NOT NULL CHECK (purpose IN ('signup','reset')),
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- CONVERSATIONS
CREATE TABLE IF NOT EXISTS conversations (
    id BIGSERIAL PRIMARY KEY,
    name TEXT,
    is_group_chat BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- PARTICIPANTS
CREATE TABLE IF NOT EXISTS participants (
    conversation_id BIGINT NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    is_admin BOOLEAN NOT NULL DEFAULT FALSE,
    joined_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    PRIMARY KEY (conversation_id, user_id)
);

-- MESSAGES
CREATE TABLE IF NOT EXISTS messages (
    id BIGSERIAL PRIMARY KEY,
    conversation_id BIGINT NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    sender_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    sent_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE, -- Add this line for soft deletion
    edited_at TIMESTAMP WITH TIME ZONE -- Add this line for message edits
);

-- MESSAGE STATUS
CREATE TABLE IF NOT EXISTS message_status (
    message_id BIGINT NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status TEXT NOT NULL CHECK(status IN ('delivered','read')),
    read_at TIMESTAMP WITH TIME ZONE,
    PRIMARY KEY (message_id, user_id)
);

-- INDEXES
CREATE INDEX IF NOT EXISTS idx_messages_conversation_time
    ON messages(conversation_id, sent_at DESC);

CREATE INDEX IF NOT EXISTS idx_messages_sender
    ON messages(sender_id);

CREATE INDEX IF NOT EXISTS idx_participants_user
    ON participants(user_id);

CREATE INDEX IF NOT EXISTS idx_participants_conversation
    ON participants(conversation_id);

CREATE INDEX IF NOT EXISTS idx_message_status_user
    ON message_status(user_id);

CREATE INDEX IF NOT EXISTS idx_message_status_message
    ON message_status(message_id);

CREATE INDEX IF NOT EXISTS idx_conversations_is_group
    ON conversations(is_group_chat);