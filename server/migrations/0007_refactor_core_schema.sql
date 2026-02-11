PRAGMA foreign_keys = ON;

-- Canonical invite table. Replaces the older invites/invitations split and
-- enforces one invite relationship per unordered user pair.
CREATE TABLE IF NOT EXISTS contact_invites (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    requester_id INTEGER NOT NULL,
    recipient_id INTEGER NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'accepted', 'rejected', 'cancelled')),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (requester_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (recipient_id) REFERENCES users(id) ON DELETE CASCADE,
    CHECK (requester_id <> recipient_id)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_contact_invites_pair_unique
ON contact_invites (
    CASE WHEN requester_id < recipient_id THEN requester_id ELSE recipient_id END,
    CASE WHEN requester_id < recipient_id THEN recipient_id ELSE requester_id END
);

CREATE INDEX IF NOT EXISTS idx_contact_invites_recipient_status
ON contact_invites (recipient_id, status);

-- Migrate data from legacy invites table if present.
INSERT OR IGNORE INTO contact_invites (requester_id, recipient_id, status, created_at, updated_at)
SELECT inviter_id, invitee_id, status, created_at, created_at
FROM invites;

-- Wallet redesign: integer cents + transfer ledger.
CREATE TABLE IF NOT EXISTS wallet_accounts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL UNIQUE,
    balance_cents INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CHECK (balance_cents >= 0)
);

CREATE INDEX IF NOT EXISTS idx_wallet_accounts_user_id
ON wallet_accounts (user_id);

-- Backfill from legacy wallets.balance (REAL dollars) where available.
INSERT OR IGNORE INTO wallet_accounts (user_id, balance_cents, created_at, updated_at)
SELECT user_id, CAST(ROUND(balance * 100.0) AS INTEGER), created_at, updated_at
FROM wallets;

CREATE TABLE IF NOT EXISTS wallet_transfers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    sender_user_id INTEGER NOT NULL,
    recipient_user_id INTEGER NOT NULL,
    amount_cents INTEGER NOT NULL CHECK (amount_cents > 0),
    note TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (sender_user_id) REFERENCES users(id) ON DELETE RESTRICT,
    FOREIGN KEY (recipient_user_id) REFERENCES users(id) ON DELETE RESTRICT,
    CHECK (sender_user_id <> recipient_user_id)
);

CREATE INDEX IF NOT EXISTS idx_wallet_transfers_sender_created
ON wallet_transfers (sender_user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_wallet_transfers_recipient_created
ON wallet_transfers (recipient_user_id, created_at DESC);

-- Message delivery tracking to support richer client semantics.
CREATE TABLE IF NOT EXISTS message_deliveries (
    message_id INTEGER PRIMARY KEY,
    delivered_at DATETIME,
    read_at DATETIME,
    FOREIGN KEY (message_id) REFERENCES messages(id) ON DELETE CASCADE,
    CHECK (read_at IS NULL OR delivered_at IS NOT NULL)
);

-- Contacts should be unique and directional; also block self-contact.
CREATE UNIQUE INDEX IF NOT EXISTS idx_contacts_unique_pair
ON contacts (user_id, contact_id);
