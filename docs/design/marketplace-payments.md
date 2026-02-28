# Marketplace + Payments Direction (Credits First, Rails Later)

## Product Direction
The project aims to support person-to-person and marketplace transactions, but the safest incremental path is:
1. internal ledger semantics (current wallet behavior)
2. escrow workflows
3. external rail adapters

This keeps early development focused on core business workflows while deferring jurisdiction-specific compliance and rail complexity.

## Terminology
- **Ledger**: internal accounting model for balances and transfers
- **Escrow**: funds hold/release/dispute workflow linked to an order
- **Payment rail adapter**: integration that executes funding/settlement outside the core ledger model

## Recommended MVP Scope
### In Scope
- internal ledger transfers (already present behavior, renamed semantically in core)
- escrow hold/release/dispute states
- order-linked payment/escrow correlation IDs
- operator-visible audit trail

### Out of Scope (for MVP)
- fiat custody/integration
- crypto rails and key custody
- automated KYC/AML workflows
- cross-border settlement optimization

## Core Design Principle
Separate:
- business intent (`Order`, `Escrow`, `LedgerTransfer`)
from:
- rail execution (`PaymentAttempt`, `Settlement`, provider-specific IDs)

This prevents external provider assumptions from leaking into core domain logic.

## Risk and Compliance Assumptions (Baseline Decisions)
- Platform model: custodial for in-app credits.
- Balances: redeemable for fiat only through operator-controlled payout workflows.
- Initial jurisdiction scope: US-only beta rollout.
- KYC/AML/sanctions checks: required before fiat funding or payout.
- Disputes/refunds: operator-mediated, with auditable event history.

## Engineering Rules for Payment-Related PRs
- Transactions must be atomic or explicitly compensating.
- Every state transition must be auditable.
- Idempotency keys required for external rail calls.
- No direct provider SDK usage in core packages.
- Failure-path tests required (timeouts, partial success, retries).
