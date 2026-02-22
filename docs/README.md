# Project Docs

This repository is evolving from a centralized chat + wallet demo into a security-first platform for:
- peer-to-peer messaging (with centralized relay fallback)
- marketplace transactions (listings, offers, orders, disputes)
- payments and escrow (starting with internal ledger semantics)

## Read This First
- `docs/course_correct.md`: course-correction brief and product/architecture direction
- `docs/adr/architecture-boundaries.md`: adopted core vs adapter boundary decision

## Design Docs
- `docs/design/api.md`: current API compatibility and future versioning direction
- `docs/design/schema.md`: current SQLite schema concepts and target data model evolution
- `docs/design/runtime.md`: runtime flows (HTTP, WS relay today; P2P later)
- `docs/design/security.md`: security scaffolding, threat model, and hardening plan
- `docs/design/roadmap.md`: phased migration plan toward P2P + marketplace + payments
- `docs/design/marketplace-payments.md`: ledger/escrow/payment-rail strategy and assumptions

## Documentation Rules (for future PRs)
- Keep current behavior vs target behavior explicit.
- Mark undecided/high-risk areas as `TODO: VERIFY`.
- Prefer small ADRs when changing boundaries or trust assumptions.
- Update tests/docs together when route or protocol behavior changes.
