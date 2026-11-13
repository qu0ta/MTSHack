ALTER TABLE chats ADD COLUMN client_data jsonb NOT NULL DEFAULT '{}'::jsonb;
