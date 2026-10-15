---
artifact_type: context
id: CONTEXT
status: active
owner: shared
updated: 2026-10-15
---

# Context

## Current Focus

- **Active Phase:** Reconciliation (Phases 1–5 implementation)
- **Active Work:** Realtime collaboration/voice, OpenAI-compatible AI, authenticated scene routes, and public sharing are deployed on VM100 and smoke-tested.
- **Next Work:** Rotate any credentials flagged by the release secret-rotation runbook before final production sign-off.

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
- Last-write-wins for scene persistence (D-06).
- Email/password is the active interactive login path; JWT middleware and authenticated API contracts remain stable. Google OAuth is not required by the runtime.
- Socket.IO is used for ephemeral room coordination; HTTP scene PATCH remains the persistence boundary.
- Coturn runs on the Homelab gateway at `turn.relayhub.dungxbuif.com:5349`; the app backend mints short-lived TURN REST credentials from `TURN_AUTH_SECRET`.
- OpenAI-compatible AI calls are backend-proxied and optional locally when `AI_API_KEY` is absent; DeepSeek is the default target.

## Blockers

- Final release sign-off still requires checking and rotating any previously exposed JWT or provider credentials; see [auth secret rotation runbook](operations/auth-secrets-rotation.md).

## Recent Changes

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
