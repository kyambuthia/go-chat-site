CREATE TABLE IF NOT EXISTS device_identities (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    label TEXT NOT NULL DEFAULT '',
    algorithm TEXT NOT NULL DEFAULT 'x3dh-ed25519-x25519-v1',
    identity_key TEXT NOT NULL,
    signed_prekey_id INTEGER NOT NULL,
    signed_prekey TEXT NOT NULL,
    signed_prekey_signature TEXT NOT NULL,
    key_state TEXT NOT NULL DEFAULT 'active' CHECK (key_state IN ('active', 'revoked')),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    published_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    rotated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    revoked_at DATETIME,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_device_identities_user_created
    ON device_identities (user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_device_identities_user_state
    ON device_identities (user_id, key_state);

CREATE TABLE IF NOT EXISTS device_prekeys (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    device_identity_id INTEGER NOT NULL REFERENCES device_identities(id) ON DELETE CASCADE,
    prekey_id INTEGER NOT NULL,
    public_key TEXT NOT NULL,
    key_state TEXT NOT NULL DEFAULT 'active' CHECK (key_state IN ('active', 'revoked')),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    revoked_at DATETIME,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(device_identity_id, prekey_id)
);

CREATE INDEX IF NOT EXISTS idx_device_prekeys_device_state
    ON device_prekeys (device_identity_id, key_state, prekey_id);

CREATE TABLE IF NOT EXISTS device_sessions (
    device_identity_id INTEGER NOT NULL REFERENCES device_identities(id) ON DELETE CASCADE,
    auth_session_id INTEGER NOT NULL REFERENCES auth_sessions(id) ON DELETE CASCADE,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_seen_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (device_identity_id, auth_session_id)
);

CREATE INDEX IF NOT EXISTS idx_device_sessions_auth_session
    ON device_sessions (auth_session_id);
