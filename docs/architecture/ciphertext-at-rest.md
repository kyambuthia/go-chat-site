# Ciphertext-At-Rest Migration Plan

## Purpose
This document defines how the repo should move from plaintext durable message storage to ciphertext-at-rest without breaking the current delivery, sync, and receipt model.

It complements:
- `docs/architecture/decisions/e2ee-1to1-protocol.md`
- `docs/architecture/api.md`
- `docs/architecture/schema.md`

## Current State
Today the server persists plaintext message bodies for the current centralized relay model.

What already exists:
- durable message ids, inbox/outbox history, thread summaries, and sync cursors
- explicit delivery/read state
- device identity registration, prekeys, and public device directory APIs
- additive ciphertext/device-routing fields on durable message rows
- browser-side private device bundle generation and storage
- encrypted envelope generation on send plus local decrypt-on-read for matching devices
- explicit `content_kind` metadata for summary/unread logic
- sender-side local content cache so ciphertext-only outbox rows can still render after reload

What does not exist yet:
- production-default ciphertext-only storage rollout
- client-side ratchet/session state beyond the current bootstrap encrypted-envelope path
- a completed migration/retention decision for legacy plaintext rows

## Target State
For encrypted 1:1 conversations, the durable store should persist opaque ciphertext envelopes plus the minimum metadata needed for:
- routing
- sync pagination
- delivery/read receipts
- optimistic-message reconciliation

The server should no longer require plaintext bodies to support normal message delivery or thread-state reconciliation.

## Metadata That Can Remain Server-Visible
The following fields remain acceptable metadata in the target design:
- durable message id
- sender user id
- recipient user id
- sender device id
- recipient device id
- created_at / delivered_at / read_at
- client message id
- protocol version / message kind
- explicit content kind / control-message classification metadata
- coarse payload length or encoding metadata when strictly needed for transport validation

The following should not remain in cleartext for encrypted threads:
- user-authored body text
- payment request note bodies or any future freeform message content
- ratchet/session secrets

## Schema Direction
Prefer additive migration over destructive rewrite.

Recommended direction:
1. Add encrypted-envelope support without removing the current plaintext field immediately.
2. Introduce explicit ciphertext fields or a dedicated payload table for encrypted messages.
3. Keep current receipt/thread/sync tables usable with metadata-only changes.
4. Remove plaintext storage only after encrypted clients and legacy-read behavior are proven.

Two acceptable implementation shapes:

### Option A: Additive Columns on `messages`
- keep the existing message row
- add fields such as `ciphertext`, `encryption_version`, `sender_device_id`, `recipient_device_id`, and `encrypted_at`
- keep plaintext body only for legacy rows during rollout

### Option B: Separate Payload Table
- keep message metadata in `messages`
- move opaque encrypted payloads into `message_payloads`
- use a foreign key from payload rows to the durable message id

This repo should prefer the option that keeps sync/query behavior simplest while avoiding mixed semantics in application code.

## Rollout Stages
### Stage 0: Documentation and Contract Lock-In
- adopt the 1:1 E2EE protocol ADR
- document the encrypted envelope boundary
- confirm which message metadata stays server-visible

### Stage 1: Additive Schema Changes
- add ciphertext storage fields/tables
- add sender/recipient device metadata
- keep existing plaintext path intact

Current repo status:
- done, including `ciphertext`, `encryption_version`, `sender_device_id`, `recipient_device_id`, and `content_kind`

### Stage 2: Encrypted Write Path Behind a Flag
- clients with device keys can submit encrypted envelopes
- the server stores ciphertext but still serves legacy plaintext rows unchanged
- receipt and sync behavior must remain identical from the client's perspective

Current repo status:
- done for the initial rollout path
- `MESSAGING_STORE_PLAINTEXT_WHEN_ENCRYPTED` controls whether encrypted rows still dual-write plaintext `body`

### Stage 3: Read Path Preference for Ciphertext
- encrypted threads use ciphertext envelopes exclusively
- summary/unread logic must rely on message metadata, not plaintext inspection
- control-message filtering should use explicit metadata such as `content_kind`, not body parsing
- any message-type branching that still requires plaintext must be removed or redesigned

Current repo status:
- partly done
- recipient read path already prefers local decryption over plaintext fallback
- undecrypted incoming ciphertext no longer trusts compatibility plaintext for bubbles/previews
- thread summaries and unread filtering already use `content_kind` metadata
- remaining work is mainly around rollout posture and removing the last compatibility assumptions from sender history/UX

### Stage 4: Legacy Plaintext Retirement
- stop writing plaintext for E2EE-capable threads
- backfill or grandfather legacy plaintext rows according to an explicit retention policy
- remove transitional dual-write behavior after client/server compatibility windows close

## Compatibility Rules
- old plaintext messages must remain readable during migration
- new encrypted messages must not break existing delivery/read receipt flows
- thread summaries and unread counts must derive from metadata, not message body parsing
- server-side search over encrypted content is out of scope and should not silently degrade into false expectations

## Risks
- mixed plaintext/ciphertext storage can create confusing edge cases if the read path is not explicit
- any thread-summary logic that depends on message body content will become brittle
- moderation/compliance assumptions must change once message bodies are opaque
- multi-device fan-out increases the chance of duplicate or partially encrypted durable records if the envelope model is vague

## Required Checks Before Rollout
- migration is additive and reversible
- sync endpoints paginate encrypted rows correctly
- receipt updates remain stable when message content is opaque
- revoked devices cannot continue publishing usable encrypted envelopes
- thread previews for encrypted conversations do not rely on plaintext body access
- senders can still reconstruct their own ciphertext-only history after reload

## Out Of Scope For This Plan
- legal/compliance policy for encrypted-content retention
- private-key recovery UX
- attachment encryption
- group-message encryption rollout
