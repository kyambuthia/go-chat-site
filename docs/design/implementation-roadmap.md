# Feature Implementation Roadmap

## Purpose
This document turns the existing product/design direction into a concrete implementation roadmap for the remaining feature work in this repository.

It complements:
- `docs/design/roadmap.md`: platform migration strategy
- `docs/design/security.md`: security hardening direction
- `docs/design/marketplace-payments.md`: ledger, escrow, and rail assumptions

## Current Baseline
Implemented today:
- username/password registration and login
- JWT-authenticated HTTP and WebSocket sessions
- contacts and invite workflows
- centralized 1:1 WebSocket relay
- durable message persistence on the backend, plus inbox/outbox/read/delivered APIs
- wallet balance lookup and direct user-to-user transfer
- read-only account/profile fetch
- health/readiness checks, request logging, and basic rate limiting

Not complete end-to-end today:
- missed-message recovery in the client
- message delivery/read/error UX in the client
- contact add/remove UX
- profile editing
- wallet transfer UX and transfer history
- marketplace, escrow, external rails, E2EE, and P2P transport

## Delivery Order
1. Finish the currently shipped chat, contacts, wallet, and profile surface.
2. Complete offline messaging and sync semantics.
3. Harden sessions and account security.
4. Add device identity and E2EE foundations.
5. Add experimental P2P transport with relay fallback.
6. Implement marketplace workflows.
7. Implement escrow and auditable ledger events.
8. Integrate the first external payment rail and operator/compliance tooling.

## Phase 1: Finish The Current Product Surface
Outcome:
- the current app works after refresh/reconnect
- all currently exposed backend capabilities are actually reachable in the UI

Deliverables:
- hydrate chat threads from existing message APIs instead of keeping message state only in React memory
- handle WebSocket `error`, `message_delivered`, and `message_read` events in the client
- expose add/remove contacts in routed API and UI
- keep invite creation available even when contacts already exist
- add profile update capability for `display_name` and `avatar_url`
- render real avatar URLs where present
- wire the existing send-money UI into the app
- add wallet transfer history API and UI
- define initial presence semantics for reconnect and multi-session use

Suggested backend work:
- mount `POST /api/contacts` and `DELETE /api/contacts`
- add account/profile update route
- add wallet transfer history route
- define and implement an initial online-roster or presence-bootstrap API/message

Suggested frontend work:
- add `getInbox`, `getOutbox`, `markDelivered`, and `markRead` client helpers
- load thread history when opening chat and on app startup
- merge persisted message history with optimistic WebSocket state
- surface offline-send failures and receipt state in chat bubbles
- add contacts management controls
- add account edit form and transfer history view

Exit criteria:
- reloading the app preserves visible conversations
- users can see send failures, delivered state, and read state
- users can add/remove contacts without relying on empty-state flows
- users can edit profile fields and see avatar URLs rendered
- users can send money and review recent transfers in the UI

## Phase 2: Offline Messaging And Sync
Outcome:
- messaging works for online and offline users without relying on WebSocket-only state

Deliverables:
- `GET /api/messaging/sync` or equivalent cursor-based sync endpoint
- explicit sync cursor model
- thread summaries and unread counts derived from durable state
- consistent delivered/read transitions across HTTP and WebSocket flows
- reconnect reconciliation between client-local optimistic messages and stored messages

Data and API work:
- add `message_threads`
- add `message_receipts` if `message_deliveries` is no longer sufficient
- add `message_sync_cursors`
- formalize sync payloads and cursor semantics in `docs/design/api.md`

Client work:
- run sync on login, reconnect, and app resume
- backfill missed direct messages and receipt events
- clear unread counts from durable state, not only local state

Exit criteria:
- a user who was offline can reopen the app and see missed messages
- receipt state survives reloads and reconnects
- unread counts are derived from persisted state and remain accurate

## Phase 3: Session Hardening And Account Security
Outcome:
- auth/session behavior is suitable for a controlled beta instead of a demo-only posture

Deliverables:
- refresh tokens and device sessions
- token rotation and revocation
- per-user and per-IP auth quotas
- login lockout/backoff policy
- structured auth and rate-limit event logging

Supporting work:
- define token/session lifecycle in docs
- add revocation storage and cleanup strategy
- add tests for stolen-token and replay-oriented failure paths

Exit criteria:
- API and WebSocket access can be revoked without rotating `JWT_SECRET`
- sessions can be listed and invalidated per device
- abusive login patterns trigger enforced backoff

## Phase 4: Device Identity And E2EE Foundations
Outcome:
- the system can support encrypted messaging without redesigning core contracts later

Deliverables:
- device identity registration APIs
- key directory model
- key rotation and revocation semantics
- protocol ADR for 1:1 E2EE
- server-side ciphertext-at-rest migration plan

Data work:
- add `device_identities`
- add `device_prekeys`
- add optional `device_sessions` metadata

Client work:
- manage device enrollment and key publication
- separate plaintext composer state from transport payload generation

Exit criteria:
- a user can register at least one device identity
- key material can be rotated and revoked without breaking account identity
- the selected encryption protocol is documented and test-covered at the integration boundary

## Phase 5: P2P Messaging Transport
Outcome:
- the app can attempt browser-first peer transport while keeping the current relay as fallback

Deliverables:
- signaling/session APIs
- transport negotiation model
- experimental WebRTC-style adapter behind a feature flag
- fallback telemetry and operator visibility

Supporting work:
- define failure modes and fallback thresholds
- track relay-vs-P2P usage and success rates
- test reconnect, renegotiation, and transport downgrade paths

Exit criteria:
- two supported clients can negotiate P2P transport experimentally
- relay fallback works automatically when P2P setup fails
- telemetry shows transport choice and failure reasons

## Phase 6: Marketplace Domain
Outcome:
- listings, offers, and orders exist as real product features instead of domain stubs

Deliverables:
- marketplace persistence
- marketplace APIs
- order event log
- dispute and moderation state machine scaffolding
- marketplace UI for listing creation, browsing, offer flow, and order detail

Data work:
- add `marketplace_listings`
- add `marketplace_offers`
- add `marketplace_orders`
- add `marketplace_order_events`
- add `marketplace_disputes`

Integration work:
- link orders to messaging threads where appropriate
- add audit logging for order state changes

Exit criteria:
- sellers can create listings
- buyers can submit offers or create orders
- order state changes are persisted and auditable

## Phase 7: Escrow And Auditable Ledger Events
Outcome:
- funds movement supports marketplace flows and operator-managed disputes

Deliverables:
- ledger event model
- escrow hold, release, and dispute flows
- order-to-escrow-to-ledger correlation IDs
- operator-visible audit trail and reconciliation surfaces

Data work:
- add `ledger_events`
- add `escrow_holds`
- add `escrow_events`
- add `payment_instructions`
- add `payment_settlements`
- add `payment_rail_attempts`

Product work:
- show escrow status in order detail
- expose dispute status and operator actions

Exit criteria:
- escrow state transitions are append-only or otherwise fully auditable
- every funds movement is traceable to order and operator context
- dispute flows are persisted and reviewable

## Phase 8: External Payment Rails And Compliance Tooling
Outcome:
- the platform can move beyond internal credits for a controlled US beta

Deliverables:
- payment adapter interface
- first rail integration
- idempotent external transfer execution
- reconciliation jobs
- operator tooling for payouts, failures, and disputes
- compliance architecture decision for KYC, AML, and sanctions checks

Supporting work:
- provider selection ADR
- payout/funding lifecycle tests
- timeout, retry, and partial-success handling

Exit criteria:
- at least one external rail works through adapters, not core packages
- reconciliation catches missing or mismatched settlement outcomes
- operator workflows exist for payout failures, disputes, and manual review

## Cross-Cutting Work Required In Every Phase
- update docs and ADRs when contracts, trust boundaries, or failure semantics change
- add route, domain, and integration tests before behavior changes
- keep migrations additive and reversible where possible
- extend observability for new critical paths
- run load and abuse tests for auth, messaging, and money movement changes

## Suggested First 15 Tickets
1. Mount `POST /api/contacts` and `DELETE /api/contacts`.
2. Add client API helpers for inbox/outbox/read/delivered message routes.
3. Hydrate chat threads from persisted messages.
4. Surface WebSocket error, delivered, and read events in the chat UI.
5. Keep invite creation visible outside the empty-contacts state.
6. Add profile update route and account edit UI.
7. Render `avatar_url` anywhere avatar data exists.
8. Wire `SendMoneyForm` into the account or chat flow.
9. Add wallet transfer history route and recent-transfers UI.
10. Define initial online-roster bootstrap behavior.
11. Add reconnect reconciliation between optimistic message IDs and stored message IDs.
12. Add cursor-based messaging sync endpoint.
13. Add refresh token and device-session storage model.
14. Add auth event logging plus stronger lockout/backoff rules.
15. Write the E2EE protocol ADR and device identity API contract.

## Explicitly Deferred Until After The Above
- crypto payment rails
- cross-border settlement optimization
- non-USD currency support
- automated compliance expansion beyond the initial US-only beta scope
