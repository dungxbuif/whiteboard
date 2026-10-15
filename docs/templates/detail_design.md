---
artifact_type: detail_design
id: DESIGN-000
status: draft
owner: ai
approval: pending
human_fields:
  - approval
  - constraints
  - scope_decisions
ai_fields:
  - problem
  - context_loaded
  - brownfield_scope
  - proposed_approach
  - design_tradeoffs
  - architecture_overview
  - execution_flow
  - api_data_model
  - security
  - test_plan
  - reconciliation_plan
shared_fields:
  - status
  - trace
  - small_task_exemption
trace:
  backlog_item: TBD
  requirement: TBD
  phase: TBD
  ticket_or_bug: TBD
  test_verification: TBD
  validation_matrix: TBD
  docs_review: TBD
  adrs: []
  master_docs_touched: []
---

# DETAIL DESIGN: [Feature/Bug Name]

Use for non-small tickets and all meaningful bug fixes.
Before filling this template, read `docs/standards/QUALITY_BAR.md#good-detail-design`.

## Copy And Fill Rules

- AI drafts this document; Human approves or blocks it.
- Replace `DESIGN-000` with a stable design ID.
- Link the source ticket or bug before writing the approach.
- State assumptions explicitly. Do not hide product, API, data, security, or runtime assumptions.
- **Diagram Rule:** Only draw Mermaid diagrams for Architecture Overview and Execution Flow if the scope of the task requires that level of detail. Otherwise, use text lists.
- If this design changes architecture, API, data, security, deployment, dependency, or standards, create or link an ADR.
- Implementation must not begin until approval is `approved`, unless `Small Task Exemption` is valid.

## Field Ownership

- Human owns approval, constraints, and scope decisions.
- AI owns analysis, architecture, flows, models, test plan, and reconciliation plan.
- Shared fields include status, trace links, and small-task exemption.

## Status

- ID: DESIGN-000
- Status: draft | ready | in_progress | blocked | in_review | verified | done
- Ticket/Bug: TBD
- Approval: not required | pending | approved
- Author: TBD
- Updated: TBD

## Trace Links

- Backlog item: TBD
- Requirement: TBD
- Phase: TBD
- Ticket/Bug: TBD
- Test verification: TBD
- Validation matrix: TBD
- Docs review: TBD
- ADRs: TBD
- Master docs touched: TBD

---

## 1. Context & Scope

### Problem Statement
- Problem statement: TBD
- Why now: TBD
- Success outcome: TBD

### Context Loaded
- `docs/CONTEXT.md`
- TBD

### Brownfield Scope
- Touched modules/files: TBD
- Direct dependencies inspected: TBD
- Contracts affected: TBD
- Known unknowns: TBD
- Scope expansion reason: TBD

### Small Task Exemption
- Small task exemption: no
- Reason: TBD
- Impact checked: API=TBD, DB=TBD, Security=TBD, Runtime=TBD, Standards=TBD

---

## 2. Design Considerations & Trade-offs

Discuss the technical trade-offs, alternatives considered, and why the proposed approach was chosen.

| Consideration / Alternative | Pros | Cons | Decision |
| --- | --- | --- | --- |
| TBD | TBD | TBD | chosen/rejected |

---

## 3. Architecture Overview

Provide an overview of the system changes. Use Mermaid ASCII diagram if the scope requires visual clarity.

### Component Responsibilities

| Component | Role |
| --- | --- |
| TBD | TBD |

---

## 4. Execution Flow

Describe the sequence of operations or execution path. Use Mermaid `sequenceDiagram` if the logic is complex.

1. Step 1
2. Step 2
3. Step 3

---

## 5. API & Data Model Design

### API Changes
- Endpoint: `METHOD /path`
- Request: TBD
- Response: TBD

### Data Model Changes
- Table / Collection: TBD
- Added / Modified fields: TBD
- Schema Migration needed: yes/no

---

## 6. Security & Authorization

- Authentication changes: TBD
- Authorization / Permissions: TBD
- Data Privacy / PII impact: TBD
- Input Validation: TBD

---

## 7. Implementation & Verification Plan

### Impacted Areas
- Code/modules: TBD
- Product behavior: TBD
- API/contracts: TBD
- Data/schema: TBD
- Security/auth: TBD
- Deployment/runtime: TBD
- Docs: TBD

### Bug Analysis (If Applicable)
- Reproduction: TBD
- Root cause: TBD
- Impact scope: TBD
- Fix strategy: TBD
- Regression risk: TBD

### Test Plan
- Unit: TBD
- Integration: TBD
- E2E: TBD
- UAT: TBD
- Manual/platform: TBD
- Docs review: TBD

### Validation Matrix Impact
- Update required: yes | no
- Row(s): TBD
- Reason if no update required: TBD

### Verification Results
*Fill after execution.*
- Command: TBD
- Result: TBD
- Notes: TBD

---

## 8. Reconciliation Plan

Update master docs and decisions to reflect this design.

- Requirements docs: update | no change
- Architecture docs: update | no change
- API docs: update | no change
- ERD docs: update | no change
- SDD docs: update | no change
- ADR: create | no change
- Context: update

### Docs Review Checklist
- Code changed but docs unchanged reason: TBD
- Requirements updated or not needed reason: TBD
- Architecture updated or not needed reason: TBD
- API updated or not needed reason: TBD
- ERD/data updated or not needed reason: TBD
- SDD updated or not needed reason: TBD
- ADR created or not needed reason: TBD
