---
artifact_type: framework_docs_guide
id: DOCS_README
status: active
owner: shared
human_fields: [framework_policy_changes, folder_purpose_changes]
ai_fields: [usage_guidance, artifact_explanations, maintenance_notes]
shared_fields: [status, links]
---

# Harness Docs Guide

This document explains the documentation system used by Harness. The root `README.md` explains the idea and philosophy; this file explains how to use the actual docs framework.

## Document Format

Harness documents use two layers:

1. YAML frontmatter for scripts and structured parsing.
2. Markdown body for humans and AI agents.

Typical frontmatter:

```yaml
---
artifact_type: ticket
id: TICKET-000
status: draft
owner: human
human_fields:
  - acceptance_criteria
ai_fields:
  - verification_results
shared_fields:
  - trace
trace:
  backlog_item: BL-000
  requirement: REQ-000
---
```

Rules:

- `artifact_type` identifies the document kind.
- `id` must be stable and unique inside its artifact family.
- `status` must use the status model when applicable.
- `owner` identifies the primary owner: `human`, `ai`, or `shared`.
- `human_fields` list fields the AI must not silently overwrite.
- `ai_fields` list fields the AI is expected to maintain.
- `shared_fields` list fields either side can update with care.
- `trace` links this artifact to upstream and downstream artifacts.

## Field Ownership

Each template also has a `Field Ownership` section in the Markdown body.

Use this rule:

- Human owns intent, priority, approval, acceptance, and business decisions.
- AI owns analysis, implementation evidence, docs reconciliation, trace maintenance, and context updates.
- Shared fields can be edited by both, but AI must not silently override human decisions.

If a required human-owned field is missing, the agent should mark work `blocked` or ask for input instead of inventing intent.

## Core Files

| File | Purpose | Primary Owner |
| --- | --- | --- |
| `../AGENTS.md` | System-prompt gateway and mandatory agent rules | Human |
| `CONTEXT.md` | Small shared project state for current work | Shared |
| `work/BACKLOG.md` | Runtime priority queue across requirements, tickets, bugs, maintenance, framework work | Shared |
| `work/ROADMAP.md` | Milestones and phase-level direction | Human |
| `work/TRACEABILITY.md` | Map requirement -> work -> design -> proof -> release | Shared |
| `work/VALIDATION_MATRIX.md` | Runtime proof state for accepted behavior | Shared |

## Folder Responsibilities

### `work/`

Runtime coordination surface.

Use it for:

- User feedback intake (FEEDBACK_LOG)
- Backlog ordering
- Roadmap and phase planning
- Tickets and bugs
- Completed work
- Traceability
- Validation evidence matrix

Important distinction:

```text
FEEDBACK_LOG.md = raw intake funnel
BACKLOG.md = what should be considered next
Ticket/Bug/Requirement/Phase = executable context packet
```

Do not use backlog rows as implementation specs unless the item is `tiny` and has a small-task exemption.

### `requirements/`

Product and behavior contract.

Use it for:

- Product specification
- Functional requirements
- Non-functional requirements
- User stories
- Acceptance criteria

Update this folder when user-facing behavior, accepted scope, or product requirements change.

### `architecture/`

Master technical docs.

Use it for:

- System overview
- Component boundaries
- API contracts
- ERD/data model
- Integrations
- Runtime/data flow

Architecture and component interaction diagrams should use ASCII diagrams. Sequence diagrams should use Mermaid `sequenceDiagram`.

### `decisions/`

Durable decisions.

Use ADRs when work changes:

- Architecture boundaries
- API/schema/auth/deployment behavior
- Major dependencies
- Security model
- Data ownership
- Workflow or standards

### `standards/`

Rules and quality bars.

Important files:

- `WORKFLOW.md`: lifecycle, backlog queue, intake lanes, gates, loop guard.
- `QUALITY_BAR.md`: what good tickets, designs, bugs, ADRs, diagrams, and master docs contain.
- `VALIDATION.md`: proof policy.
- `DEBUGGING.md`: root-cause-first debugging.
- `TESTING.md`: evidence and test expectations.
- `DOCS.md`: docs reconciliation.
- `GIT.md`: branch, commit, PR rules.
- `CODE.md`: code and dependency rules.

### `templates/`

Source templates for new artifacts.

Use these when creating new work artifacts:

- `TICKET.md`
- `BUG.md`
- `PHASE.md`
- `DETAIL_DESIGN.md`
- `FEEDBACK.md`
- `TEST_VERIFICATION.md`
- `VALIDATION_MATRIX.md`
- `DOCS_REVIEW.md`
- `UAT.md`
- `ADR.md`

Master-doc templates also live here:

- `SPEC.md`
- `ARCHITECTURE.md`
- `API.md`
- `ERD.md`
- `SDD.md`

### `engineering/`

Developer setup and local work guidance.

Use it for:

- Setup
- Local development
- Troubleshooting
- Developer commands

### `operations/`

Deployment and runtime operations.

Use it for:

- Deployment process
- Environments
- Release checklist
- Rollback notes

### `releases/`

Release history.

Use it for:

- Changelog
- Release notes
- Known issues
- Verification summary for shipped work

## Agent Lifecycle And Documents

| Lifecycle Phase | Required Docs |
| --- | --- |
| Hydration | `AGENTS.md`, `CONTEXT.md`, `work/BACKLOG.md`, relevant standards, active work item |
| Detail Design | `templates/DETAIL_DESIGN.md`, ticket/bug, relevant master docs |
| Execution & Test | active ticket/bug, `standards/TESTING.md`, `standards/VALIDATION.md` |
| Reconciliation | requirements, architecture, API, ERD, SDD, decisions, docs review |
| Dehydration | `CONTEXT.md`, `BACKLOG.md`, `VALIDATION_MATRIX.md`, active work status |

## Status Model

Use this primary status flow:

```text
draft -> ready -> in_progress -> blocked -> in_review -> verified -> done
```

Meaning:

- `draft`: not executable yet.
- `ready`: enough context exists to start.
- `in_progress`: work is active.
- `blocked`: needs human input, external dependency, or design decision.
- `in_review`: implementation or docs are awaiting review/reconciliation.
- `verified`: proof and docs review are complete.
- `done`: state, backlog, trace, validation, and release docs are updated.

## Creating A New Ticket

1. Add or reorder an item in `work/BACKLOG.md`.
2. If the item is not `tiny`, create a ticket from `templates/TICKET.md`.
3. Fill human-owned fields: intent, priority, scope, acceptance criteria.
4. Let AI fill impacted areas, test expectations, trace links, and verification sections.
5. If high-risk, require `DETAIL_DESIGN.md` approval before code.

## Creating A New Bug

1. Add or reorder the bug in `work/BACKLOG.md`.
2. Create a bug from `templates/BUG.md`.
3. Follow `standards/DEBUGGING.md`.
4. Record reproduction before fix.
5. Record root cause before implementation.
6. Add regression proof and update `work/VALIDATION_MATRIX.md`.

## Creating A Detail Design

Use `templates/DETAIL_DESIGN.md` for non-small work.

A good detail design must include:

- Problem
- Context loaded
- Brownfield touched scope when applicable
- Proposed approach
- Alternatives considered
- Impacted areas
- Test plan
- Validation matrix impact
- Reconciliation plan
- Approval state

## Validation And Evidence

Validation policy lives in `standards/VALIDATION.md`.

Runtime proof state lives in `work/VALIDATION_MATRIX.md`.

Verification evidence can live in:

- Ticket verification section
- Bug verification section
- `TEST_VERIFICATION.md`
- `UAT.md`
- `DOCS_REVIEW.md`
- Release notes

Do not mark behavior `implemented` in the validation matrix without evidence.

## Diagram Rules

Use diagrams only when they reduce ambiguity.

| Need | Required Format |
| --- | --- |
| Architecture overview | ASCII diagram in fenced `text` block |
| Component interaction | ASCII diagram in fenced `text` block |
| Sequence/request/event flow | Mermaid `sequenceDiagram` |
| ERD | Mermaid `erDiagram` or tables |
| State lifecycle | Mermaid `stateDiagram-v2` |
| Workflow/branching | Mermaid `flowchart` |
| Deployment/runtime environment | ASCII diagram |

See `standards/QUALITY_BAR.md#diagram-guide` for full rules.

## Completion Checklist

Before any work is called done:

- [ ] Active work item status updated.
- [ ] Tests or verification evidence recorded.
- [ ] Loop guard respected.
- [ ] `work/VALIDATION_MATRIX.md` updated or explicitly not affected.
- [ ] Master docs reconciled.
- [ ] ADR created or explicitly not needed.
- [ ] Docs review completed.
- [ ] `CONTEXT.md` updated.
- [ ] `work/BACKLOG.md` updated.
- [ ] Trace links updated.
- [ ] Release notes/changelog checked.
