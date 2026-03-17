# Repository Improvement Roadmap (Engineering + Delivery)

## Snapshot (updated March 4, 2026)
- Backend tests are wired into CI with coverage reporting and threshold enforcement.
- Frontend quality gates include lint, typecheck, tests, and build in CI.
- CI runs on both `pull_request` and `push` to `main`.
- Security automation is enabled (`govulncheck`, `npm audit`, `gitleaks`).
- Compatibility/legacy packages removed (`server/internal/ws`, `server/internal/handlers`).
- Operational readiness endpoints are available (`/healthz`, `/readyz`) with DB/migration checks.

## Roadmap Outcomes
- Reduce regression risk for API, messaging, and ledger behavior.
- Make local development and CI behavior consistent and repeatable.
- Improve maintainability by completing architectural boundary cleanup.
- Add operational and security guardrails before higher-risk feature rollout.

## Phase 1 (Completed): Developer Workflow Baseline
Goals:
- make local setup and validation predictable
- align tooling choices

Deliverables:
- choose one JS package manager for the repo and remove mixed lockfile drift
- add root-level task runner commands (for example: `test`, `test-go`, `test-client`, `lint`, `build`)
- add `.env.example` files for server and client with documented required variables
- update `README.md` with a single "quick start + verify" flow

Exit Criteria:
- new contributors can run setup + all checks with one documented command sequence
- local check commands and CI commands are aligned

## Phase 2 (Completed): CI and Quality Gates
Goals:
- catch breakage before merge
- increase confidence in high-change areas

Deliverables:
- run CI on both `pull_request` and `push`
- add backend linting (`golangci-lint` or equivalent) and formatting checks
- add frontend lint/typecheck gate
- publish backend coverage report artifact in CI
- raise backend coverage from `50.5%` to at least `65%`, prioritizing:
  - `server/internal/auth`
  - `server/internal/store`
  - `server/internal/adapters/httpapi/router.go` paths
- expand frontend tests beyond runtime config (App/chat/contact flows)

Exit Criteria:
- pull requests are blocked on failed lint/test/build checks
- coverage trend is visible in CI and minimum threshold is enforced

## Phase 3 (Completed): Architecture Cleanup
Goals:
- finish migration boundaries
- remove unused/deprecated execution paths

Deliverables:
- remove or fully isolate deprecated `server/internal/handlers` package
- decide and document end-state for `server/internal/ws` compatibility shim
- continue splitting legacy store concerns in `server/internal/store/sqlite.go` toward adapter-specific packages
- add/expand route contract tests to prove no API regressions during cleanup

Exit Criteria:
- deprecated packages are either removed or clearly isolated from runtime wiring
- compatibility behavior is covered by tests for all currently supported `/api/*` and `/ws` contracts

## Phase 4 (Completed): Operational Readiness
Goals:
- make runtime behavior observable and supportable
- reduce deploy-time surprises

Deliverables:
- structured logging format with request IDs preserved
- health endpoints (`/healthz`, `/readyz`) and startup checks (DB + migration status)
- documented backup/restore workflow for SQLite data
- lightweight load test scenarios for login, websocket handshake, and message relay
- operator runbook in docs (common failure modes + recovery steps)

Exit Criteria:
- service health and critical paths can be validated quickly in staging
- on-call/operator diagnostics do not require code-level debugging for common issues

## Phase 5 (Completed): Security and Release Governance
Goals:
- add baseline secure-development automation
- make releases auditable and repeatable

Deliverables:
- dependency automation (Dependabot or Renovate)
- security scanning in CI (`govulncheck` for Go, JS dependency audit, secret scan)
- `SECURITY.md` and `CONTRIBUTING.md` with reporting and development expectations
- automated release process (tagging/changelog/build artifacts)

Exit Criteria:
- every release has traceable checks, notes, and reproducible build steps
- security checks run automatically on pull requests and default branch

## Suggested Execution Order (First 10 Tickets)
1. CI: add `pull_request` trigger and status checks.
2. Tooling: pick one client package manager and remove lockfile duplication.
3. Docs: add `.env.example` files and update quick start.
4. Backend lint gate + formatter check.
5. Frontend lint/typecheck gate.
6. Coverage job + threshold enforcement.
7. Backend tests for `auth` middleware/login edge cases.
8. Frontend component tests for login/chat/contact happy paths.
9. Deprecation cleanup plan PR for `server/internal/handlers` and `server/internal/ws`.
10. Add `SECURITY.md` and security scan workflow.
