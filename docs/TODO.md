# TODO

This file tracks the major product features that are still incomplete in the repository.

It is intentionally scoped to feature work, not general repo hygiene.

## Phase 4: Device Identity And E2EE Foundations

### Remaining
- tighten the broader device identity / E2EE user experience so enrollment, recovery expectations, and multi-device behavior are clear end to end
- move beyond the current bootstrap encrypted-envelope path toward the full X3DH + Double Ratchet target described in the ADR

## Cross-Cutting Messaging Gaps

### Remaining
- broader multi-device presence semantics
- final controlled rollout validation for encrypted messaging with plaintext suppression enabled

## Phase 5: P2P Messaging Transport

### Remaining
- signaling/session APIs
- transport negotiation model
- experimental browser-first P2P adapter behind a feature flag
- automatic relay fallback when P2P setup fails
- fallback telemetry and operator visibility
- reconnect, renegotiation, and downgrade-path test coverage

## Phase 6: Marketplace Domain

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
- linking marketplace orders to messaging threads where appropriate
- audit logging for order state changes

## Phase 7: Escrow And Auditable Ledger Events

### Remaining
- ledger event model
- escrow hold flow
- escrow release flow
- escrow dispute flow
- order-to-escrow-to-ledger correlation IDs
- operator-visible audit trail
- reconciliation surfaces
- `ledger_events` data model
- `escrow_holds` data model
- `escrow_events` data model
- `payment_instructions` data model
- `payment_settlements` data model
- `payment_rail_attempts` data model

## Phase 8: External Payment Rail And Compliance Tooling

### Remaining
- payment adapter interfaces for external rails
- first external rail integration
- operator tooling for funding, payout, and settlement workflows
- reconciliation jobs
- compliance architecture decision for the first rollout jurisdiction
- KYC / AML / sanctions workflow integration before fiat funding or payout
- operator-controlled payout workflows for custodial balances

## Marketplace + Payments Product Scope Still Out Of MVP

### Remaining
- fiat custody / direct external balance integration
- crypto rail support
- crypto key custody
- automated KYC / AML workflows
- cross-border settlement optimization
