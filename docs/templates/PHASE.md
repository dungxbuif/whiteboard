---
artifact_type: phase
id: PHASE-000
status: draft
owner: human
human_fields:
  - goal
  - scope
  - out_of_scope
  - priority
  - success_criteria
ai_fields:
  - risks
  - dependencies
  - verification_plan
  - completion_summary
shared_fields:
  - status
  - trace
  - tickets_and_bugs
trace:
  backlog_items: []
  roadmap: TBD
  requirements: []
  tickets: []
  bugs: []
  test_verification: TBD
  validation_matrix: TBD
  adrs: []
  release_notes: TBD
---

# Phase Template

Copy this template to `docs/work/phases/PHASE-<id>-<short-name>.md`.

## Copy And Fill Rules

- Human owns goal, scope, priority, and success criteria.
- AI may propose risks, dependencies, verification plan, ticket/bug grouping, and completion summary.
- Replace `PHASE-000` with `PHASE-<id>-<short-name>`.
- A phase is not executable until it has linked work items or an explicit discovery goal.

## Field Ownership

- Human fills goal, scope, out of scope, priority, and success criteria.
- AI fills risks, dependencies, verification plan, and completion summary.
- Shared fields include status, trace links, and ticket/bug list.

## Status

- ID: PHASE-000
- Status: draft | ready | in_progress | blocked | in_review | verified | done
- Owner: TBD
- Created: TBD
- Updated: TBD

## Trace Links

- Backlog items: TBD
- Roadmap: TBD
- Requirements: TBD
- Tickets: TBD
- Bugs: TBD
- Test verification: TBD
- Validation matrix: TBD
- ADRs: TBD
- Release notes: TBD

## Goal

TBD

## Scope

- TBD

## Out Of Scope

- TBD

## Tickets And Bugs

| ID | Type | Title | Status | Link |
| --- | --- | --- | --- | --- |
| TBD | TBD | TBD | draft | TBD |

## Dependencies

- TBD

## Risks

- TBD

## Success Criteria

- TBD

## Verification Plan

- TBD

## Gate Checklist

- [ ] Phase has linked requirements or explicit discovery goal
- [ ] Tickets/bugs are created or planned
- [ ] Risks and dependencies are recorded
- [ ] Verification plan is defined
- [ ] Release/changelog need is identified

## Completion Summary

TBD
