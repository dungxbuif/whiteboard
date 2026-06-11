---
artifact_type: software_design_document
id: SDD-000
status: draft
owner: architect
approval: pending
human_fields:
  - approval
  - architectural_constraints
  - business_alignment
ai_fields:
  - executive_summary
  - system_context
  - architectural_decisions
  - component_design
  - data_design
  - security_and_compliance
  - infrastructure_and_deployment
  - failure_modes_and_resilience
  - migration_and_compatibility
  - open_questions
  - feedback_log
shared_fields:
  - status
  - trace
trace:
  backlog_item: TBD
  requirement: TBD
  phase: TBD
  ticket_or_bug: TBD
  validation_matrix: TBD
  docs_review: TBD
  adrs: []
  master_docs_touched: []
---

# Software Design Document (SDD)

Use for high-level system/architectural changes, structural refactoring, or new system/service design.

Before filling this template, read `docs/standards/QUALITY_BAR.md#good-detail-design` (adapted for high-level design).

## Copy And Fill Rules

- This document is co-authored: AI Architect drafts the structure, Human aligns constraints and business goals.
- Replace `SDD-000` with a stable SDD ID.
- Link relevant high-level backlog phases/requirements.
- Implementation of features under this architecture MUST not begin until this SDD is `approved`.

## Status

- ID: SDD-000
- Status: draft | ready | in_progress | blocked | in_review | verified | done
- Approval: not required | pending | approved
- Author/Architect: TBD
- Updated: TBD

## Trace Links

- Backlog item: TBD
- Requirement: TBD
- Phase: TBD
- ADRs: TBD
- Master docs touched: TBD

## 1. Executive Summary

AI fill:
- Context & Objectives (High-level goal):
- Target Audience & Scope:
- Business Value & Success Metrics:

## 2. System Context & Boundary

AI fill (ASCII Diagram showing boundaries, users, and external integrations):

```text
+-------------+        +-------------+        +-------------+
| User/Client | -----> | New System  | -----> | Ext Service |
+-------------+        +-------------+        +-------------+
```

- System Boundary description:
- External Dependencies & Integrations:

## 3. Architectural Decisions & Key Choices

AI fill:
- Key design patterns (e.g., Microservices, Event-Driven, Monolith):
- Technology stack selection & rationale:
- ADR Links:

## 4. Component Design

AI fill (ASCII Component Diagram detailing internal modules and relationships):

```text
+-------------------------------------------------------------+
|                       New System                            |
|  +------------------+             +----------------------+  |
|  |    Module A      | ----------> |       Module B       |  |
|  +------------------+             +----------------------+  |
+-------------------------------------------------------------+
```

- Component Responsibilities:
- Communication Protocols (HTTP/gRPC/Event Bus):

## 5. Data Design

AI fill (Entity Relationship Diagram or Schema tables):
- DB Engine & Storage Strategy:
- ERD or Key Tables/Collections:
- Data retention, archiving, and privacy constraints:

## 6. Security & Compliance

AI fill:
- Authentication & Authorization flow:
- Data encryption (at rest and in transit):
- Risk analysis (STRIDE model or similar):

## 7. Infrastructure & Deployment Design

AI fill (ASCII deployment topology):
- Hosting/Cloud Provider specifications:
- Scaling & Redundancy models:
- CI/CD pipeline modifications:

## 8. Failure Modes & Resilience (Kịch bản lỗi & Phục hồi)

AI fill:
- Key Failure Scenarios (e.g., DB down, network partition, timeout):
- Graceful degradation strategies (circuit breakers, fallbacks):
- Monitoring, Alerting, & Observability:

## 9. Migration & Compatibility

AI fill:
- Backward/Forward compatibility details:
- Migration steps (Zero-downtime, blue-green, database migrations):
- Rollback strategy:

## 10. Open Questions

AI fill (Design tradeoffs requiring human discussion):
- TBD

## 11. Feedback Log & Revision History

Human/AI shared:
- Log of review sessions, feedback items, and how they were resolved.

| Date | Reviewer | Feedback Item | Resolution / Action Taken | Status |
| --- | --- | --- | --- | --- |
| TBD | Human/AI | TBD | TBD | open/resolved |
