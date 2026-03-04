# ADR: Retire `server/internal/ws` Compatibility Shim

## Status
Accepted (March 4, 2026)

## Context
The codebase migrated WebSocket transport behavior to `server/internal/adapters/transport/wsrelay`.
A compatibility shim package (`server/internal/ws`) remained temporarily and duplicated adapter types/functions.

## Decision
Remove `server/internal/ws` and consume `wsrelay` directly from runtime wiring (`cmd/main.go`, `internal/api`).

## Consequences
- Less duplicated code and fewer indirection layers.
- Coverage and test focus move to active adapter implementation.
- Future transport work (fallback/P2P) proceeds only through adapter boundaries.
