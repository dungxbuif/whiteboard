---
artifact_type: context
id: CONTEXT
status: active
owner: shared
updated: 2026-06-11
---

# Context

## Current Focus

- **Active Phase:** PHASE-1 (Self-hosted Dashboard & Core Infrastructure)
- **Active Work:** Documentation migration to harness framework complete
- **Next Work:** Resume PHASE-1 ticket execution; EXC-034 restores Google OAuth near the end of the phase.

## Project

Excalidraw Ecosystem Rewrite — self-hosted whiteboard with Go Clean Architecture backend and React Vite frontend.

## Key State

- Planning covers all 5 phases (34 tickets total).
- Phase 1 has 15 tickets (EXC-001 to EXC-014 plus EXC-034).
- EXC-001 (Go Backend Scaffold), EXC-002 (Frontend Scaffold), EXC-005 (Shadcn UI) have some implementation evidence.
- All other tickets remain in `planned` status.
- Architecture, requirements, and ADR documentation migrated to harness format.

## Active Decisions

- Materialized Path for folder hierarchy (D-01).
- UUIDs for primary keys (D-02).
- Single active workspace per session (D-03).
- Context-based tenancy via middleware (D-04).
- Last-write-wins for scene persistence (D-06).
- Temporary development authentication uses the single hard-coded `admin/admin` account; EXC-034 replaces it with Google OAuth before release.

## Blockers

None currently.

## Recent Changes

- 2026-06-11: Migrated all documentation from old devtool/docs format to harness framework structure.
- 2026-06-05: Implemented optimized Materialized Path folder backend (EXC-009).
- 2026-06-04: Implemented Workspace CRUD (EXC-008) and auth/tenant (EXC-006, EXC-004).
