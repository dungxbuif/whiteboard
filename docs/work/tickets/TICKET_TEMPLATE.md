---
artifact_type: ticket
id: TICKET-000
status: draft
owner: human
priority: TBD
lane: tiny|normal|high-risk|blocked
human_fields:
  - title
  - priority
  - acceptance_criteria
  - scope
  - approval
ai_fields:
  - impacted_areas
  - test_expectations
  - verification_results
  - docs_review
  - context_updates
shared_fields:
  - status
  - trace
  - small_task_exemption
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

# Ticket: <ID> <Title>

## Field Ownership

- Human fills intent, priority, acceptance criteria, scope, and approval.
- AI fills impact analysis, test expectations, verification evidence, docs review, and context/backlog updates.
- Shared fields include status, trace links, and small-task exemption.

## Status

- ID: TICKET-000
- Status: draft | ready | in_progress | blocked | in_review | verified | done
- Type: feature
- Priority: TBD
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

## Context

Describe the user need, business reason, or technical reason.

## Acceptance Criteria

- TBD

## Small Task Exemption

- Small task exemption: no
- Reason: TBD
- Impact checked: API=TBD, DB=TBD, Security=TBD, Runtime=TBD, Standards=TBD

## Impacted Areas

- Code: TBD
- Requirements docs: TBD
- Architecture docs: TBD
- API docs: TBD
- ERD/data docs: TBD
- Decisions: TBD

## Detail Design

- Required: TBD
- Link: TBD
- Approval: not required | pending | approved

## Test Expectations

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

## UAT

- Required: yes | no
- Reason if not required: TBD
- Expected behavior: TBD
- Verified behavior: TBD
- Sign-off: TBD

## Docs Review

- Requirements updated or not needed reason: TBD
- Architecture updated or not needed reason: TBD
- API updated or not needed reason: TBD
- ERD/data updated or not needed reason: TBD
- ADR created or not needed reason: TBD
- `docs/CONTEXT.md` updated: TBD

## Completion Checklist

- [ ] Implementation complete
- [ ] Tests run and recorded
- [ ] Fix/test loop guard respected
- [ ] Validation matrix updated or explicitly not affected
- [ ] UAT completed or explicitly not required
- [ ] Master docs reconciled
- [ ] Docs review completed
- [ ] ADR created or explicitly not needed
- [ ] `docs/CONTEXT.md` updated
- [ ] `docs/work/BACKLOG.md` updated
- [ ] Trace links updated
