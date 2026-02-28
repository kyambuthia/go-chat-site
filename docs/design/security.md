# Security Direction and Threat Model (Initial)

## Security Posture for Current Phase
This project is transitioning from a demo toward a platform architecture. Security work in this phase focuses on **scaffolding and boundaries** rather than full production hardening.

Implemented in current codebase:
- Request ID correlation in API logs (`X-Request-ID` propagation/generation)
- WS origin allowlist (env-driven, localhost-safe defaults)
- Lightweight rate limiting on `/api/login` and `/ws` handshake
- JWT-based authenticated API and WS sessions

## Critical Risks (Current + Near-Term)
### 1. Account Abuse / Brute Force
- Risk: repeated login attempts, WS handshake floods, resource exhaustion
- Current mitigation: lightweight per-IP fixed-window rate limits
- Next steps:
  - per-user + per-IP quotas (rolling window backed by shared store)
  - structured security event logging
  - lockout/backoff policy: 5 failed logins in 15 minutes triggers a 15-minute cool-down

### 2. Session Token Misuse
- Risk: stolen JWT used for API and WS impersonation
- Current mitigation: bearer token validation only
- Next steps:
  - refresh tokens/device sessions
  - token rotation and revocation list
  - device-bound sessions: planned for post-MVP, after refresh-token rollout

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
- `WS_HANDSHAKE_RATE_LIMIT_PER_MINUTE` (optional; default `120`)

## Logging Requirements (Going Forward)
- Include request correlation IDs in API logs.
- Avoid logging secrets, credentials, raw JWTs, or private key material.
- Add event categories for auth failures, rate-limit hits, and critical ledger operations.
- Retention and access policy: application logs retained 30 days by default; access restricted to operators.

## Security Design Rules for Future PRs
- New external integrations must enter through adapters.
- High-risk changes require tests for failure/abuse cases, not just success cases.
- Document trust boundaries and attack assumptions in ADRs.
- Prefer reversible migrations for auth, ledger, and messaging protocols.
