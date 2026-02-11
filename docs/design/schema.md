# Database Schema (Refactored)

## Core Entities

### `users`

Stores account identity and profile data.

```sql
CREATE TABLE users (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  username TEXT UNIQUE NOT NULL,
  password_hash TEXT NOT NULL,
  display_name TEXT,
  avatar_url TEXT,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### `contacts`

Directional contact edges. Accepted invites create two rows for bidirectional contact.

```sql
CREATE TABLE contacts (
  user_id INTEGER NOT NULL,
  contact_id INTEGER NOT NULL,
  PRIMARY KEY (user_id, contact_id),
  FOREIGN KEY(user_id) REFERENCES users(id),
  FOREIGN KEY(contact_id) REFERENCES users(id)
);
```

## Invites

### `contact_invites`

Canonical invite table with one relationship per unordered user pair.

```sql
CREATE TABLE contact_invites (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  requester_id INTEGER NOT NULL,
  recipient_id INTEGER NOT NULL,
  status TEXT NOT NULL DEFAULT 'pending',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CHECK (requester_id <> recipient_id),
  FOREIGN KEY (requester_id) REFERENCES users(id) ON DELETE CASCADE,
  FOREIGN KEY (recipient_id) REFERENCES users(id) ON DELETE CASCADE
);
```

## Messaging

### `messages`

Stores direct message payloads.

### `message_deliveries`

Tracks delivery/read timestamps per message.

## Wallet

### `wallet_accounts`

Balances in integer cents.

```sql
CREATE TABLE wallet_accounts (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL UNIQUE,
  balance_cents INTEGER NOT NULL DEFAULT 0,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CHECK (balance_cents >= 0),
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
```

### `wallet_transfers`

Transfer ledger for auditability.

```sql
CREATE TABLE wallet_transfers (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  sender_user_id INTEGER NOT NULL,
  recipient_user_id INTEGER NOT NULL,
  amount_cents INTEGER NOT NULL CHECK (amount_cents > 0),
  note TEXT NOT NULL DEFAULT '',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CHECK (sender_user_id <> recipient_user_id),
  FOREIGN KEY (sender_user_id) REFERENCES users(id) ON DELETE RESTRICT,
  FOREIGN KEY (recipient_user_id) REFERENCES users(id) ON DELETE RESTRICT
);
```
