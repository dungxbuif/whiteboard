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
updated: 2026-10-19
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
| blocked | Accepted proof cannot run until an external input or environment is available |
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
| REQ-04,NFR-01,NFR-08 | 1 | AUTH-HARDENING | Credential and invitation boundary hardening | yes | yes | yes | no | yes | yes | implemented | [Authentication hardening verification](verification/auth-hardening-2026-10-19.md): bounded strict JSON, active-user JWT checks, constant-cost unknown login, and privacy-preserving fixed-window throttling with `429 RATE_LIMITED` plus `Retry-After`. |
| P2-REQ-01 | 2 | EXC-015 | Frame Persistence & Scene Store Extension | yes | no | yes | no | no | yes | implemented | `npm run test:unit`; `npm run build`; presentation E2E flow. |
| P2-REQ-01 | 2 | EXC-016 | Frame Tool Activation | no | no | yes | no | no | yes | implemented | Native Excalidraw frame tool remains available; browser canvas flow passes. |
| P2-REQ-02 | 2 | EXC-017 | Presentation Engine & Controls | yes | no | yes | no | no | yes | implemented | Presentation store and controls; `npx playwright test` passes. |
| P2-REQ-03 | 2 | EXC-018 | Slide Navigator & Keyboard Support | no | no | yes | no | no | yes | implemented | Navigator controls and keyboard handling; browser presentation flow passes. |
| P3-REQ-01 | 3 | EXC-019 | Sharing DB Schema & Domain | yes | no | no | no | no | yes | implemented | Migration plus ScenePermission domain/repository contract. |
| P3-REQ-01 | 3 | EXC-020 | Sharing Backend | yes | yes | no | no | no | yes | implemented | Share/revoke usecase, bcrypt password handling, authenticated handler, and Go tests. |
| P3-REQ-02 | 3 | EXC-021 | Public API & Share Dialog UI | no | no | yes | no | no | yes | implemented | Public handler, share dialog, `/p/:token` read-only viewer, Playwright viewer test, and live production share smoke test. |
| P3-REQ-03 | 3 | EXC-022 | Password Protection | yes | no | yes | no | no | yes | implemented | bcrypt challenge response and public viewer password prompt. |
| P3-REQ-04 | 3 | EXC-023 | Sharing Integration Tests | yes | yes | yes | no | no | yes | implemented | Sharing usecase tests, public viewer browser flow, and full suite verification. |
| P4-REQ-03 | 4 | EXC-024 | Coturn & Docker Setup | no | no | no | yes | yes | yes | implemented | Compose/env checks plus generated two-network audio through the proven gateway TURN endpoint. VM100 cutover is V2-RELEASE. |
| P4-REQ-01 | 4 | EXC-025 | Legacy Socket.IO transport | yes | yes | no | no | no | yes | retired | Historical implementation was removed; custom /collab/ is the only accepted realtime transport. |
| V2-REQ-05 | 4 | EXC-026 | Native cursors, presence and follow | yes | yes | yes | yes | no | yes | implemented | Runtime tests and real-stack native two-browser presence/follow acceptance. |
| P4-REQ-01 | 4 | EXC-027 | Legacy WebRTC data-channel scene sync | yes | yes | no | no | no | yes | retired | Durable Yjs sync over /collab/ supersedes this contract; WebRTC remains voice media only. |
| V2-REQ-05 | 4 | EXC-028 | Voice Calling | yes | yes | yes | yes | yes | yes | implemented | Generated two-browser audio, ten-participant admission, mute/leave/denial cleanup, TURN evidence and continued edits pass locally. |
| P5-REQ-01 | 5 | EXC-029 | AI Domain & OpenAI-Compatible Repository | yes | yes | no | no | no | yes | implemented | OpenAI-compatible repository tests and AI contracts |
| P5-REQ-01 | 5 | EXC-030 | AI Proxy Endpoint & FE Client | yes | yes | no | no | no | yes | implemented | authenticated routes, AI store, backend tests |
| P5-REQ-01 | 5 | EXC-031 | Text-to-Diagram & Diagram-to-Code (Backend) | yes | yes | no | no | no | yes | implemented | output cleaning and scene validation tests |
| P5-REQ-02 | 5 | EXC-032 | AI Command Panel & Code Preview (Frontend) | no | no | yes | no | no | yes | implemented | in-canvas panel and frontend build |
| P5-REQ-03 | 5 | EXC-033 | Multimodal: Image-to-Diagram & Visual Chat | yes | no | yes | no | no | yes | implemented | base64 validation tests and image attachment UI |
| P5-REQ-04 | 5 | AI-PROVIDER-UAT | Public-origin live provider workflow | no | yes | yes | yes | yes | yes | blocked | Requires a user-provided backend-only OpenAI-compatible key. No production live-provider result is claimed. |

## Collaboration v2 proof

| Requirement | Tasks | Required proof | Status | Plan |
| --- | --- | --- | --- | --- |
| V2-REQ-01–02 | A1–A4 | A1 reviewed with 79 backend tests and stalled-body regression; A2 reviewed with 15 invitation tests plus origin validation and shared-clock regression; A3 reviewed with 16 public-link/credential tests; A4 scoped capability tests and deadline/renewal fixes pass with clean re-review; access/browser cases pass in the 17/17 real-stack run | implemented | [A](../superpowers/plans/2026-10-16-collaboration-access.md) |
| V2-REQ-03–05 | B1–B4 | B1–B4 implemented; durable ACK/replay/fault/lock proof, browser convergence/offline recovery, native presence/follow, generated two-browser audio, mute/leave, permission denial, and continued edits pass | implemented | [B](../superpowers/plans/2026-10-16-collaboration-runtime.md) |
| V2-REQ-01 | C1–C2 | C1 unit/browser/visual proof recorded; C2 access/share/routes and settings navigation pass 29 Chromium/35 unit tests, scoped review clean; route/role/share flows also pass the real-stack suite | implemented | [C](../superpowers/plans/2026-10-16-collaboration-dashboard.md) |
| V2-REQ-06–07 | D1–D4 | Scoped keys, HTTP/stdio discovery, native IDs, presets, durable mutations, duplicate/stale receipts, preservation of human edits, current export, and revoked downloads pass against the real stack | implemented | [D](../superpowers/plans/2026-10-16-collaboration-agents.md) |
| V2-REQ-08 | E1–E3 | Local real-stack 17/17 passes; 650/650 durable ACKs with p95 13.236 ms; native assets and actual two-browser audio pass. VM100 rollout, public-origin browser smoke, and fresh backup/rollback record remain | in_progress | [E](../superpowers/plans/2026-10-16-collaboration-release.md) |

C1–D4 implementation and local acceptance evidence: [verification](verification/collaboration-v2.md) and [real-stack acceptance](verification/collaboration-v2-real-stack.md). Only V2-REQ-08 remains in progress; existing v1 evidence is not reused as v2 proof.

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

## E1 real-stack acceptance — 2026-10-16

Evidence and safe reproduction: [real-stack acceptance](verification/collaboration-v2-real-stack.md). Dedicated command: `rtk npm run test:e2e:collab --workspace ./frontend` from src; use the bundled Node PATH recorded in evidence.

| Contract | Proof | State | Boundary |
| --- | --- | --- | --- |
| Actual password/invitation roles and owner/editor/viewer browser permissions; UUID/alias reload | Real production Go/Node/DB + browser | pass in 17/17 consolidated run | No HTTP route mocks |
| Offline edits/downgrade recovery; actual commit-before-ACK child kill/restart; epoch mismatch | Four real browser cases | pass | Guarded QA schema; production rollback drill remains |
| MCP HTTP/stdio real keys/native browser IDs/stale conflict/human preservation/scoped PNG and revoked grant | Real-stack MCP case | pass | Actual local renderer; VM100 smoke remains |
| Native paste/reload, actual signed S3 PUT/GET, failed PUT retention, renewal and byte equality | Real browser + SeaweedFS S3 | pass | First load and reload both fetched/rendered the native image |
| Native presence/follow and actual generated two-browser audio; mute/leave/denial cleanup | Real browsers and WebRTC media | pass | Local host proof; public-origin/TURN smoke remains |
| 50 clients each 1 native update/sec, p95 <1s, durable ACKs and convergence | Actual local runtime/process profiling | pass | 650/650 ACKs; p50 9.742ms, p95 13.236ms; peak RSS 399,097,856B |
