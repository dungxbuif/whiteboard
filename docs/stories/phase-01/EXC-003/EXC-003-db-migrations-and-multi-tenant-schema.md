# EXC-003: DB Migrations & Multi-tenant Schema

## Status

implemented

## Lane

normal

## Product Contract

SQL migration for users, auth_sessions, workspaces, workspace_members, folders, scenes, and scene_assets tables with UUID PKs, workspace_id indexes, is_starred, deleted_at columns, and tenant-scoped asset metadata.

## Relevant Product Docs

- `01-02-PLAN.md`

## Acceptance Criteria

- SQL migration script creates the necessary tables with tenant routing.
- Domain entities (Folder, Scene) are updated with appropriate fields (is_starred, deleted_at, etc.).
- Asset metadata, if implemented in this story, is tenant-scoped and metadata-only: no raw bytes or base64 `dataURL` in PostgreSQL.

## Design Notes

A Detail Design document following `docs/templates/detail_design.md` must be created at `docs/stories/phase-01/EXC-003-design.md` and approved by the reviewer before implementation starts.

- **Link to Detail Design**: [Design Document](EXC-003-design.md)

- **Modified Files**: `migrations/000001_init.up.sql`, `migrations/000001_init.down.sql`, `src/backend/internal/domain/user.go`, `src/backend/internal/domain/workspace.go`, `src/backend/internal/domain/folder.go`, `src/backend/internal/domain/scene.go`
- **Sprint Phase**: phase-01

## Implementation Notes

- Treat [Phase 1 Database Design](../../../product/phase-01-database-design.md) as the source for asset columns and constraints.
- Do not copy the legacy `files` table shape from `references/server`; it lacks `workspace_id`, `scene_id`, asset kind, and object-key namespacing.
- Store binary asset bytes only in S3-compatible storage. PostgreSQL should store asset metadata such as `asset_kind`, Excalidraw `file_id`, bucket/key, MIME type, size, hash, and dimensions.
- Use namespaced object keys such as `workspaces/{workspace_id}/scenes/{scene_id}/assets/{asset_id}`.

## Validation

| Layer | Expected proof |
| --- | --- |
| Unit | grep -i "is_starred" migrations/000001_init.up.sql && grep -i "deleted_at" migrations/000001_init.up.sql |
| Unit | If asset schema is included: grep -i "scene_assets" migrations/000001_init.up.sql && grep -i "asset_kind" migrations/000001_init.up.sql |
| Integration | |
| E2E | |
| Platform | |
| Release | |

## Harness Delta

- Registered EXC-003 in the harness database.

## Evidence

- Added `migrations/000001_init.up.sql` and `migrations/000001_init.down.sql`.
- Added tenant-aware tables for identity, sessions, workspaces, workspace membership, folders, scenes, and scene assets.
- Updated Go domain structs for user OAuth fields, workspace soft delete, folder star/trash/path depth, scene JSONB/version/lifecycle metadata, and scene asset metadata.
- Verification run:
  - `rg -n "is_starred" migrations/000001_init.up.sql`
  - `rg -n "deleted_at" migrations/000001_init.up.sql`
  - `rg -n "scene_assets" migrations/000001_init.up.sql`
  - `rg -n "asset_kind" migrations/000001_init.up.sql`
  - `rg -n "data_url|dataURL|bytea| raw asset|base64" migrations/000001_init.up.sql src/backend/internal/domain` returned no matches.
  - `go test ./...` from `src/backend` passed.
