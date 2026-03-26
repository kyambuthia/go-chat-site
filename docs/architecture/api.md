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
- `POST /api/auth/refresh`
- `POST /api/logout`
- `GET /api/sessions`
- `DELETE /api/sessions`
- `GET /api/me`
- `PATCH /api/me`

Device identity:
- `GET /api/devices`
- `POST /api/devices`
- `DELETE /api/devices`
- `POST /api/devices/rotate`
- `GET /api/devices/directory`
- `POST /api/messaging/prekeys`

Contacts and invites:
- `GET /api/contacts`
- `GET /api/invites`
- `POST /api/invites/send`
- `POST /api/invites/accept`
- `POST /api/invites/reject`

Wallet compatibility routes (current behavior retained):
- `GET /api/wallet`
- `GET /api/wallet/transfers`
- `POST /api/wallet/send`

Messaging sync:
- `GET /api/messaging/threads`
- `GET /api/messaging/sync`
- `POST /api/messaging/read-thread`

Realtime:
- `GET /ws` (WebSocket upgrade)

Operational:
- `GET /healthz`
- `GET /readyz`

## Current WebSocket Protocol (Compatibility Contract)
Supported message types:
- `direct_message`
- `message_ack`
- `presence_state`
- `user_online`
- `user_offline`
- `error` (server-generated for invalid recipient/offline recipient)

Auth transport:
- `Authorization: Bearer <token>` header, or
- WebSocket subprotocol `bearer.<token>`

Current delivery behavior:
- new connections receive `presence_state` with the currently online usernames
- `direct_message` to online user is forwarded with durable `id` when available
- `direct_message` payloads may also carry `content_kind`, `ciphertext`, `encryption_version`, `sender_device_id`, and `recipient_device_id`
- sender receives `message_ack` on successful relay; the ack echoes the client `id` and may include `stored_message_id`
- if recipient offline, sender receives `error` and **no ack**; offline-send errors may include `stored_message_id` when the message was persisted

Current sync payload notes:
- `GET /api/messaging/sync` accepts optional `after_id` and `limit`
- sync responses return `cursor.after_id`, `cursor.next_after_id`, `messages`, and `has_more`
- clients should treat `cursor.next_after_id` as the next durable checkpoint after each page
- `GET /api/messages/outbox` and `GET /api/messaging/sync` may include `client_message_id` on sent messages so the client can reconcile optimistic local bubbles with durable stored messages after reconnect
- sent messages may include `delivery_failed: true` when the original real-time send failed because the recipient was offline; if `delivered_at` is still missing and `delivery_failed` is absent, clients should treat the message as pending rather than failed
- stored message payloads may also include optional E2EE envelope metadata: `ciphertext`, `encryption_version`, `sender_device_id`, and `recipient_device_id`
- stored message payloads and thread summaries may also include `content_kind` so clients can distinguish user-visible messages from control updates without inspecting plaintext bodies
- encrypted-capable senders may persist a local client-side content cache keyed by `client_message_id` / `stored_message_id` so ciphertext-only outbox rows still render after reload
- `GET /api/messaging/threads` derives `unread_count` and `last_message` from user-visible thread activity; control-style microapp updates such as `payment_request_update` still appear in thread history/sync payloads, but they do not increment unread counts or replace thread-list previews
- `POST /api/messaging/read-thread` accepts `{ "with_user_id": <id> }` and marks all unread incoming messages in that one 1:1 conversation as delivered/read so thread-level unread state survives reloads and reconnects

## Current Auth Session Contract
Login and refresh responses return:
- `token`: compatibility alias for `access_token`
- `access_token`
- `refresh_token`
- `access_token_expires_at`
- `refresh_token_expires_at`
- `session`

Current `session` payload fields:
- `id`
- `device_label`
- `user_agent`
- `last_seen_ip`
- `created_at`
- `last_seen_at`
- `access_token_expires_at`
- `refresh_token_expires_at`

Session management behavior:
- `POST /api/auth/refresh` accepts `{ "refresh_token": "...", "device_label": "..." }`
- throttled `POST /api/login` and `POST /api/auth/refresh` responses return HTTP `429`, `Retry-After`, and JSON `retry_after_seconds`; login lockouts additionally include `locked_until`
- `POST /api/logout` revokes the current authenticated session
- `GET /api/sessions` returns the caller's active sessions and marks the current one with `current: true`
- `DELETE /api/sessions` accepts `{ "session_id": <id> }` and revokes that session if it belongs to the caller

## Auth and Security Notes
- JWT secret is configured by `JWT_SECRET`
- Access tokens are validated against session state when a `session_id` claim is present
- Refresh tokens are opaque, rotated on refresh, and replay-protected
- Login uses per-IP and per-user quotas plus a lockout/backoff table
- Refresh uses per-IP quotas to bound refresh-token abuse and replay probing
- WS origin checks use `WS_ALLOWED_ORIGINS` with localhost-safe defaults
- encrypted-message plaintext dual-write can be controlled with `MESSAGING_STORE_PLAINTEXT_WHEN_ENCRYPTED` and is disabled by default

## Current Device Identity Contract
- `POST /api/devices` registers a device identity with `label`, `algorithm`, `identity_key`, `signed_prekey_id`, `signed_prekey`, `signed_prekey_signature`, and `prekeys`
- `GET /api/devices` returns the caller's registered devices with `state`, `prekey_count`, and `current_session`
- `POST /api/messaging/prekeys` accepts `{ "device_id": <id>, "prekeys": [{ "prekey_id": <id>, "public_key": "..." }] }`
- `DELETE /api/devices` accepts `{ "device_id": <id> }` and revokes the matching device identity plus its active prekeys
- `GET /api/devices/directory?username=<name>` resolves active device bundles for that user

## Encrypted Message Envelope Direction
- durable message records now reserve optional fields for `ciphertext`, `encryption_version`, `sender_device_id`, and `recipient_device_id`
- encrypted rows may also carry `content_kind` metadata so unread counts and thread previews do not depend on plaintext parsing
- plaintext `body` remains supported during migration, but encrypted rows are now stored without plaintext by default; set `MESSAGING_STORE_PLAINTEXT_WHEN_ENCRYPTED=true` only for temporary compatibility rollback
- ciphertext-only durable rows should still preserve sender UX via local client-side sent-content caching and recipient UX via local decryption
- clients and adapters should treat `ciphertext` as opaque payload data and should not require server-side plaintext inspection for receipts or pagination

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
