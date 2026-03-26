PRAGMA foreign_keys = ON;

ALTER TABLE messages ADD COLUMN ciphertext TEXT NOT NULL DEFAULT '';
ALTER TABLE messages ADD COLUMN encryption_version TEXT NOT NULL DEFAULT '';
ALTER TABLE messages ADD COLUMN sender_device_id INTEGER NOT NULL DEFAULT 0;
ALTER TABLE messages ADD COLUMN recipient_device_id INTEGER NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_messages_sender_device_id
ON messages (sender_device_id);

CREATE INDEX IF NOT EXISTS idx_messages_recipient_device_id
ON messages (recipient_device_id);
