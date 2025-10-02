Core Principles (non-negotiable)

Test-Driven Development (TDD) first.

Always write the tests that describe the behaviour before writing implementation code.

Tests are executable documentation: they must be readable, deterministic, and fast.

Do not hallucinate / do not assume.

If you are unsure about behaviour, model it with a failing test or mark it as TODO: VERIFY in the design doc. Never implement guessed behaviour without a test that codifies the expectation.

Keep assumptions explicit (small design doc, API contract, or ASSUMPTIONS.md entry).

Small batches.

Break every feature into tiny, independently testable tasks (see "Batching" section).

Each commit should be a single small, reversible change: either (A) add tests, or (B) implement to satisfy tests, not both in a large sweep.

Reason before action.

Before touching code, write a one-paragraph design + a test-plan with the key happy-path and failure cases. Then code the tests.

CI & lint gates.

PRs must run unit tests, linters, and integration tests where applicable. No PR without passing tests.

Project layout (recommended)
