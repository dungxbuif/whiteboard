# EXC-003: DB Migrations & Multi-tenant Schema

## Status

planned

## Lane

normal

## Product Contract

SQL migration for users, workspaces, folders, scenes tables with UUID PKs, workspace_id indexes, is_starred, deleted_at columns

## Relevant Product Docs

- `01-02-PLAN.md`

## Acceptance Criteria

- SQL migration script creates the necessary tables with tenant routing.
- Domain entities (Folder, Scene) are updated with appropriate fields (is_starred, deleted_at, etc.).

## Design Notes

A Detail Design document following `docs/templates/detail_design.md` must be created at `docs/stories/phase-01/EXC-003-design.md` and approved by the reviewer before implementation starts.

- **Link to Detail Design**: [Design Document](EXC-003-design.md)

- **Modified Files**: `migrations/000001_init.up.sql`, `internal/domain/folder.go`, `internal/domain/scene.go`
- **Sprint Phase**: phase-01

## Validation

| Layer | Expected proof |
| --- | --- |
| Unit | grep -i "is_starred" migrations/000001_init.up.sql && grep -i "deleted_at" migrations/000001_init.up.sql |
| Integration | |
| E2E | |
| Platform | |
| Release | |

## Harness Delta

- Registered EXC-003 in the harness database.

## Evidence

(No evidence yet)
