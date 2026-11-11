CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE chats
(
    id         UUID PRIMARY KEY     DEFAULT gen_random_uuid(),
    title      VARCHAR(255),
    status     VARCHAR(20) NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'closed')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    closed_at  TIMESTAMPTZ
);

CREATE TABLE messages
(
    id               UUID PRIMARY KEY     DEFAULT gen_random_uuid(),
    chat_id          UUID        NOT NULL REFERENCES chats (id) ON DELETE CASCADE,
    role             VARCHAR(20) NOT NULL
        CHECK (role IN ('user', 'assistant')),
    content          TEXT        NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    response_time_ms INTEGER
        CHECK (response_time_ms IS NULL OR response_time_ms >= 0),
    metadata         JSONB       NOT NULL DEFAULT '{}'::jsonb
);

CREATE TABLE applications
(
    id         UUID PRIMARY KEY      DEFAULT gen_random_uuid(),
    chat_id    UUID         NOT NULL UNIQUE REFERENCES chats (id) ON DELETE CASCADE,
    full_name  VARCHAR(255) NOT NULL,
    phone      VARCHAR(30)  NOT NULL,
    company    VARCHAR(255) NOT NULL,
    summary    TEXT         NOT NULL,
    status     VARCHAR(20)  NOT NULL DEFAULT 'new'
        CHECK (status IN ('new', 'processed')),
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- Индексы
CREATE INDEX idx_chats_updated_at
    ON chats (updated_at DESC);

CREATE INDEX idx_messages_chat_created
    ON messages (chat_id, created_at);

CREATE INDEX idx_applications_created_at
    ON applications (created_at DESC);
