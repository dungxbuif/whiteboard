---
artifact_type: test_verification
id: TEST-000
status: draft
owner: ai
human_fields:
  - uat_sign_off
  - manual_acceptance_notes
ai_fields:
  - commands
  - automated_tests
  - manual_checks
  - failure_summary
  - evidence_notes
  - attempt_log
shared_fields:
  - status
  - trace
  - uat
trace:
  backlog_item: TBD
  requirement: TBD
  phase: TBD
  ticket_or_bug: TBD
  detail_design: TBD
  validation_matrix: TBD
  docs_review: TBD
  release_notes: TBD
---

# Test Verification Template

Use this template to record execution-based verification.

## Copy And Fill Rules

- AI owns command evidence and attempt logs.
- Human owns UAT sign-off when required.
- Replace `TEST-000` with a stable verification ID.
- Do not paste excessive logs. Summarize relevant output.
- If tests are skipped, record the reason and residual risk.
- If loop guard triggers, mark the source ticket/bug `blocked`.

## Field Ownership

- Human fills UAT sign-off and manual acceptance notes when required.
- AI fills commands, results, automated/manual verification, failure summary, and attempt log.
- Shared fields include status, trace links, and UAT requirement.

## Status

- ID: TEST-000
- Status: draft | ready | in_progress | blocked | in_review | verified | done
- Owner: TBD

## Scope

- Ticket/Bug/Phase: TBD
- Tested behavior: TBD

## Trace Links

- Backlog item: TBD
- Requirement: TBD
- Phase: TBD
- Ticket/Bug: TBD
- Detail design: TBD
- Validation matrix: TBD
- Docs review: TBD
- Release notes: TBD

## Commands

| Command | Result | Notes |
| --- | --- | --- |
| `npm test` | pass \| fail \| skipped | Summary and relevant failure, not full log |

## Fix/Test Attempt Log

Use this section when verification fails and fixes are attempted.

| Attempt | Change Made | Command | Result | Failure Summary |
| --- | --- | --- | --- | --- |
| 1 | TBD | TBD | TBD | TBD |

Loop guard:

- Same-path failure attempts: TBD / 3
- Total fix/test cycles: TBD / 5
- Blocked: yes | no
- Human/design input needed: TBD

## Automated Tests

- Passed: TBD
- Failed: TBD
- Skipped: TBD

## Manual Checks

- TBD

## UAT

- Required: yes | no
- Reason if not required: TBD
- Expected behavior: TBD
- Verified behavior: TBD
- Sign-off: TBD

## Failures And Follow-Up

- TBD

## Evidence Notes

Summarize terminal output or browser evidence. Do not paste excessive logs.
