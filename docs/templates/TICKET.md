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

# Ticket Template

Copy this template to `docs/work/tickets/TICKET-<id>-<short-name>.md`.

Before filling this template, read `docs/standards/QUALITY_BAR.md#good-ticket`.

## Copy And Fill Rules

- Human must fill: title, priority, scope, acceptance criteria, and approval when required.
- AI must fill: impacted areas, test expectations, trace links, validation impact, verification results, docs review, context/backlog updates.
- Replace `TICKET-000` with `TICKET-<id>-<short-name>` in frontmatter and body.
- Keep this ticket small enough for one implementation slice. Split if acceptance criteria cover unrelated behavior.
- If the ticket is tiny, fill `Small Task Exemption`. Otherwise create or link a `DETAIL_DESIGN.md`.
- Never mark `done` until completion checklist is fully satisfied.

## Field Ownership

- Human fills intent, priority, acceptance criteria, scope, and approval.
- AI fills impact analysis, test expectations, verification evidence, docs review, and context/backlog updates.
- Shared fields include status, trace links, and small-task exemption.

## Status

- ID: TICKET-000
- Status: draft | ready | in_progress | blocked | in_review | verified | done
- Type: feature | task | refactor | docs
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

## Context

Human fill:

- User/business/system problem:
- Source prompt or requirement:
- Out of scope:

AI fill:

- Current repository context read:
- Brownfield touched scope, if applicable:

## Acceptance Criteria

Human fill observable criteria:

- [ ] Given <context>, when <action>, then <expected result>.
- [ ] Error/failure behavior is defined.
- [ ] UAT requirement is clear: required | not_required because <reason>.

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

- Required: yes | no
- Link: TBD
- Approval: not required | pending | approved

## Test Expectations

- Unit:
- Integration:
- E2E:
- UAT:
- Manual/platform:
- Docs review:

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
