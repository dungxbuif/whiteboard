---
artifact_type: validation_matrix
id: VALIDATION_MATRIX
status: active
owner: shared
human_fields:
  - proof_override
  - acceptance_signoff
ai_fields:
  - proof_recommendation
  - evidence_links
  - status_updates
shared_fields:
  - matrix_rows
  - validation_status
updated: YYYY-MM-DD
---

# Validation Matrix Template

Copy this template to `docs/work/VALIDATION_MATRIX.md` when bootstrapping a project or resetting proof tracking.

Before filling this template, read `docs/standards/VALIDATION.md`.

## Field Ownership

- Human owns proof overrides and acceptance sign-off.
- AI recommends proof types, links evidence, and updates status from verification.
- Shared fields include matrix rows and validation status.

## Copy And Fill Rules

- Replace `YYYY-MM-DD` with the update date.
- Use stable artifact IDs such as `REQ-001`, `PHASE-001`, `TICKET-001`, `BUG-001`, `TEST-001`.
- Set proof columns to `yes`, `no`, or `not_required`.
- Do not set `Status` to `implemented` unless the `Evidence` column links to real proof.
- If proof is `not_required`, record the reason in the linked ticket, bug, or verification artifact.

## Status Values

| Status | Meaning |
| --- | --- |
| planned | Accepted as intended behavior, not implemented |
| in_progress | Actively being built or verified |
| implemented | Implemented and evidence exists |
| changed | Contract or expected proof changed after earlier implementation |
| retired | No longer part of the accepted project contract |

## Matrix

| Requirement | Phase | Ticket/Bug | Contract/Behavior | Unit | Integration | E2E | UAT | Platform/Manual | Docs Review | Status | Evidence |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| REQ-001 | PHASE-001 | TICKET-001 | Example accepted behavior with observable outcome | yes | yes | not_required | yes | no | yes | planned | TEST-001, DOCS-REVIEW-001 |

## Evidence Rules

- Unit proof covers isolated domain, validation, parsing, or helper logic.
- Integration proof covers persistence, backend enforcement, service contracts, jobs, or provider behavior.
- E2E proof covers user-visible browser/app flows.
- UAT proof covers acceptance criteria and human sign-off for user-facing work.
- Platform/manual proof covers deployment, runtime, shell, mobile, desktop, or environment-specific behavior.
- Docs review proof covers documentation reconciliation against actual implementation.

## Update Log

| Date | Updated By | Change |
| --- | --- | --- |
| YYYY-MM-DD | AI | Initialized validation matrix. |
