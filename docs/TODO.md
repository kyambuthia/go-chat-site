# TODO

This file tracks the remaining feature work in the repository by roadmap phase.

It is intentionally scoped to product and platform feature delivery, not general repo hygiene. For engineering/process work, see `docs/roadmaps/repo-improvements.md`.

## Phase 1: Finish The Current Product Surface

### Status
- complete

### Remaining
- none

## Phase 2: Offline Messaging And Sync

### Status
- complete

### Remaining
- none

## Phase 3: Session Hardening And Account Security

### Status
- complete

### Remaining
- none

## Phase 4: Device Identity And E2EE Foundations

### Status
- in progress

### Remaining
- tighten the broader device identity / E2EE UX so enrollment, recovery/import-export expectations, and multi-device behavior are clear end to end
- add real private-key recovery or import-export flows for enrolled devices instead of only surfacing local-key availability state
- validate the final ciphertext-only rollout path in a controlled environment with plaintext suppression enabled by default
- reduce the remaining sender-side compatibility reliance on locally cached plaintext in long-tail encrypted history edge cases
- move beyond the current bootstrap encrypted-envelope path toward the full X3DH + Double Ratchet target described in the ADR
- define and implement ratchet/session state handling for durable encrypted conversations

## Cross-Cutting Messaging Gaps

### Status
- in progress

### Remaining
- broader multi-device presence semantics
- define how multi-device delivery/read state should behave across concurrent sessions
- final controlled rollout validation for encrypted messaging with plaintext suppression enabled

## Phase 5: P2P Messaging Transport

### Status
- not started

### Remaining
- signaling/session APIs
- signaling adapters for transport negotiation
- transport negotiation model
- feature-flagged experimental browser-first P2P adapter
- automatic relay fallback when P2P setup fails
- fallback telemetry and operator visibility
- define failure modes and fallback thresholds
- track relay-versus-P2P usage and success rates
- reconnect, renegotiation, and downgrade-path test coverage

## Phase 6: Marketplace Domain

### Status
- not started

### Remaining
- marketplace persistence
- marketplace APIs
- listings UI
- listing browsing UI
- offer flow
- order detail flow
- order event log
- dispute workflow scaffolding
- moderation state machine scaffolding
- `marketplace_listings` data model
- `marketplace_offers` data model
- `marketplace_orders` data model
- `marketplace_order_events` data model
- `marketplace_disputes` data model
- linking marketplace orders to messaging threads where appropriate
- audit logging for order state changes

## Phase 7: Escrow And Auditable Ledger Events

### Status
- not started

### Remaining
- ledger event model
- escrow hold flow
- escrow release flow
- escrow dispute flow
- order-to-escrow-to-ledger correlation IDs
- operator-visible audit trail
- reconciliation surfaces
- show escrow status in order detail
- expose dispute status and operator actions
- `ledger_events` data model
- `escrow_holds` data model
- `escrow_events` data model
- `payment_instructions` data model
- `payment_settlements` data model
- `payment_rail_attempts` data model

## Phase 8: External Payment Rail And Compliance Tooling

### Status
- not started

### Remaining
- payment adapter interface
- payment adapter interfaces for external rails
- first external rail integration
- idempotent external transfer execution
- operator tooling for funding, payout, settlement failures, and disputes
- reconciliation jobs
- compliance architecture decision for the first rollout jurisdiction
- KYC / AML / sanctions workflow integration before fiat funding or payout
- operator-controlled payout workflows for custodial balances

## Marketplace + Payments Product Scope Still Out Of MVP

### Status
- deferred

### Remaining
- fiat custody / direct external balance integration
- crypto rail support
- crypto key custody
- automated KYC / AML workflows
- cross-border settlement optimization

## Platform Beta Exit Gaps

### Status
- not yet satisfied

### Remaining
- relay fallback works automatically when P2P fails
- ledger and escrow events are auditable end to end
- abuse controls and observability are sufficient for controlled rollout
- compliance posture is explicitly documented for the chosen rollout jurisdiction
