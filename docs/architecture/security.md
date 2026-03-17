# Security Direction and Threat Model (Initial)

## Security Posture for Current Phase
This project is transitioning from a demo toward a platform architecture. Security work in this phase focuses on **scaffolding and boundaries** rather than full production hardening.

Implemented in current codebase:
- Request ID correlation in API logs (`X-Request-ID` propagation/generation)
- Structured JSON HTTP request logs with request IDs
- WS origin allowlist (env-driven, localhost-safe defaults)
- Per-IP and per-user login quotas backed by shared SQLite windows
- Login lockout/backoff after repeated password failures
- Structured auth/rate-limit event logging for login, refresh, revocation, and quota violations
- Session-backed JWT access tokens plus rotating refresh tokens
- Per-device session listing and revocation
- JWT-based authenticated API and WS sessions with explicit session revocation support

## Critical Risks (Current + Near-Term)
### 1. Account Abuse / Brute Force
- Risk: repeated login attempts, WS handshake floods, resource exhaustion
- Current mitigation:
  - per-IP and per-user login quotas
  - structured auth event logging
  - lockout/backoff policy: 5 failed logins in 15 minutes triggers a 15-minute cool-down
- Remaining next steps:
  - operator-visible auth event dashboards/alerting
  - CAPTCHA or other challenge flow if the beta threat model requires it

### 2. Session Token Misuse
- Risk: stolen JWT used for API and WS impersonation
- Current mitigation:
  - short-lived access tokens
  - rotating refresh tokens
  - session-backed JWT validation
  - per-session logout/revoke endpoints
  - replay detection on refresh token reuse, which revokes the session
- Remaining next steps:
  - device-bound session proofs: planned for post-MVP, after refresh-token rollout
  - operator tooling for suspicious-session review

### 3. WebSocket Cross-Origin Abuse
- Risk: browser-based malicious sites connecting to local/dev WS endpoint
- Current mitigation: origin allowlist via `WS_ALLOWED_ORIGINS`
- Notes:
  - requests without `Origin` are currently allowed for non-browser tooling/dev use
  - revisit stricter policy before production deployment

### 4. Messaging Privacy
- Risk: server-visible plaintext messages in centralized relay architecture
- Current mitigation: transport auth only (no E2EE)
- Next steps:
  - add device key infrastructure
  - protocol direction selected: X3DH + Double Ratchet for 1:1 messaging; MLS deferred for group messaging

### 5. Payment / Ledger Fraud and Compliance Exposure
- Risk: fund theft, fake disputes, sanctions/KYC violations once real rails exist
- Current mitigation: none beyond basic transfer validation + DB transactionality
- Next steps:
  - escrow workflow with explicit states and audit events
  - reconciliation and anomaly detection
  - compliance architecture decision (jurisdiction/KYC/AML)

## Environment Variables (Security-Relevant)
- `JWT_SECRET` (required)
- `WS_ALLOWED_ORIGINS` (optional; comma-separated)
- `LOGIN_RATE_LIMIT_PER_MINUTE` (optional; default `60`)
- `LOGIN_USER_RATE_LIMIT_PER_MINUTE` (optional; default `20`)
- `WS_HANDSHAKE_RATE_LIMIT_PER_MINUTE` (optional; default `120`)
- `ACCESS_TOKEN_TTL_MINUTES` (optional; default `15`)
- `REFRESH_TOKEN_TTL_HOURS` (optional; default `720`)
- `LOGIN_LOCKOUT_THRESHOLD` (optional; default `5`)
- `LOGIN_LOCKOUT_WINDOW_MINUTES` (optional; default `15`)
- `LOGIN_LOCKOUT_DURATION_MINUTES` (optional; default `15`)

## Session Lifecycle
- `POST /api/login` issues a session-backed access token and a refresh token.
- Access tokens are bearer JWTs intended for short-lived API and WebSocket authentication.
- Refresh tokens are opaque secrets stored only as hashes in `auth_sessions`.
- `POST /api/auth/refresh` rotates both the access token and the refresh token for the same session.
- Reusing a previous refresh token is treated as replay and revokes the affected session.
- `POST /api/logout` revokes the current session and disconnects matching WebSocket clients.
- `GET /api/sessions` and `DELETE /api/sessions` provide per-device session inventory and invalidation.
- Expired/revoked sessions are cleaned up opportunistically on auth hot paths; there is not yet a dedicated background janitor.

## Logging Requirements (Going Forward)
- Include request correlation IDs in API logs.
- Avoid logging secrets, credentials, raw JWTs, or private key material.
- Auth event categories currently include login success/failure, refresh success/failure, session revocation, rate-limit hits, and lockout events.
- Add event categories for critical ledger operations as those paths land.
- Retention and access policy: application logs retained 30 days by default; access restricted to operators.

## Security Design Rules for Future PRs
- New external integrations must enter through adapters.
- High-risk changes require tests for failure/abuse cases, not just success cases.
- Document trust boundaries and attack assumptions in ADRs.
- Prefer reversible migrations for auth, ledger, and messaging protocols.

## CI Security Automation
- `govulncheck` for Go code/dependencies
- `npm audit` for client dependencies
- `gitleaks` for secret scanning
