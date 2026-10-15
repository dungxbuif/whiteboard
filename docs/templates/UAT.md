---
artifact_type: uat
id: UAT-000
status: draft
owner: human
human_fields:
  - sign_off
  - acceptance_notes
  - reviewer
ai_fields:
  - expected_behavior
  - verified_behavior
  - verification_steps
shared_fields:
  - status
  - trace
trace:
  backlog_item: TBD
  requirement: TBD
  phase: TBD
  ticket_or_bug: TBD
  detail_design: TBD
  test_verification: TBD
  validation_matrix: TBD
  release_notes: TBD
---

# UAT Template

Use this template for user-facing work that needs acceptance verification.

## Copy And Fill Rules

- Human owns sign-off.
- AI may prepare expected behavior, verification steps, and observed behavior.
- Replace `UAT-000` with a stable UAT ID.
- If UAT is not required, record the reason in both this artifact and the source ticket/bug.

## Field Ownership

- Human owns sign-off, acceptance notes, and reviewer identity.
- AI may prepare expected behavior, verified behavior, and verification steps.

## Status

- ID: UAT-000
- Status: draft | ready | in_progress | blocked | in_review | verified | done
- Reviewer: TBD

## Trace Links

- Backlog item: TBD
- Requirement: TBD
- Phase: TBD
- Ticket/Bug: TBD
- Detail design: TBD
- Test verification: TBD
- Validation matrix: TBD
- Release notes: TBD

## Acceptance Criteria Checked

- TBD

## Expected Behavior

TBD

## Verified Behavior

TBD

## Verification Steps

1. TBD

## Sign-Off

- Required: yes | no
- Sign-off: TBD
- Reason if not required: TBD
