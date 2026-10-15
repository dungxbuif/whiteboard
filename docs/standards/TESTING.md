# Testing Standards

## Test Evidence

Agents must record real command results in the active ticket, bug, detail design, or test verification artifact.

Record:

- Command
- Result: pass, fail, skipped
- Summary of failures
- Manual checks, if any
- Reason if a test could not be run

## Fix/Test Loop Guard

Tests are allowed to fail during development, but repeated failure must become evidence, not an infinite loop.

Agents MUST stop when:

- The same test path fails after 3 fix attempts.
- The task reaches 5 total fix/test cycles.
- A passing fix would require unapproved scope expansion.

After stopping, mark the active work item `blocked` and record the attempt log.

## Required Coverage

- Feature work: cover new behavior and acceptance criteria.
- Bug work: cover reproduction or regression path.
- API work: cover contract success and failure cases.
- Data work: cover migration or schema-sensitive behavior when applicable.
- User-facing work: record UAT or a reason UAT is not required.

## Skipping Tests

Tests may be skipped only when:

- The task is docs-only, or
- The environment cannot run the test command, and the reason is recorded.

Skipping tests does not remove the reconciliation requirement.

## UAT

For user-facing work, record:

- Acceptance criteria checked
- Expected behavior
- Verified behavior
- Manual/browser/API steps, when applicable
- Human sign-off, or `not required` with reason

Use `docs/templates/UAT.md` when this does not fit cleanly in the ticket.
