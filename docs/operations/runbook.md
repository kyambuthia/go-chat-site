# Operator Runbook

## Health Checks
- Liveness: `GET /healthz`
- Readiness: `GET /readyz`

Examples:

```bash
curl -sS http://localhost:8080/healthz
curl -sS http://localhost:8080/readyz
```

## Common Incidents

### 1) `/readyz` returns 503
Likely causes:
- DB unavailable
- migrations table missing/corrupt

Actions:
1. Check server logs for `startup readiness check failed` or `not_ready` errors.
2. Validate DB file exists and is readable.
3. Run DB integrity check:

```bash
sqlite3 chat.db "PRAGMA integrity_check;"
```

4. If needed, restore from backup (see `docs/operations/backup-restore.md`).

### 2) Login requests return rate-limit errors
Likely cause:
- brute-force traffic or overly strict limits.

Actions:
1. Inspect logs by `path=/api/login` and status `429`.
2. Tune env vars:
- `LOGIN_RATE_LIMIT_PER_MINUTE`
- `WS_HANDSHAKE_RATE_LIMIT_PER_MINUTE`

3. Restart service with adjusted values.

### 3) WebSocket handshake failures
Likely causes:
- invalid/expired JWT
- rejected `Origin`
- upstream proxy stripping headers

Actions:
1. Confirm JWT validity (`/api/login` then reconnect).
2. Confirm `WS_ALLOWED_ORIGINS` includes client origin.
3. Verify proxy forwards `Upgrade` and `Connection` headers.

## Log Format
HTTP requests are logged in structured JSON lines with keys:
- `event`
- `request_id`
- `method`
- `path`
- `status`
- `duration_ms`
