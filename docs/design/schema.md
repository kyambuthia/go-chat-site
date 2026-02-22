# Data Model Direction (SQLite Today, Adapter-Friendly Tomorrow)

## Purpose
This document describes:
- current SQLite-backed entities that power the app today
- the target domain model evolution for messaging, ledger, escrow, and marketplace
- boundaries that allow SQLite to remain the first adapter while future stores/rails are added

## Current Operational Model (Implemented)
### Identity / Users
- `users`
  - username/password hash
  - optional profile fields (`display_name`, `avatar_url`)

### Contacts
- `contacts`
  - directional edges
  - accepted invite creates reciprocal rows

### Invites
- `contact_invites`
  - pending/accepted/rejected invite workflow

### Wallet (Compatibility Name)
- `wallet_accounts`
  - integer `balance_cents`
- `wallet_transfers`
  - transfer records for auditability

## Domain Renaming Direction: Wallet -> Ledger
Implementation may keep current table names initially, but domain semantics should move to:
- `ledger_accounts`
- `ledger_transfers`
- `ledger_events`

Why:
- "wallet" implies product UX only
- "ledger" better supports escrow, reversals, settlements, and external rails

## Target Messaging Data Model (Planned)
### Core Messaging
- `message_threads`
- `messages`
- `message_receipts`
- `message_sync_cursors`

### E2EE / Device-Key Support (if adopted)
- `device_identities`
- `device_prekeys`
- `device_sessions` (optional server metadata only)

Notes:
- ciphertext storage is expected if E2EE is implemented
- server should avoid storing plaintext content long-term if product direction requires privacy-first posture (`TODO: VERIFY`)

## Target Marketplace Data Model (Planned)
- `marketplace_listings`
- `marketplace_offers`
- `marketplace_orders`
- `marketplace_order_events`
- `marketplace_disputes`

Notes:
- keep order/event state transitions append-only where possible
- do not bake jurisdiction-specific compliance policy into schema too early (`TODO: VERIFY`)

## Target Escrow / Payment Adapter Data Model (Planned)
- `escrow_holds`
- `escrow_events`
- `payment_instructions`
- `payment_settlements`
- `payment_rail_attempts`

Notes:
- separate business intent (escrow/order) from rail execution (payment adapters)
- preserve correlation IDs for audits and reconciliation

## Migration Strategy
1. Keep existing SQLite tables and behavior stable.
2. Add service/domain abstractions first.
3. Add new tables in additive migrations (no destructive rewrites).
4. Backfill or dual-write only after behavior is covered by tests.
5. Remove compatibility names only after client/server migration is complete.

## Invariants (Must Hold)
- Monetary values stored in integer cents (or smallest unit per currency) for now.
- Transfers/ledger movements must remain transactional.
- Message delivery/receipt semantics must be explicit (no silent partial success).
- State transitions must be auditable (prefer append-only events for critical domains).
