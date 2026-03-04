# SQLite Backup and Restore

## Scope
This runbook covers backup/restore of the repository's SQLite database (`chat.db`).

## Backup
1. Stop write traffic to the server (recommended), or stop the server entirely.
2. Run:

```bash
sqlite3 chat.db ".backup '/tmp/chat-$(date +%Y%m%d-%H%M%S).db'"
```

3. Verify backup integrity:

```bash
sqlite3 /tmp/chat-YYYYMMDD-HHMMSS.db "PRAGMA integrity_check;"
```

Expected output: `ok`.

## Restore
1. Stop the server.
2. Copy backup into place:

```bash
cp /tmp/chat-YYYYMMDD-HHMMSS.db chat.db
```

3. Start server and verify readiness:

```bash
curl -sS http://localhost:8080/readyz
```

Expected payload includes `"status":"ready"`.

## Retention
- Keep at least 7 daily backups for local/dev environments.
- For staging/prod, keep policy in deployment infra and encrypt backups at rest.
