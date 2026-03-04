# Contributing

## Development Setup
1. Copy env files:

```bash
cp .env.example .env
cp client/.env.example client/.env
```

2. Install client dependencies:

```bash
make setup
```

3. Run full local validation before opening a PR:

```bash
make check
```

## Pull Request Requirements
- Keep changes small and focused.
- Add or update tests for behavior changes.
- Update docs when routes/protocol/runtime behavior changes.
- Ensure CI passes (lint, tests, coverage, build).

## Commit Message Guidance
Use imperative, scoped messages where possible.

Examples:
- `harden auth middleware error handling`
- `add readiness endpoints and db health checks`
- `expand store test coverage for contacts and invites`
