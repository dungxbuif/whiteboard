---
artifact_type: architecture_doc
id: ARCH-000
status: draft
owner: shared
human_fields:
  - architectural_constraints
  - approved_boundaries
  - tradeoff_approval
ai_fields:
  - overview
  - diagrams
  - modules
  - data_flow
  - runtime_flow
  - dependencies
  - risks_and_tradeoffs
shared_fields:
  - status
  - trace
  - linked_decisions
trace:
  requirements: []
  phases: []
  tickets_or_bugs: []
  detail_designs: []
  test_verification: TBD
  docs_review: TBD
  adrs: []
  release_notes: TBD
---

# Architecture Template

Before filling this template, read `docs/standards/QUALITY_BAR.md#diagram-guide` and `docs/standards/QUALITY_BAR.md#good-master-doc-update`.

## Copy And Fill Rules

- Human owns approved boundaries and tradeoff approval.
- AI updates architecture only from accepted design or implemented evidence.
- Use ASCII diagrams for architecture overview and component interaction.
- Use Mermaid `sequenceDiagram` only when order of runtime interaction matters.
- Link ADRs for durable architecture decisions.

## Field Ownership

- Human owns approved constraints, boundaries, and tradeoff approvals.
- AI drafts overview, diagrams, module maps, flows, dependencies, and risks from implementation evidence.

## Status

- ID: ARCH-000
- Status: draft | ready | in_progress | blocked | in_review | verified | done
- Owner: TBD

## Trace Links

- Requirements: TBD
- Phases: TBD
- Tickets/Bugs: TBD
- Detail designs: TBD
- Test verification: TBD
- Docs review: TBD
- ADRs: TBD
- Release notes: TBD

## Overview

TBD

## Architecture Overview Diagram

Use an ASCII diagram in a fenced `text` block.

```text
+-------------+        +-------------+        +-------------+
| Component A | -----> | Component B | -----> | Component C |
+-------------+        +-------------+        +-------------+
```

## System Boundaries

- TBD

## Modules

| Module | Responsibility | Key Files | Notes |
| --- | --- | --- | --- |
| TBD | TBD | TBD | TBD |

## Data Flow

Use ASCII or Mermaid flowchart, whichever is clearer.

TBD

## Runtime Flow

Use Mermaid `sequenceDiagram` when order of interaction matters.

TBD

## Dependencies

- TBD

## Risks And Tradeoffs

- TBD

## Linked Decisions

- TBD
