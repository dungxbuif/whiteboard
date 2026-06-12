# EXC-009: Folder Backend Materialized Path

## Status

completed

## Lane

normal

## Product Contract

FolderRepository (CRUD, UpdatePath, SoftDelete, Restore, SetStar), FolderUsecase (Create with path calc, Move, lifecycle), FolderHandler

## Relevant Product Docs

- `01-06-PLAN.md`

## Acceptance Criteria

- FolderRepository implements Materialized Path logic using 32-character stripped UUIDs for efficiency.
- FolderUsecase calculates correct path for child folders on Create and Move.
- FolderUsecase enforces a maximum depth of 30 levels.
- SoftDelete recursively marks children as deleted; Restore does the opposite.
- Endpoints are registered and tested.

## Design Notes

A Detail Design document following `docs/templates/detail_design.md` must be created at `docs/stories/phase-01/EXC-009-design.md` and approved by the reviewer before implementation starts.

- **Link to Detail Design**: [Design Document](../EXC-009-design.md)

- **Modified Files**: `src/backend/internal/repository/postgres/folder_repo.go`, `src/backend/internal/repository/postgres/folder_repo_test.go`, `src/backend/internal/usecase/folder_ucase.go`, `src/backend/internal/usecase/folder_ucase_test.go`, `src/backend/internal/delivery/http/handler/folder_handler.go`, `src/backend/internal/delivery/http/router.go`
- **Sprint Phase**: phase-01

## Validation

| Layer | Expected proof | Result |
| --- | --- | --- |
| Unit | `cd src/backend && go test ./...` | PASS (24 tests) |
| Integration | PostgreSQL repository tests and migration up/down on a temporary database | PASS |
| E2E | Folder route tenant-header regression test | PASS |
| Platform | |
| Release | |

## Harness Delta

- Registered EXC-009 in the harness database.
- Implemented Optimized Materialized Path strategy (stripped UUIDs).
- Codified 30-level depth constraint for index safety.

## Evidence

- Added tenant middleware to all folder routes.
- Added recursive restore and subtree-aware depth validation.
- Verified migration up/down ordering on temporary database `devtool_migration_audit`.
- Verified PostgreSQL repository integration tests against the local development database.
