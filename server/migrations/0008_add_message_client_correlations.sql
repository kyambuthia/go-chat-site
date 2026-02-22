PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS message_client_correlations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    sender_user_id INTEGER NOT NULL,
    recipient_user_id INTEGER NOT NULL,
    client_message_id INTEGER NOT NULL,
    stored_message_id INTEGER NOT NULL,
    delivered INTEGER NOT NULL DEFAULT 0 CHECK (delivered IN (0, 1)),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (sender_user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (recipient_user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (stored_message_id) REFERENCES messages(id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_message_client_correlations_sender_client
ON message_client_correlations (sender_user_id, client_message_id);

CREATE INDEX IF NOT EXISTS idx_message_client_correlations_stored_message
ON message_client_correlations (stored_message_id);
