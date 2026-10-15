---
artifact_type: detail_design
id: DESIGN-EXC-014
status: approved
owner: shared
approval: user-requested-project-completion
human_fields: [approval, constraints, scope_decisions]
ai_fields: [problem, context_loaded, brownfield_scope, proposed_approach, design_tradeoffs, architecture_overview, api_data_model, security, test_plan, reconciliation_plan]
shared_fields: [status, trace]
trace:
  backlog_item: EXC-014
  requirement: [REQ-07, REQ-08, REQ-09]
  phase: PHASE-1
  ticket_or_bug: EXC-014
  validation_matrix: docs/work/VALIDATION_MATRIX.md
  docs_review: pending
  adrs: []
  master_docs_touched:
    - docs/architecture/API.md
    - docs/architecture/ARCHITECTURE.md
    - docs/CONTEXT.md
    - docs/work/BACKLOG.md
    - docs/work/VALIDATION_MATRIX.md
    - docs/releases/CHANGELOG.md
---

# DETAIL DESIGN: EXC-014 Search + Starred/Trash

## Problem and outcome

The backend already stores starred and soft-deleted folders/scenes, but the dashboard only shows active folders and scenes. The search input is also presentational only. This slice makes those stored capabilities usable through one workspace-scoped search contract and three dashboard views: all files, starred items, and trash.

Acceptance is met when a user can type a query and receive matching active folders and scenes, open the Starred view, open the Trash view, and restore a deleted scene or folder. Main dashboard lists must continue to exclude deleted records.

## Context loaded

- `docs/CONTEXT.md`, `docs/work/BACKLOG.md`, `docs/standards/README.md`, `docs/standards/QUALITY_BAR.md`, `docs/standards/VALIDATION.md`
- `docs/work/phases/PHASE-1.md` and `docs/work/tickets/phase-01/EXC-014/EXC-014-search-starred-trash.md`
- `docs/requirements/SPEC.md`, `docs/architecture/API.md`, `docs/architecture/ARCHITECTURE.md`
- Existing scene/folder domain, PostgreSQL repositories, router, Zustand stores, and dashboard components.

## Scope and design

Add a `SearchRepository -> SearchUsecase -> SearchHandler` path. `GET /api/v1/search` accepts `q` and `view` (`all`, `starred`, or `trash`) and returns a bounded, unified list of folder and scene result records. Every query is filtered by the tenant workspace. Active search excludes soft-deleted rows; starred excludes deleted rows; trash returns only deleted rows.

The frontend adds a debounced `SearchBar`, a search result view, and sidebar navigation for All files, Starred, and Trash. Existing scene star/trash actions remain the source of mutations; folder star/trash/restore actions are exposed in the folder store for the same views. Search results are refreshed after a restore or mode change.

The star mutation responds with the updated scene so the frontend can update its store without a second fetch. This removes the current mismatch where the client expects JSON from a `204 No Content` response.

## Alternatives and trade-offs

| Option | Decision | Reason |
| --- | --- | --- |
| Separate endpoints for search, starred, and trash | Rejected | Duplicates workspace filtering and result mapping. |
| One unified search endpoint with a view filter | Chosen | Keeps the UI contract small and supports folders and scenes consistently. |
| Client-side filtering after loading every record | Rejected | Leaks deleted data to the client and does not scale with workspace size. |
| Full-text search infrastructure now | Rejected | Existing `ILIKE` plus trigram indexes satisfy Phase 1; ranking can be added later. |

## Security and data behavior

- Search never accepts a workspace identifier from the query string; tenant middleware supplies it.
- Trash is intentionally unscoped at the repository layer but remains workspace-filtered and only returns rows with `deleted_at IS NOT NULL`.
- Result limits and query length are enforced by named constants.
- Folder and scene IDs are returned as opaque UUIDs; no scene content is returned in search results.

## Verification plan

- Go unit tests for search usecase validation and result view selection.
- Go handler/router tests for query parsing and workspace-scoped search route.
- Frontend unit tests for debounce, view changes, and restore behavior.
- `go test ./...` and `npm run test:unit && npm run build`.
- Browser verification is required for the dashboard search/sidebar flow after a runnable backend is available.

## Reconciliation plan

Update the API index, architecture behavior notes, backlog/ticket status, validation matrix, context, and unreleased changelog after implementation. No new ADR is needed because the design uses the existing workspace-scoped soft-delete and `ILIKE` contracts.
