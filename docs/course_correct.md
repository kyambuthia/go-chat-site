# Course Correction: From Chat Demo to Messaging + Marketplace + Payments Platform

## Why This Exists
The original project is a centralized chat application with a simple internal wallet demo. That foundation is useful, but the intended product direction is larger:
- private messaging between users (eventually P2P-capable)
- commerce workflows (listing, offer, order, dispute)
- payment and escrow orchestration (real rails later)

This document replaces a broad research dump with a concise engineering direction for this repository.

## Current Baseline (What We Have Today)
Backend (`server/`) currently provides:
- JWT auth (`JWT_SECRET`) and mux-based HTTP APIs
- SQLite (`chat.db`) with migrations on startup
- centralized WebSocket relay at `/ws`
- contact graph + invites
- wallet-like balance and transfer behavior

Client (`client/`) currently depends on:
- `/api/*` endpoints
- `/ws` with bearer token passed via WebSocket subprotocol `bearer.<token>`

## Target Direction (What We Are Building Toward)
### 1. Messaging
- Today: centralized WS relay for online direct messaging
- Next: pluggable transport layer with relay + signaling responsibilities isolated
- Later: P2P transport adapter(s) (WebRTC first for browsers), relay fallback retained

### 2. Identity
- Today: username/password + JWT session token
- Next: isolate identity abstractions behind core interfaces
- Later: WebAuthn/passkeys and device keys (DID-compatible if adopted)

### 3. Ledger / Payments
- Today: internal wallet balances and transfers in SQLite
- Next: rename semantics at the domain layer from "wallet" to "ledger"
- Later: escrow state machine, external rail adapters, compliance controls

### 4. Marketplace
- Today: no marketplace domain logic
- Next: define domain vocabulary and interfaces only (Listing, Offer, Order, Escrow)
- Later: implement policies, dispute resolution, fulfillment workflows

## Architectural Rule
Use **core domain interfaces + adapters**.
- Core packages define types, invariants, and interfaces.
- Adapters implement transport, HTTP, persistence, and external integrations.
- Compatibility shims are acceptable during migration if they preserve client behavior.

See `docs/adr/architecture-boundaries.md`.

## Non-Negotiable Engineering Approach
- TDD first for behavior changes
- Small reversible commits
- Explicit assumptions
- No guessing: encode behavior in tests or mark `TODO: VERIFY`

## What Is Deliberately Not Decided Yet (TODO: VERIFY)
- Jurisdiction and regulatory scope
- KYC/AML posture and provider choices
- Custodial vs non-custodial money model
- Payment rails (fiat, crypto, hybrid)
- E2EE protocol choice and key recovery UX
- Marketplace moderation and dispute arbitration policy

## Immediate Priorities (Next PRs)
1. Add explicit messaging service interfaces and tests around WS relay adapter behavior.
2. Add ledger service layer (preserve current wallet routes as compatibility aliases).
3. Add escrow domain model + storage tables (no real rail integration yet).
4. Add signaling/session APIs for future WebRTC P2P experiments.
5. Expand threat model tests (rate limits, auth failures, replay/abuse scenarios).
