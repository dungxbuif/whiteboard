---
artifact_type: traceability_matrix
id: TRACEABILITY
status: active
owner: shared
human_fields:
  - requirement_source
  - release_scope
ai_fields:
  - trace_links
  - evidence_links
  - adr_links
shared_fields:
  - matrix_rows
updated: TBD
---

# Traceability

## Field Ownership

- Human owns requirement source and release scope.
- AI maintains trace links, evidence links, and ADR links during reconciliation.

Use this file to map requirements to execution, verification, decisions, and releases.

## Trace Matrix

| Requirement | Phase | Ticket/Bug | Detail Design | Test Verification | Docs Review | ADR/Docs | Release |
| --- | --- | --- | --- | --- | --- | --- | --- |
| TBD | TBD | TBD | TBD | TBD | TBD | TBD | TBD |

## Rules

- Every phase should link to at least one requirement, ticket, or bug.
- Every ticket or bug should link to test evidence.
- Every durable decision should link to an ADR.
- Every release should link to completed phases, tickets, bugs, and verification.
