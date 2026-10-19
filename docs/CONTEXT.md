---
artifact_type: context
id: CONTEXT
status: active
owner: shared
updated: 2026-10-19
---

# Context

## Current Focus

- **Active Phase:** Paused before production release — [Collaboration v2](work/phases/PHASE-COLLAB-V2.md).
- **Active Work:** Implementation is complete and is being committed/merged into local `master` as the pause checkpoint. Production remains on v1; no VM100 deployment was started.
- **Next Work:** Resume [COLLAB-V2-RELEASE](work/BACKLOG.md): run the final sequential repository gate, take a fresh cutover backup, install distinct v2 service secrets, build and record Linux/amd64 images, migrate6→13, deploy, run the public browser/authorization/S3/MCP/WebSocket/voice canary, then reconcile Homelab and close the release note/checklist. Machine and human handoff live under `.planning/`.

## Project

Excalidraw Ecosystem Rewrite — self-hosted whiteboard with Go Clean Architecture backend and React Vite frontend.

## Key State

- Planning covers all 5 phases (34 tickets total).
- Phase 1 has 15 tickets (EXC-001 to EXC-014 plus EXC-034).
- EXC-001 through EXC-034 are implemented; external credential rotation remains a release operation.
- Architecture, requirements, and ADR documentation migrated to harness format.

## Active Decisions

- Materialized Path for folder hierarchy (D-01).
- UUIDs for primary keys (D-02).
- Single active workspace per session (D-03).
- Context-based tenancy via middleware (D-04).
- Legacy v1 uses last-write-wins scene persistence (D-06); promoted v2 scenes use the durable Yjs writer and reject legacy content PATCH.
- Email/password is the active interactive login path; JWT middleware and authenticated API contracts remain stable. Google OAuth is not required by the runtime.
- Production v1 uses Socket.IO coordination and HTTP scene PATCH until cutover. The completed v2 release replaces Socket.IO with one custom authenticated WebSocket runtime; promoted scenes reject HTTP content PATCH.
- Shared Coturn runs on the gateway. The configured hostname currently resolves incorrectly on the local network; two-network relay/audio passed with `turn:103.82.21.202:5349`. Release configuration will use that verified endpoint until split DNS is corrected and tested. Credentials remain short-lived and API-issued.
- OpenAI-compatible AI calls are backend-proxied and optional locally when `AI_API_KEY` is absent; DeepSeek is the default target.

## Blockers

- Final release sign-off still requires checking and rotating any previously exposed JWT or provider credentials; see [auth secret rotation runbook](operations/auth-secrets-rotation.md).

## Recent Changes

- 2026-10-19: Collaboration v2 local acceptance passed 17/17, including 650/650 durable ACKs at p95 13.236 ms, native S3 image reload/recovery, MCP writes/exports, presence/follow and generated two-browser audio. Legacy Socket.IO code and dependencies were removed. VM100 rollout remains.

- 2026-10-17: B3 revision-conflict review closed (`d42dd02`); recovery slice reviewed (`403f7c9`); architecture/backup reconciliation (`0eef886`); bounded worker/fanout performance repair (`f2dd6cf`) reviewed. [Database restore evidence](work/verification/collaboration-v2-backup.md).

- 2026-10-16: A1 authority/aliases/migration committed and reviewed, including a real stalled-request regression protecting owner revocation. B1 workspace/protocol and D2 catalog are implemented under review; frontend33 unit tests/build and14 browser tests pass after dependency migration. [Evidence](work/verification/collaboration-v2.md). Homelab commit42cdf59 records actual topology and recovery of the existing v1 frontend; no v2 deployment.

- 2026-10-16: Implementation started in codex/collaboration-v2. User removed the monochrome override; preserve the old dashboard's complete style, including colored accents. Baseline: 59 Go tests and 19 frontend unit tests passed.

- 2026-10-16: Recorded approved v2 design and five implementation slices (17 tasks), with tests, dependencies, documentation targets, canary and rollback; user confirmed org means workspace. No v2 runtime code shipped.

- 2026-07-15: Completed EXC-013 S3 assets backend, caching logic, frontend hook, and tests.
- 2026-07-15: Approved EXC-013 S3 assets detail design and moved to in_progress.
- 2026-06-11: Migrated all documentation from old devtool/docs format to harness framework structure.
- 2026-06-12: Completed EXC-010 nested folder tree, breadcrumbs, folder listing, and creation UI.
- 2026-06-12: Completed EXC-011 workspace-scoped scene JSONB persistence and lifecycle APIs.
- 2026-06-12: Completed EXC-012 embedded Excalidraw, debounced scene sync, and scene actions UI.
- 2026-06-05: Implemented optimized Materialized Path folder backend (EXC-009).
- 2026-06-04: Implemented Workspace CRUD (EXC-008) and auth/tenant (EXC-006, EXC-004).
- 2026-10-15: Completed EXC-014 workspace-scoped search, Starred/Trash views, and restore actions; backend and frontend unit/build checks pass.
- 2026-10-15: Added authenticated scene deep links and runtime workspace-owner authorization for workspace-scoped routes.
- 2026-10-15: Required JWTs for realtime handshakes, enforced workspace-owner checks on room joins, and upgraded the frontend toolchain to TypeScript 7.0.2.
- 2026-10-15: Replaced the interactive Google OAuth runtime path with email/password login, `userctl` provisioning, and default-workspace creation; legacy identity columns remain migration-compatible and deployment credential rotation remains operational follow-up.
- 2026-10-15: Completed EXC-015 through EXC-018 frame persistence, presentation mode, slide navigator, and keyboard navigation; frontend and Playwright checks pass.
- 2026-10-15: Completed EXC-019 through EXC-023 public sharing, revoke, password challenge, share dialog, and read-only viewer; backend and Playwright checks pass.
- 2026-10-15: Completed EXC-024 through EXC-028 realtime rooms, presence/cursors, scene sync, WebRTC voice signaling, and coturn compose configuration.
- 2026-10-15: Completed EXC-029 through EXC-033 with an OpenAI-compatible DeepSeek proxy, diagram/code generation, image input validation, visual chat, and in-canvas AI controls.
- 2026-10-15: Fixed public viewer hydration for JSON-persisted collaborator state; unit, browser, build, health, and live public-share checks pass on VM100.
