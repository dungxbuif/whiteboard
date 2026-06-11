---
artifact_type: erd
id: ERD-000
status: draft
owner: shared
human_fields:
  - data_ownership_decisions
  - migration_approval
ai_fields:
  - entities
  - relationships
  - constraints
  - migrations
  - erd_diagram
shared_fields:
  - status
  - trace
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

# ERD Template

Before filling this template, read `docs/standards/QUALITY_BAR.md#diagram-guide` and `docs/standards/QUALITY_BAR.md#good-master-doc-update`.

## Copy And Fill Rules

- Human owns data ownership and migration approval.
- AI documents entities, relationships, constraints, and migration notes from accepted design or implementation.
- Use Mermaid `erDiagram` when relationships matter.
- Data ownership, deletion, retention, or migration decisions may require ADR.

## Field Ownership

- Human owns data ownership and migration approval decisions.
- AI fills entities, relationships, constraints, migration notes, and ERD diagram from implementation evidence.

## Status

- ID: ERD-000
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

## Entities

| Entity/Table | Purpose | Owner | Notes |
| --- | --- | --- | --- |
| TBD | TBD | TBD | TBD |

## ERD Diagram

Use Mermaid `erDiagram` when relationships matter.

```mermaid
erDiagram
    ENTITY_A ||--o{ ENTITY_B : owns
```

## Relationships

| From | To | Relationship | Notes |
| --- | --- | --- | --- |
| TBD | TBD | TBD | TBD |

## Constraints

- TBD

## Migrations

- TBD

## Linked Decisions

- TBD
