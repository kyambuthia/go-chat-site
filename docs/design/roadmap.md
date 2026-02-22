# Roadmap: Incremental Migration to P2P Messaging + Marketplace + Payments

## Phase 0 (Completed / In Progress): Boundary Refactor
Goals:
- establish core vs adapter boundaries
- preserve existing routes and WS behavior
- add security scaffolding

Status:
- core domain packages created (`identity`, `messaging`, `contacts`, `ledger`, `marketplace`)
- WS relay moved to adapter with compatibility shim
- HTTP handlers moved to adapter with compatibility shim
- request IDs, WS origin allowlist, and basic rate limiting added

## Phase 1: Core Services and Compatibility Layer Tightening
Goals:
- introduce explicit core service layer for messaging/ledger/contacts
- reduce handler-to-store coupling
- keep `/api/wallet*` as compatibility aliases while introducing ledger service terminology internally

Deliverables:
- `core/*` service interfaces and unit tests
- adapter implementations for SQLite store operations
- route tests that codify compatibility behavior

## Phase 2: Offline Messaging + Sync (Still Centralized Relay)
Goals:
- support durable messages and receipts for offline users
- preserve WS real-time relay for online users

Deliverables:
- message/thread/receipt persistence
- sync endpoint(s) and cursor model
- explicit delivery state transitions

## Phase 3: Device Identity + E2EE Foundations
Goals:
- add device identity registration and key directory model
- prepare for end-to-end encrypted messaging

Deliverables:
- device identity storage + APIs
- key rotation and revocation semantics
- protocol decision ADR (`TODO: VERIFY`: X3DH/Double Ratchet, MLS, or hybrid)

## Phase 4: P2P Messaging Transport (Experimental Adapter)
Goals:
- add browser-first P2P transport adapter (likely WebRTC signaling + fallback)
- retain WS relay as fallback path

Deliverables:
- signaling APIs/adapters
- transport negotiation model
- feature flag and fallback telemetry

## Phase 5: Marketplace Domain Logic
Goals:
- implement listings/offers/orders and order events
- add dispute workflow scaffolding

Deliverables:
- marketplace APIs + persistence
- moderation/dispute state machine
- audit/event logging

## Phase 6: Escrow and Payment Rail Adapters
Goals:
- add escrow holds/releases/disputes in ledger
- integrate external rails after compliance decisions

Deliverables:
- escrow event model
- payment adapter interfaces + first implementation (`TODO: VERIFY`)
- reconciliation jobs and operator tooling

## Exit Criteria for “Platform Beta” (Draft)
- messaging works online/offline with receipts
- relay fallback works when P2P fails
- ledger/escrow events are auditable
- abuse controls and observability are sufficient for controlled rollout
- compliance posture explicitly documented for chosen jurisdiction(s)
