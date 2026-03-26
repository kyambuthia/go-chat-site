# ADR: 1:1 E2EE Protocol Baseline

## Status
Accepted for Phase 4 implementation planning

## Context
The current product still uses a centralized relay and durable server-side message storage. Phase 4 adds device identity registration, signed prekeys, one-time prekeys, and a device directory so the system can evolve toward end-to-end encryption without rewriting the core messaging contracts later.

We need a concrete 1:1 protocol direction now so:
- device identity APIs are shaped around a real protocol instead of placeholders
- ciphertext-at-rest migration can target a stable envelope format
- future client work can generate and rotate key material without reopening the server contract

Constraints in this repo:
- 1:1 chat is the only in-scope encrypted messaging target today
- the server remains the delivery relay and durable store
- the current client/server contract already assumes per-user conversations and explicit receipts
- group messaging is not implemented and should not block 1:1 E2EE progress

## Decision
Adopt the following baseline for 1:1 encrypted messaging:

- Initial session establishment: `X3DH`-style authenticated key agreement using per-device identity keys, signed prekeys, and one-time prekeys.
- Ongoing message encryption: `Double Ratchet` per sender-device / recipient-device session.
- Group messaging: `MLS` is explicitly deferred until there is a real group product surface.

## Why This Protocol Family
`X3DH + Double Ratchet` is the best fit for the current product shape because:
- it matches the repo's per-device identity and prekey directory model
- it works well for asynchronous 1:1 messaging where recipients may be offline
- it lets the server relay opaque encrypted envelopes without participating in key agreement
- it is well understood and easier to reason about for the current single-conversation model than introducing MLS too early

## Server Responsibilities
The server remains responsible for:
- authenticating users and their device-management actions
- storing public device identity bundles and active prekeys
- removing revoked devices and revoked prekeys from the public directory
- relaying and durably storing opaque ciphertext envelopes
- preserving delivery, read, and thread-summary semantics without parsing message plaintext

The server must not:
- generate long-term client private keys
- receive or store client private identity keys
- inspect ciphertext payload contents to derive application behavior

## Device Model
The protocol boundary is per device, not only per user.

Implications:
- each user may publish multiple active device identities
- a single user-to-user send may fan out into multiple encrypted device envelopes
- revoking one device must not break the user's other active devices
- prekeys are consumed at the device level, not the account level

## Envelope Contract Direction
The stable server-facing encrypted message contract should evolve toward an opaque envelope with metadata only for delivery and reconciliation. The exact route shape can evolve, but the boundary should preserve:

- sender user id
- sender device id
- recipient user id
- recipient device id
- client message id / reconciliation id
- ciphertext payload
- protocol version
- created-at / durable message id

The server should treat ciphertext as an opaque byte/string payload and should not branch behavior on decrypted message content.

## Consequences
Positive:
- fits the current device directory and prekey storage model
- keeps the relay/server architecture compatible with an eventual encrypted transport
- lets the current receipt and durable sync model survive with mostly metadata-only changes

Tradeoffs:
- multi-device fan-out increases message send complexity
- server-side moderation/search on plaintext becomes unavailable for encrypted threads
- key backup, recovery, and new-device history bootstrap remain product problems that are intentionally deferred

## Explicitly Deferred
- MLS or any group-encryption protocol
- cross-device private-key backup and recovery UX
- encrypted attachment transport
- post-compromise recovery UX beyond the baseline Double Ratchet model
- deniability guarantees beyond what the chosen primitive set already provides

## Required Integration Tests Before E2EE Rollout
- device directory excludes revoked devices and revoked prekeys
- device rotation leaves unaffected devices usable
- a sender can resolve active recipient device bundles without reading private key material
- encrypted message persistence treats ciphertext as opaque payload data
- delivery/read receipts still reconcile correctly when message bodies are not server-readable

## Follow-On Work
- formalize the encrypted message envelope in the HTTP/WebSocket contracts
- add client-side key generation and ratchet state management
- add ciphertext-at-rest schema changes and a staged rollout plan
