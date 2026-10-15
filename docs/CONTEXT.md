---
artifact_type: context
id: CONTEXT
status: active
owner: shared
updated: 2026-10-15
---

# Context

## Current Focus

- **Active Phase:** PHASE-1 (Self-hosted Dashboard & Core Infrastructure)
- **Active Work:** Phase 3 sharing is implemented and verified across backend, frontend, and browser layers.
- **Next Work:** Continue with Phase 4 collaboration, while tracking deployment credential rotation as the Phase 1 release operation.

## Project

Excalidraw Ecosystem Rewrite — self-hosted whiteboard with Go Clean Architecture backend and React Vite frontend.

## Key State

- Planning covers all 5 phases (34 tickets total).
- Phase 1 has 15 tickets (EXC-001 to EXC-014 plus EXC-034).
- EXC-001 (Go Backend Scaffold), EXC-002 (Frontend Scaffold), EXC-005 (Shadcn UI) have some implementation evidence.
- EXC-006 through EXC-023 and EXC-034 are implemented; external credential rotation remains a release operation.
- Architecture, requirements, and ADR documentation migrated to harness format.

## Active Decisions

- Materialized Path for folder hierarchy (D-01).
- UUIDs for primary keys (D-02).
- Single active workspace per session (D-03).
- Context-based tenancy via middleware (D-04).
- Last-write-wins for scene persistence (D-06).
- Google OAuth is the only interactive login path; JWT middleware and authenticated API contracts remain stable.

## Blockers

- Deployment release is blocked until the Google client secret and any previously exposed JWT secret are rotated in the secret store; see [auth secret rotation runbook](operations/auth-secrets-rotation.md).

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
- 2026-10-15: Completed EXC-034 Google OAuth login/callback, frontend token handoff, and browser coverage; deployment credential rotation remains operational follow-up.
- 2026-10-15: Completed EXC-015 through EXC-018 frame persistence, presentation mode, slide navigator, and keyboard navigation; frontend and Playwright checks pass.
- 2026-10-15: Completed EXC-019 through EXC-023 public sharing, revoke, password challenge, share dialog, and read-only viewer; backend and Playwright checks pass.
