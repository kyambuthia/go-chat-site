# Database Schema

## Users

The `users` table stores user account information.

- `id`: Primary key.
- `username`: Unique username for each user.
- `password_hash`: Hashed password.
- `created_at`: Timestamp of account creation.

```sql
CREATE TABLE users (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  username TEXT UNIQUE NOT NULL,
  password_hash TEXT NOT NULL,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

## Messages

The `messages` table stores chat messages between users.

- `id`: Primary key.
- `from_user_id`: Foreign key to the `users` table, indicating the sender.
- `to_user_id`: Foreign key to the `users` table, indicating the recipient.
- `body`: The content of the message.
- `created_at`: Timestamp of when the message was sent.

```sql
CREATE TABLE messages (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  from_user_id INTEGER NOT NULL,
  to_user_id INTEGER NOT NULL,
  body TEXT NOT NULL,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY(from_user_id) REFERENCES users(id),
  FOREIGN KEY(to_user_id) REFERENCES users(id)
);
```
