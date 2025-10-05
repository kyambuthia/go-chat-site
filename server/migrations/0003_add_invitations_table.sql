CREATE TABLE invitations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    inviter_id INTEGER NOT NULL,
    invitee_username TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending', -- pending, accepted, rejected
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (inviter_id) REFERENCES users (id)
);
