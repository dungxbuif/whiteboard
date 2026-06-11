---
artifact_type: bug
id: BUG-000
status: draft
owner: human
severity: TBD
priority: TBD
bug_markers: []
human_fields:
  - symptoms
  - expected_behavior
  - priority
  - severity
ai_fields:
  - reproduction
  - actual_behavior
  - root_cause
  - impact_scope
  - fix_strategy
  - regression_tests
  - verification_results
shared_fields:
  - status
  - trace
  - docs_review
trace:
  backlog_item: TBD
  requirement: TBD
  phase: TBD
  detail_design: TBD
  test_verification: TBD
  validation_matrix: TBD
  docs_review: TBD
  adrs: []
  release_notes: TBD
---

# Bug Template

Copy this template to `docs/work/bugs/BUG-<id>-<short-name>.md`.

Before filling this template, read `docs/standards/QUALITY_BAR.md#good-bug-report`.

## Copy And Fill Rules

- Human should fill symptoms, expected behavior, severity, and priority when known.
- AI must reproduce before proposing a fix.
- AI must write root cause before implementation.
- Follow `docs/standards/DEBUGGING.md`.
- Replace `BUG-000` with `BUG-<id>-<short-name>`.
- Do not mark `fixed`, `regression_verified`, `verified`, or `done` without evidence.

## Field Ownership

- Human fills symptom, expected behavior, severity, and priority when known.
- AI fills reproduction, actual behavior, root cause, impact scope, fix strategy, regression tests, and verification evidence.
- Shared fields include status, trace links, docs review, and release impact.

## Status

- ID: BUG-000
- Status: draft | ready | in_progress | blocked | in_review | verified | done
- Bug markers: reproduced | fixed | regression_verified | none
- Severity: low | medium | high | critical
- Priority: low | medium | high | urgent
- Phase: TBD
- Owner: TBD

## Trace Links

- Backlog item: TBD
- Requirement: TBD
- Phase: TBD
- Detail design: TBD
- Test verification: TBD
- Validation matrix: TBD
- Docs review: TBD
- ADRs: TBD
- Release notes: TBD

## Symptoms

Human fill:

- What was observed:
- Who/what is affected:
- When it started, if known:

## Reproduction

AI fill exact steps:

1. Setup/input:
2. Command/action:
3. Observed result:
4. Reproducibility: always | intermittent | unknown

For bug, failure, or unexpected behavior work, follow `docs/standards/DEBUGGING.md`.

## Expected Behavior

TBD

## Actual Behavior

TBD

## Root Cause

AI fill only after investigation:

- Root cause:
- Evidence:
- Why this is not only a symptom:

## Impact Scope

- Code: TBD
- Users: TBD
- Data: TBD
- API/contracts: TBD
- Related modules: TBD

## Fix Strategy

- Smallest root-cause fix:
- Regression protection:
- Risk:
- Scope boundaries:

## Regression Tests

- TBD

## Verification Results

- Command: TBD
- Result: TBD
- Notes: TBD

## Fix/Test Attempt Log

- Same-path failure attempts: TBD / 3
- Total fix/test cycles: TBD / 5
- Blocked by loop guard: yes | no
- Human/design input needed: TBD

## Docs Review

- Requirements updated or not needed reason: TBD
- Architecture updated or not needed reason: TBD
- API updated or not needed reason: TBD
- ERD/data updated or not needed reason: TBD
- ADR created or not needed reason: TBD
- `docs/CONTEXT.md` updated: TBD

## Completion Checklist

- [ ] Root cause identified
- [ ] Fix implemented
- [ ] Regression test added or updated
- [ ] Impacted behavior retested
- [ ] Fix/test loop guard respected
- [ ] Validation matrix updated or explicitly not affected
- [ ] Master docs reconciled
- [ ] Docs review completed
- [ ] `docs/CONTEXT.md` updated
- [ ] `docs/work/BACKLOG.md` updated
- [ ] Trace links updated
