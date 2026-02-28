# ADR: Core / Adapter Boundaries for Messaging + Marketplace + Payments

## Status
Accepted (initial refactor boundary PR)

## Context
The current backend is a centralized demo application with:
- HTTP API handlers coupled to persistence store interfaces
- a centralized Gorilla WebSocket relay hub
- wallet endpoints backed by SQLite balance updates

The next product direction is a platform that can evolve toward:
- peer-to-peer messaging transports (while keeping a centralized relay adapter as fallback)
- marketplace workflows (listing, offers, orders)
- escrow and payments integrations (real rails later)

We need boundaries now so new capabilities can be added as adapters without rewriting the existing client/API behavior.

## Decision
Introduce `server/internal/core/*` domain packages (types + interfaces only in this PR) and move infrastructure code behind adapters:
- `core/identity`: identity/auth abstractions (today username/password + JWT; later DID/WebAuthn)
- `core/messaging`: message/thread/receipt types and transport + key provider seams
- `core/contacts`: contact graph and invite domain types
- `core/ledger`: replaces "wallet" naming in domain model while preserving current behavior
- `core/marketplace`: minimal stubs for future listing/offer/order/escrow flows
- `adapters/transport/wsrelay`: current centralized websocket relay implementation
- `adapters/httpapi`: current mux HTTP handlers/routing (route paths preserved)

Compatibility shims may remain temporarily (`internal/api`, `internal/ws`) to reduce risk and keep imports stable while refactoring incrementally.

## Consequences
Positive:
- Messaging transport can later gain a P2P adapter without changing API/domain contracts.
- JWT auth can later be replaced/augmented by DID/WebAuthn adapters.
- Current wallet behavior can be evolved into a ledger + payment-rail adapter model with explicit events.
- Existing client routes and websocket protocol remain stable during migration.

Tradeoffs:
- Some temporary duplication/shims exist during the transition.
- Domain interfaces are intentionally conservative and may change after more product decisions are finalized.

## Explicitly Decided Baselines
- Jurisdiction and regulatory posture: US-only beta, compliance-first expansion.
- KYC/AML and identity proofing: required before fiat rails; provider selected during rail integration phase.
- Custody model: custodial internal ledger for MVP.
- Escrow and disputes: operator-mediated dispute workflow with immutable audit events.
- Currency support: USD-only for MVP; no FX in initial rollout.
- Encryption model: centralized relay now, X3DH + Double Ratchet migration path for 1:1 E2EE.
- Key policy: per-device key material with rotation and revocation primitives in Phase 3.
- Fraud/moderation/sanctions: blocklist + anomaly rules in MVP, dedicated providers post-MVP.
- Data retention: 30-day operational logs, longer retention only for auditable ledger events.

## Migration Notes for This PR
- No real P2P networking yet
- No real payment rails yet
- No marketplace business logic yet
- Existing `/api/*` routes and `/ws` endpoint remain client-compatible

## Environment Variables Added (Security Scaffolding)
- `WS_ALLOWED_ORIGINS` (comma-separated origins) for WebSocket origin allowlist. If unset, development-safe localhost defaults are allowed and requests with no `Origin` header are accepted for local tooling/non-browser clients.
- `LOGIN_RATE_LIMIT_PER_MINUTE` (optional, default `60`)
- `WS_HANDSHAKE_RATE_LIMIT_PER_MINUTE` (optional, default `120`)
