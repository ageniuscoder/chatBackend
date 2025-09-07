PRAGMA foreign_keys=ON;

-- USERS TABLE
CREATE TABLE IF NOT EXISTS users(
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL UNIQUE,
    phone_number TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    profile_pic TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- OTP CODES (used for signup, reset)
CREATE TABLE IF NOT EXISTS otp_codes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    phone_number TEXT NOT NULL,
    code TEXT NOT NULL,
    purpose TEXT NOT NULL CHECK (purpose IN ('signup','reset')),
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- CONVERSATIONS (1-to-1 or group)
CREATE TABLE IF NOT EXISTS conversations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT, -- null for private chats
    is_group_chat BOOLEAN NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- PARTICIPANTS (membership of users in conversations)
CREATE TABLE IF NOT EXISTS participants (
    conversation_id INTEGER NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    is_admin BOOLEAN NOT NULL DEFAULT 0,
    joined_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (conversation_id, user_id) -- composite PK prevents duplicates
);

-- MESSAGES (content sent by users)
CREATE TABLE IF NOT EXISTS messages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    conversation_id INTEGER NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    sender_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    sent_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- MESSAGE STATUS (delivery + read tracking)
CREATE TABLE IF NOT EXISTS message_status (
    message_id INTEGER NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status TEXT NOT NULL CHECK(status IN ('delivered','read')),
    read_at TIMESTAMP,
    PRIMARY KEY (message_id, user_id) -- ensures one row per recipient per message
);

-- INDEXES FOR PERFORMANCE
CREATE INDEX IF NOT EXISTS idx_messages_conversation_time 
    ON messages(conversation_id, sent_at DESC);

CREATE INDEX IF NOT EXISTS idx_participants_user 
    ON participants(user_id);

CREATE INDEX IF NOT EXISTS idx_message_status_user 
    ON message_status(user_id);
