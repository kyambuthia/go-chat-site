# API Direction and Compatibility

## Scope
This document defines:
- the **current stable API contract** used by the client
- compatibility expectations during refactors
- the **future API shape** for messaging, marketplace, and ledger/escrow features

## Current Stable Routes (Must Preserve During Refactor)
Authentication and profile:
- `POST /api/register`
- `POST /api/login`
- `GET /api/me`

Contacts and invites:
- `GET /api/contacts`
- `GET /api/invites`
- `POST /api/invites/send`
- `POST /api/invites/accept`
- `POST /api/invites/reject`

Wallet compatibility routes (current behavior retained):
- `GET /api/wallet`
- `POST /api/wallet/send`

Realtime:
- `GET /ws` (WebSocket upgrade)

## Current WebSocket Protocol (Compatibility Contract)
Supported message types:
- `direct_message`
- `message_ack`
- `user_online`
- `user_offline`
- `error` (server-generated for invalid recipient/offline recipient)

Auth transport:
- `Authorization: Bearer <token>` header, or
- WebSocket subprotocol `bearer.<token>`

Current delivery behavior:
- `direct_message` to online user is forwarded
- sender receives `message_ack` on successful relay
- if recipient offline, sender receives `error` and **no ack**

## Auth and Security Notes
- JWT secret is configured by `JWT_SECRET`
- Login and WS handshake are rate-limited (lightweight per-IP fixed window)
- WS origin checks use `WS_ALLOWED_ORIGINS` with localhost-safe defaults

## Future Versioning Strategy
Short-term:
- keep `/api/*` routes stable for existing client
- add new endpoints under `/api/...` only when necessary to avoid breaking frontend

Medium-term:
- introduce `/api/v1/*` for expanded messaging/ledger/marketplace surface
- keep old routes as compatibility shims until client migration completes

## Planned API Additions (Not Implemented Yet)
### Messaging / Signaling
- `POST /api/messaging/sessions` (issue signaling/session descriptors)
- `POST /api/messaging/prekeys` (if E2EE protocol requires server-hosted prekeys)
- `GET /api/messaging/sync` (offline event/message sync)

### Ledger / Escrow
- `GET /api/ledger/account`
- `POST /api/ledger/transfers`
- `POST /api/escrow/holds`
- `POST /api/escrow/{id}/release`
- `POST /api/escrow/{id}/dispute`

### Marketplace
- `POST /api/marketplace/listings`
- `GET /api/marketplace/listings`
- `POST /api/marketplace/offers`
- `POST /api/marketplace/orders`

## Change Management Rules
- Any behavioral change to existing routes requires tests first.
- If renaming concepts (e.g., wallet -> ledger), preserve compatibility routes until frontend is updated.
- Document all new error semantics and message types before rollout.
