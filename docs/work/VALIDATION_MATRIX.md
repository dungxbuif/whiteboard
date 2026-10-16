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
updated: 2026-10-15
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
| REQ-02 | 1 | EXC-003 | DB Migrations & Multi-tenant Schema | yes | yes | no | no | no | yes | implemented | Migration assertions and `go test ./...` pass. |
| REQ-02 | 1 | EXC-004 | Tenant Middleware & Workspace Repository | yes | yes | no | no | yes | yes | implemented | Tenant middleware and repository tests pass. |
| REQ-01 | 1 | EXC-005 | Shadcn UI & Tailwind Setup | yes | no | no | no | no | no | implemented | [EXC-005](tickets/phase-01/EXC-005/EXC-005-shadcn-ui-and-tailwind-setup.md) |
| REQ-04 | 1 | EXC-006 | Legacy Google OAuth Backend | yes | yes | yes | no | yes | yes | retired | Replaced as the active runtime login path by email/password; legacy identity fields remain for migration compatibility. |
| REQ-04 | 1 | EXC-007 | Frontend Auth Store | yes | no | yes | no | yes | yes | implemented | Email/password session handoff, authenticated dashboard flows, frontend unit/build, and Playwright coverage. |
| REQ-05 | 1 | EXC-008 | Workspace CRUD Full Stack | yes | yes | yes | no | no | yes | implemented | Workspace usecase/repository tests and dashboard browser flows pass. |
| REQ-06 | 1 | EXC-009 | Folder Backend Materialized Path | yes | yes | no | no | yes | no | implemented | [EXC-009](tickets/phase-01/EXC-009/EXC-009-folder-backend-materialized-path.md) |
| REQ-06 | 1 | EXC-010 | Folder Frontend Tree+Nav | yes | yes | yes | no | yes | no | implemented | [EXC-010](tickets/phase-01/EXC-010/EXC-010-folder-frontend-tree-nav.md) |
| REQ-10 | 1 | EXC-011 | Scene Backend | yes | yes | yes | no | yes | yes | implemented | [EXC-011](tickets/phase-01/EXC-011/EXC-011-scene-backend.md) |
| REQ-10 | 1 | EXC-012 | Excalidraw Integration | yes | yes | yes | yes | yes | yes | implemented | [EXC-012 evidence](tickets/phase-01/EXC-012/EXC-012-excalidraw-integration.md) |
| REQ-03 | 1 | EXC-013 | S3 Assets | yes | no | no | no | no | no | implemented | Unit tests for AssetUsecase and AssetHandler pass |
| REQ-08,09 | 1 | EXC-014 | Search + Starred/Trash | yes | no | yes | no | no | yes | implemented | `go test ./...`; `npm run test:unit`; `npm run build`; `npx playwright test` (11 passed) |
| REQ-04 | 1 | EXC-034 | Email/Password Auth and User Provisioning | yes | yes | yes | no | yes | yes | implemented | Password auth/userctl tests; default-workspace creation; frontend unit/build; `npx playwright test` (11 passed). Deployment credential rotation is tracked as an operational release gate. |
| P2-REQ-01 | 2 | EXC-015 | Frame Persistence & Scene Store Extension | yes | no | yes | no | no | yes | implemented | `npm run test:unit`; `npm run build`; presentation E2E flow. |
| P2-REQ-01 | 2 | EXC-016 | Frame Tool Activation | no | no | yes | no | no | yes | implemented | Native Excalidraw frame tool remains available; browser canvas flow passes. |
| P2-REQ-02 | 2 | EXC-017 | Presentation Engine & Controls | yes | no | yes | no | no | yes | implemented | Presentation store and controls; `npx playwright test` passes. |
| P2-REQ-03 | 2 | EXC-018 | Slide Navigator & Keyboard Support | no | no | yes | no | no | yes | implemented | Navigator controls and keyboard handling; browser presentation flow passes. |
| P3-REQ-01 | 3 | EXC-019 | Sharing DB Schema & Domain | yes | no | no | no | no | yes | implemented | Migration plus ScenePermission domain/repository contract. |
| P3-REQ-01 | 3 | EXC-020 | Sharing Backend | yes | yes | no | no | no | yes | implemented | Share/revoke usecase, bcrypt password handling, authenticated handler, and Go tests. |
| P3-REQ-02 | 3 | EXC-021 | Public API & Share Dialog UI | no | no | yes | no | no | yes | implemented | Public handler, share dialog, `/p/:token` read-only viewer, Playwright viewer test, and live production share smoke test. |
| P3-REQ-03 | 3 | EXC-022 | Password Protection | yes | no | yes | no | no | yes | implemented | bcrypt challenge response and public viewer password prompt. |
| P3-REQ-04 | 3 | EXC-023 | Sharing Integration Tests | yes | yes | yes | no | no | yes | implemented | Sharing usecase tests, public viewer browser flow, and full suite verification. |
| P4-REQ-03 | 4 | EXC-024 | Coturn & Docker Setup | no | no | no | no | yes | yes | implemented | `docker compose config`; compose/env docs |
| P4-REQ-01 | 4 | EXC-025 | Socket.io Backend + Frontend Client | yes | yes | no | no | no | yes | implemented | Go tests; frontend build; phase detail design |
| P4-REQ-01 | 4 | EXC-026 | Collaborative Cursors & Presence | yes | no | no | no | no | yes | implemented | realtime store and toolbar; frontend build |
| P4-REQ-01 | 4 | EXC-027 | WebRTC DataChannel + Scene Sync | yes | yes | no | no | no | yes | implemented | room event handlers; scene update loop guard |
| P4-REQ-02 | 4 | EXC-028 | Voice Calling | no | no | no | yes | yes | yes | implemented | WebRTC signaling and media controls; TURN config |
| P5-REQ-01 | 5 | EXC-029 | AI Domain & OpenAI-Compatible Repository | yes | yes | no | no | no | yes | implemented | OpenAI-compatible repository tests and AI contracts |
| P5-REQ-01 | 5 | EXC-030 | AI Proxy Endpoint & FE Client | yes | yes | no | no | no | yes | implemented | authenticated routes, AI store, backend tests |
| P5-REQ-01 | 5 | EXC-031 | Text-to-Diagram & Diagram-to-Code (Backend) | yes | yes | no | no | no | yes | implemented | output cleaning and scene validation tests |
| P5-REQ-02 | 5 | EXC-032 | AI Command Panel & Code Preview (Frontend) | no | no | yes | no | no | yes | implemented | in-canvas panel and frontend build |
| P5-REQ-03 | 5 | EXC-033 | Multimodal: Image-to-Diagram & Visual Chat | yes | no | yes | no | no | yes | implemented | base64 validation tests and image attachment UI |

## Collaboration v2 planned proof

| Requirement | Tasks | Required proof | Status | Plan |
| --- | --- | --- | --- | --- |
| V2-ACCESS | A1–A4 | A1 reviewed with 79 backend tests and stalled-body regression; A2 reviewed with 15 invitation tests plus origin validation and shared-clock regression; A3 reviewed with 16 public-link/credential tests; A4 scoped capability tests and deadline/renewal fixes pass with clean re-review | in_progress | [A](../superpowers/plans/2026-10-16-collaboration-access.md) |
| V2-SYNC | B1–B4 | B1 reviewed; B2 durable ACK/replay/fault/lock proof passes awaiting review; browser convergence/offline/voice pending | in_progress | [B](../superpowers/plans/2026-10-16-collaboration-runtime.md) |
| V2-DASH | C1–C2 | C1 unit/browser/visual proof recorded; C2 public-password regression passes; full access workflows implementing | in_progress | [C](../superpowers/plans/2026-10-16-collaboration-dashboard.md) |
| V2-MCP | D1–D4 | D1 scoped keys/HTTP+stdio discovery implemented awaiting review; D2 all preset fixtures pass; durable mutations/export integration pending | in_progress | [D](../superpowers/plans/2026-10-16-collaboration-agents.md) |
| V2-RELEASE | E1–E3 | Host inventory recorded in Homelab 42cdf59; real-stack E2E/load, cross-network audio and restore/rollback pending | in_progress | [E](../superpowers/plans/2026-10-16-collaboration-release.md) |

C1 execution evidence: [verification](verification/collaboration-v2.md). Remaining v2 slices are in progress; existing v1 evidence is not reused as v2 proof.

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
