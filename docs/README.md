# Project Docs

This repository is evolving from a centralized chat + wallet demo into a security-first platform for:
- peer-to-peer messaging (with centralized relay fallback)
- marketplace transactions (listings, offers, orders, disputes)
- payments and escrow (starting with internal ledger semantics)

## Directory Layout
- `docs/overview/`: high-level project intent and product direction
- `docs/architecture/`: current system contracts, runtime/data model docs, and ADRs
- `docs/roadmaps/`: phased delivery and engineering execution plans
- `docs/product/`: domain strategy documents for marketplace and payments
- `docs/operations/`: runbooks and operator-facing procedures

## Read This First
- `docs/overview/course-correction.md`: course-correction brief and the top-level product direction
- `docs/architecture/decisions/architecture-boundaries.md`: adopted core vs adapter boundary decision
- `docs/roadmaps/feature-implementation.md`: concrete feature roadmap from the current repo state

## Architecture
- `docs/architecture/api.md`: current API compatibility and future versioning direction
- `docs/architecture/schema.md`: current SQLite schema concepts and target data model evolution
- `docs/architecture/runtime.md`: runtime flows (HTTP, WS relay today; P2P later)
- `docs/architecture/security.md`: security scaffolding, threat model, and hardening plan
- `docs/architecture/decisions/architecture-boundaries.md`: architectural boundary decision record
- `docs/architecture/decisions/ws-shim-retirement.md`: decision record for removing the websocket compatibility shim

## Roadmaps
- `docs/roadmaps/platform-roadmap.md`: phased migration plan toward P2P + marketplace + payments
- `docs/roadmaps/feature-implementation.md`: feature delivery order and exit criteria
- `docs/roadmaps/repo-improvements.md`: phased engineering roadmap for repo quality, CI, and maintainability

## Product
- `docs/product/marketplace-payments.md`: ledger, escrow, and payment-rail strategy

## Operations
- `docs/operations/runbook.md`: operator troubleshooting and recovery steps
- `docs/operations/backup-restore.md`: SQLite backup and restore workflow
- `docs/operations/load-testing.md`: lightweight k6 load test scenarios for critical paths

## Documentation Rules
- Keep current behavior vs target behavior explicit.
- Mark undecided/high-risk areas as `TODO: VERIFY`.
- Prefer small ADRs when changing boundaries or trust assumptions.
- Update tests/docs together when route or protocol behavior changes.
