---
artifact_type: docs_review
id: DOCS-REVIEW-000
status: draft
owner: ai
human_fields:
  - reviewer_override
  - approval
ai_fields:
  - review_checklist
  - findings
  - result
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
  adrs: []
  release_notes: TBD
---

# Docs Review Template

Use this template when a ticket, bug, or detail design needs a standalone docs review artifact.

## Copy And Fill Rules

- AI performs docs review after implementation and verification.
- Replace `DOCS-REVIEW-000` with a stable review ID.
- Every changed contract must map to an updated doc or a clear not-needed reason.
- If docs are stale, result is `fail` or `blocked`, not `pass`.

## Field Ownership

- Human may override approval or request reviewer follow-up.
- AI fills checklist, findings, result, and trace links.

## Status

- ID: DOCS-REVIEW-000
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
- ADRs: TBD
- Release notes: TBD

## Review Checklist

- [ ] Code changed but docs unchanged: reason recorded
- [ ] User-facing behavior changed: requirements updated or not needed reason recorded
- [ ] API/contract changed: `docs/architecture/API.md` updated or not needed reason recorded
- [ ] Data model changed: `docs/architecture/ERD.md` updated or not needed reason recorded
- [ ] Architecture/runtime changed: `docs/architecture/ARCHITECTURE.md` updated or not needed reason recorded
- [ ] Durable decision changed: ADR added or not needed reason recorded
- [ ] `docs/CONTEXT.md` updated
- [ ] Trace links updated

## Findings

- TBD

## Result

- Result: pass | fail | blocked
- Notes: TBD
