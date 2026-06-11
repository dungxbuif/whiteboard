# Workflow Conventions

## Task Lifecycle

1. Hydration
2. Detail Design
3. Execution & Test
4. Reconciliation
5. Dehydration

## Runtime Work Selection

`docs/work/BACKLOG.md` is the runtime priority queue. It decides what should be worked on next, but it does not replace execution artifacts.

Agents MUST:

- Read `docs/work/BACKLOG.md` during hydration.
- Use backlog rank, priority, severity, readiness, and blockers to identify the next work item.
- Update `docs/CONTEXT.md` with the current queue focus.
- Promote non-tiny backlog items into a requirement slice, phase, ticket, or bug before execution.
- Avoid assuming all tickets must finish before bugs or new requirements. Runtime priority can reorder work.

## User Feedback Intake

Raw user feedback, feature requests, and bug reports from users must be recorded in `docs/work/FEEDBACK_LOG.md` before they can enter the `BACKLOG.md`.
1. **Intake:** Log the raw text in `FEEDBACK_LOG.md`.
2. **Triage:** Classify as `Bug`, `Feature`, `Enhancement`, or `Noise`.
3. **Conversion:** Only triaged items that are actionable can be converted into a `BUG.md` or `TICKET.md` and added to `BACKLOG.md`. All converted artifacts must link back to their origin Feedback ID.

## Intake Lanes

Every backlog item MUST have one lane:

| Lane | Meaning | Execution Rule |
| --- | --- | --- |
| tiny | Low-risk, isolated, reversible work | May run from backlog with small-task exemption |
| normal | Bounded story-sized feature, bug, docs, or maintenance work | Requires ticket or bug before execution |
| high-risk | Touches security, auth, data, public contracts, external systems, deployment, or major dependency | Requires ticket/bug, detail design, approval, and stronger validation |
| blocked | Cannot proceed without missing input or dependency | Do not execute until blocker is resolved |

High-risk hard gates:

- Auth or authorization
- Data model, migration, data deletion, retention, or data ownership
- Audit, privacy, secrets, or security behavior
- External provider, payment, email, queue, webhook, or cloud service behavior
- Public API, event, CLI, or user-visible contract
- Deployment/runtime configuration
- Removing or weakening validation requirements
- Repository standards changes

## Status Model

Phases, tickets, and bugs MUST use this primary status flow:

```text
draft -> ready -> in_progress -> blocked -> in_review -> verified -> done
```

- `draft`: artifact is being shaped and is not executable.
- `ready`: artifact has enough context and acceptance criteria to start.
- `in_progress`: implementation or investigation is active.
- `blocked`: progress needs human input or an external dependency.
- `in_review`: implementation is complete and awaiting review or reconciliation.
- `verified`: tests, docs review, and UAT requirements are satisfied.
- `done`: work is closed and context/release state is updated.

Bug work MAY record secondary markers: `reproduced`, `fixed`, `regression_verified`.

## Small Task Criteria

A task is small only if it is isolated, reversible, and does not affect:

- Public API or user-facing behavior
- Database schema or migrations
- Architecture boundaries
- Authentication, authorization, security, or permissions
- Deployment/runtime configuration
- Major dependencies
- Repository standards

Small tasks may proceed without human approval only when the artifact records this evidence:

```text
Small task exemption: yes
Reason: TBD
Impact checked: API=no, DB=no, Security=no, Runtime=no, Standards=no
```

All other tasks require detail design approval.

## Required Gate Checklists

Hydration MUST include:

- [ ] `AGENTS.md` read
- [ ] `docs/CONTEXT.md` read
- [ ] `docs/work/BACKLOG.md` read
- [ ] Active phase/ticket/bug read
- [ ] Relevant standards read
- [ ] Relevant master docs read
- [ ] Work lane and risk flags identified
- [ ] Brownfield touched scope identified, when applicable

Detail Design MUST include:

- [ ] Problem and context
- [ ] Proposed approach
- [ ] Impacted areas
- [ ] Test plan
- [ ] Reconciliation plan
- [ ] Approval or small-task exemption

Execution & Test MUST include:

- [ ] Implementation complete
- [ ] Tests written or updated when needed
- [ ] Real command results recorded
- [ ] Manual checks recorded when needed
- [ ] Validation matrix updated when behavior proof changed
- [ ] Fix/test attempt count recorded when tests fail

Reconciliation MUST include:

- [ ] Requirements docs checked
- [ ] Architecture docs checked
- [ ] API docs checked
- [ ] ERD/data docs checked
- [ ] SDD docs checked (Detail Design changes synchronized to SDD)
- [ ] ADR need checked
- [ ] Docs review completed

Dehydration MUST include:

- [ ] `docs/CONTEXT.md` updated
- [ ] `docs/work/BACKLOG.md` updated
- [ ] Work status updated
- [ ] Trace links updated
- [ ] Release notes/changelog checked

## Definition Of Done

- Implementation matches the ticket or bug.
- Tests were run and recorded.
- UAT is complete or explicitly not required.
- Master docs were reconciled.
- Docs review checklist is complete.
- ADRs were created when durable decisions changed.
- `docs/CONTEXT.md` was updated.
- Task status was updated.

## Loop Guard

Agents MUST NOT run endless fix/test loops.

Stop and mark the work item `blocked` when any threshold is reached:

| Threshold | Meaning | Required Action |
| --- | --- | --- |
| 3 same-path failures | The same test, command, or verification path still fails after 3 fix attempts | Stop, summarize attempts, request human/design review |
| 5 total cycles | The work item has consumed 5 fix/test cycles, even across different failures | Stop, summarize current state, request review |
| Scope expansion | A fix requires architecture, API, data, security, runtime, or standards changes outside approved scope | Stop and require detail design or ADR |
| Moving failure | Each attempted fix reveals a new unrelated failure | Stop and reassess design/root cause |

When stopped by loop guard, record:

- Attempt count
- Commands run
- What changed in each attempt
- Failure output summary
- Current root-cause hypothesis
- What human decision or input is needed
