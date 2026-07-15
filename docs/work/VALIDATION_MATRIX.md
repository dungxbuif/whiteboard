---
artifact_type: validation_matrix
id: VALIDATION_MATRIX
status: active
owner: shared
human_fields:
  - proof_override
  - acceptance_signoff
ai_fields:
  - proof_recommendation
  - evidence_links
  - status_updates
shared_fields:
  - matrix_rows
  - validation_status
updated: 2026-06-11
---

# Validation Matrix

## Field Ownership

- Human owns proof overrides and acceptance sign-off.
- AI recommends proof types, links evidence, and updates status from verification.

This file maps accepted behavior and work items to proof.

Policy lives in `docs/standards/VALIDATION.md`. This matrix is runtime project state and should change as work is planned, implemented, changed, or retired.

## Status Values

| Status | Meaning |
| --- | --- |
| planned | Accepted as intended behavior, not implemented |
| in_progress | Actively being built or verified |
| implemented | Implemented and evidence exists |
| changed | Contract or expected proof changed after earlier implementation |
| retired | No longer part of the accepted project contract |

## Matrix

| Requirement | Phase | Ticket/Bug | Contract/Behavior | Unit | Integration | E2E | UAT | Platform/Manual | Docs Review | Status | Evidence |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| REQ-01 | 1 | EXC-001 | Go Backend Scaffold | yes | yes | no | no | no | no | implemented | [EXC-001](tickets/phase-01/EXC-001/EXC-001-go-backend-scaffold.md) |
| REQ-01 | 1 | EXC-002 | Frontend Scaffold Vite+React | yes | no | no | no | no | no | implemented | [EXC-002](tickets/phase-01/EXC-002/EXC-002-frontend-scaffold-vite-react.md) |
| REQ-02 | 1 | EXC-003 | DB Migrations & Multi-tenant Schema | no | no | no | no | no | no | planned | none |
| REQ-02 | 1 | EXC-004 | Tenant Middleware & Workspace Repository | no | no | no | no | no | no | planned | none |
| REQ-01 | 1 | EXC-005 | Shadcn UI & Tailwind Setup | yes | no | no | no | no | no | implemented | [EXC-005](tickets/phase-01/EXC-005/EXC-005-shadcn-ui-and-tailwind-setup.md) |
| REQ-04 | 1 | EXC-006 | Google OAuth Backend | yes | yes | no | no | no | no | changed | Temporarily superseded by single-admin login; restore through EXC-034. |
| REQ-04 | 1 | EXC-007 | Frontend Auth Store | yes | no | yes | no | yes | no | implemented | Temporary admin login flow verified by unit and Playwright tests. |
| REQ-05 | 1 | EXC-008 | Workspace CRUD Full Stack | no | no | no | no | no | no | planned | none |
| REQ-06 | 1 | EXC-009 | Folder Backend Materialized Path | yes | yes | no | no | yes | no | implemented | [EXC-009](tickets/phase-01/EXC-009/EXC-009-folder-backend-materialized-path.md) |
| REQ-06 | 1 | EXC-010 | Folder Frontend Tree+Nav | yes | yes | yes | no | yes | no | implemented | [EXC-010](tickets/phase-01/EXC-010/EXC-010-folder-frontend-tree-nav.md) |
| REQ-10 | 1 | EXC-011 | Scene Backend | yes | yes | yes | no | yes | yes | implemented | [EXC-011](tickets/phase-01/EXC-011/EXC-011-scene-backend.md) |
| REQ-10 | 1 | EXC-012 | Excalidraw Integration | yes | yes | yes | yes | yes | yes | implemented | [EXC-012 evidence](tickets/phase-01/EXC-012/EXC-012-excalidraw-integration.md) |
| REQ-03 | 1 | EXC-013 | S3 Assets | yes | no | no | no | no | no | implemented | Unit tests for AssetUsecase and AssetHandler pass |
| REQ-08,09 | 1 | EXC-014 | Search + Starred/Trash | no | no | no | no | no | no | planned | none |
| REQ-04 | 1 | EXC-034 | Replace Temporary Admin Auth With Google OAuth | no | no | no | no | no | no | planned | [EXC-034](tickets/phase-01/EXC-034/EXC-034-replace-temporary-admin-auth-with-google-oauth.md) |
| P2-REQ-01 | 2 | EXC-015 | Frame Persistence & Scene Store Extension | no | no | no | no | no | no | planned | none |
| P2-REQ-01 | 2 | EXC-016 | Frame Tool Activation | no | no | no | no | no | no | planned | none |
| P2-REQ-02 | 2 | EXC-017 | Presentation Engine & Controls | no | no | no | no | no | no | planned | none |
| P2-REQ-03 | 2 | EXC-018 | Slide Navigator & Keyboard Support | no | no | no | no | no | no | planned | none |
| P3-REQ-01 | 3 | EXC-019 | Sharing DB Schema & Domain | no | no | no | no | no | no | planned | none |
| P3-REQ-01 | 3 | EXC-020 | Sharing Backend | no | no | no | no | no | no | planned | none |
| P3-REQ-02 | 3 | EXC-021 | Public API & Share Dialog UI | no | no | no | no | no | no | planned | none |
| P3-REQ-03 | 3 | EXC-022 | Password Protection | no | no | no | no | no | no | planned | none |
| P3-REQ-04 | 3 | EXC-023 | Sharing Integration Tests | no | no | no | no | no | no | planned | none |
| P4-REQ-03 | 4 | EXC-024 | Coturn & Docker Setup | no | no | no | no | no | no | planned | none |
| P4-REQ-01 | 4 | EXC-025 | Socket.io Backend + Frontend Client | no | no | no | no | no | no | planned | none |
| P4-REQ-01 | 4 | EXC-026 | Collaborative Cursors & Presence | no | no | no | no | no | no | planned | none |
| P4-REQ-01 | 4 | EXC-027 | WebRTC DataChannel + Scene Sync | no | no | no | no | no | no | planned | none |
| P4-REQ-02 | 4 | EXC-028 | Voice Calling | no | no | no | no | no | no | planned | none |
| P5-REQ-01 | 5 | EXC-029 | AI Domain & Gemini Repository | no | no | no | no | no | no | planned | none |
| P5-REQ-01 | 5 | EXC-030 | AI Proxy Endpoint & FE Client | no | no | no | no | no | no | planned | none |
| P5-REQ-01 | 5 | EXC-031 | Text-to-Diagram & Diagram-to-Code (Backend) | no | no | no | no | no | no | planned | none |
| P5-REQ-02 | 5 | EXC-032 | AI Command Panel & Code Preview (Frontend) | no | no | no | no | no | no | planned | none |
| P5-REQ-03 | 5 | EXC-033 | Multimodal: Image-to-Diagram & Visual Chat | no | no | no | no | no | no | planned | none |

## Rules

- Add or update a row when a requirement, ticket, bug, public contract, or accepted behavior is created or changed.
- Mark proof columns `yes`, `no`, or `not_required`.
- Link evidence to `docs/templates/TEST_VERIFICATION.md`, ticket verification sections, UAT, docs review, release notes, or command output summaries.
- Do not set `implemented` until required proof has evidence.
- If a proof type is `not_required`, record the reason in the linked ticket, bug, or verification artifact.

## Evidence Rules

- Unit proof covers pure domain and application rules.
- Integration proof covers backend enforcement, data integrity, provider behavior, jobs, or service contracts.
- E2E proof covers user-visible browser flows.
- Platform proof covers only shell, deployment, mobile, desktop, or runtime behavior that cannot be proven in lower layers.
- A ticket can be implemented without every proof column if the ticket explains why.
