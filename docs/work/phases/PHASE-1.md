---
artifact_type: phase
id: PHASE-1
status: in_progress
owner: human
human_fields:
  - goal
  - scope
  - out_of_scope
  - priority
  - success_criteria
ai_fields:
  - risks
  - dependencies
  - verification_plan
  - completion_summary
shared_fields:
  - status
  - trace
  - tickets_and_bugs
trace:
  backlog_items: []
  roadmap: ROADMAP
  requirements: [REQ-01, REQ-02, REQ-03, REQ-04, REQ-05, REQ-06, REQ-07, REQ-08, REQ-09, REQ-10]
  tickets: [EXC-001, EXC-002, EXC-003, EXC-004, EXC-005, EXC-006, EXC-007, EXC-008, EXC-009, EXC-010, EXC-011, EXC-012, EXC-013, EXC-014, EXC-034]
  bugs: []
  test_verification: TBD
  validation_matrix: VALIDATION_MATRIX
  adrs: [0001, 0002, 0003, 0004, 0005]
  release_notes: TBD
---

# Phase: PHASE-1 Self-hosted Dashboard & Core Infrastructure

## Field Ownership

- Human fills goal, scope, out of scope, priority, and success criteria.
- AI fills risks, dependencies, verification plan, and completion summary.
- Shared fields include status, trace links, and ticket/bug list.

## Status

- ID: PHASE-1
- Status: in_progress
- Owner: human
- Created: 2026-06-03
- Updated: 2026-06-11

## Trace Links

- Backlog items: EXC-001 through EXC-014
- Roadmap: [ROADMAP.md](../ROADMAP.md)
- Requirements: [SPEC.md](../../requirements/SPEC.md) (REQ-01 through REQ-10)
- Tickets: See table below
- Bugs: None
- Test verification: TBD
- Validation matrix: [VALIDATION_MATRIX.md](../VALIDATION_MATRIX.md)
- ADRs: [0001](../../decisions/0001-harness-first-development.md), [0002](../../decisions/0002-post-spec-product-lifecycle.md), [0003](../../decisions/0003-generic-spec-intake-harness.md), [0004](../../decisions/0004-sqlite-durable-layer.md), [0005](../../decisions/0005-prebuilt-rust-harness-cli.md)
- Release notes: TBD

## Goal

Replace NestJS/Next.js stack with Go/React Vite. Core rewrite of Dashboard and Backend into a SaaS-ready single-product system with explicit workspace tenancy, PostgreSQL metadata, and S3-compatible binary asset storage.

## Scope

- Full rewrite of Frontend in React Vite + Shadcn (reusing UI components).
- Full rewrite of Backend in Go (Clean Architecture).
- Google OAuth authorization-code login with backend-issued JWT sessions.
- Multi-tenant database design.
- Folder/File CRUD with Trash, Star, and Search.
- S3 integration for media.
- Materialized Path for folder hierarchies.
- Multi-tenant isolation at DB and Middleware levels.
- Debounced auto-sync for drawings.

## Out Of Scope

- Multi-user collaboration (Phase 4).
- AI features (Phase 5).
- Complex permission systems (e.g., inviting other users to workspaces).
- Public sharing links (Phase 3).

## Implementation Decisions

- **D-01: Application-Level Materialized Path:** Folder hierarchies managed using materialized path string. Path updates handled by Go application logic.
- **D-02: UUID for Primary Keys:** Use UUIDs for all public-facing and internal entity IDs.
- **D-03: Single Active Workspace per Session:** One active workspace per user session. Multi-tab switching NOT required for Phase 1.
- **D-04: Context-based Tenancy:** `WorkspaceID` extracted at middleware level and passed through Go `context.Context`.
- **D-05: Zustand for Dashboard:** Pure Zustand for managing folder tree, file lists, and UI state.
- **D-06: Last-Write-Wins (LWW):** Simple conflict resolution for whiteboard sync.
- **D-07: Debounced Auto-sync:** Whiteboard data and S3 assets sync after a short delay of inactivity.
- **D-08: Go Clean Architecture Pattern:** Strictly follow `domain`, `usecase`, `repository`, `delivery` separation.

## Tickets And Bugs

| ID | Type | Title | Status | Link |
| --- | --- | --- | --- | --- |
| EXC-001 | ticket | Go Backend Scaffold | done | [EXC-001](../tickets/phase-01/EXC-001/EXC-001-go-backend-scaffold.md) |
| EXC-002 | ticket | Frontend Scaffold Vite+React | done | [EXC-002](../tickets/phase-01/EXC-002/EXC-002-frontend-scaffold-vite-react.md) |
| EXC-003 | ticket | DB Migrations & Multi-tenant Schema | done | [EXC-003](../tickets/phase-01/EXC-003/EXC-003-db-migrations-and-multi-tenant-schema.md) |
| EXC-004 | ticket | Tenant Middleware & Workspace Repository | done | [EXC-004](../tickets/phase-01/EXC-004/EXC-004-tenant-middleware-and-workspace-repository.md) |
| EXC-005 | ticket | Shadcn UI & Tailwind Setup | done | [EXC-005](../tickets/phase-01/EXC-005/EXC-005-shadcn-ui-and-tailwind-setup.md) |
| EXC-006 | ticket | Google OAuth Backend | done | [EXC-006](../tickets/phase-01/EXC-006/EXC-006-google-oauth-backend.md) |
| EXC-007 | ticket | Frontend Auth Store | done | [EXC-007](../tickets/phase-01/EXC-007/EXC-007-frontend-auth-store.md) |
| EXC-008 | ticket | Workspace CRUD Full Stack | done | [EXC-008](../tickets/phase-01/EXC-008/EXC-008-workspace-crud-full-stack.md) |
| EXC-009 | ticket | Folder Backend Materialized Path | completed | [EXC-009](../tickets/phase-01/EXC-009/EXC-009-folder-backend-materialized-path.md) |
| EXC-010 | ticket | Folder Frontend Tree+Nav | completed | [EXC-010](../tickets/phase-01/EXC-010/EXC-010-folder-frontend-tree-nav.md) |
| EXC-011 | ticket | Scene Backend | completed | [EXC-011](../tickets/phase-01/EXC-011/EXC-011-scene-backend.md) |
| EXC-012 | ticket | Excalidraw Integration | done | [EXC-012](../tickets/phase-01/EXC-012/EXC-012-excalidraw-integration.md) |
| EXC-013 | ticket | S3 Assets | implemented | [EXC-013](../tickets/phase-01/EXC-013/EXC-013-s3-assets.md) |
| EXC-014 | ticket | Search + Starred/Trash | done | [EXC-014](../tickets/phase-01/EXC-014/EXC-014-search-starred-trash.md) |
| EXC-034 | ticket | Replace Temporary Admin Auth With Google OAuth | done | [EXC-034](../tickets/phase-01/EXC-034/EXC-034-replace-temporary-admin-auth-with-google-oauth.md) |

## Dependencies

- PostgreSQL 16
- MinIO / S3-compatible storage
- Google OAuth credentials
- Go 1.22+, Node.js 18+

## Risks

- Materialized path updates may be slow for deep hierarchies.
- Last-write-wins may cause data loss in edge cases.
- S3 signed URL expiration needs UX tuning.

## Success Criteria

- [ ] Backend code passes "Clean Architecture" review.
- [ ] Database schema verified for multi-tenancy support.
- [x] User can log in through Google OAuth and receive a JWT session.
- [x] Google OAuth replaces temporary admin auth in runtime code and browser flow.
- [ ] Deployment OAuth/JWT credentials are rotated in the secret store before release.
- [ ] User can create multiple Workspaces.
- [ ] User can create, rename, and move drawings into nested folders.
- [ ] User can "Star" a drawing and see it in the Starred section.
- [ ] User can move a drawing to Trash and restore it.
- [ ] Searching for a file name returns correct results.
- [ ] Drawing state (JSON) is saved to DB and assets to S3.
- [x] Excalidraw UI is fully functional within the React app.

## Verification Plan

- Unit tests for domain and usecase layers.
- Integration tests for repository layer (workspace isolation).
- E2E tests for critical user flows (login, create workspace, CRUD folders/scenes).
- Manual verification of Excalidraw embedding.

## Gate Checklist

- [x] Phase has linked requirements or explicit discovery goal
- [x] Tickets/bugs are created or planned
- [x] Risks and dependencies are recorded
- [x] Verification plan is defined
- [ ] Release/changelog need is identified

## Completion Summary

Complete this section when the phase is done.
