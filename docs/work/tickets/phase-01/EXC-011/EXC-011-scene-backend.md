# EXC-011: Scene Backend

## Status

completed

## Lane

normal

## Product Contract

SceneRepository (Upsert JSONB, List, SoftDelete, Restore, SetStar), SceneUsecase, SceneHandler + routes

## Relevant Product Docs

- `01-08-PLAN.md`

## Acceptance Criteria

- SceneRepository handles JSONB storage for Excalidraw element data.
- SceneUsecase handles upsert logic, folder mapping, and soft delete.
- Scene HTTP handlers are registered on chi router and tested.

## Design Notes

A Detail Design document following `docs/templates/detail_design.md` must be created at `docs/stories/phase-01/EXC-011-design.md` and approved by the reviewer before implementation starts.

- **Link to Detail Design**: [Design Document](EXC-011-design.md)

- **Modified Files**: `src/backend/internal/repository/postgres/scene_repo.go`, `scene_repo_test.go`, `src/backend/internal/usecase/scene_ucase.go`, `scene_ucase_test.go`, `src/backend/internal/delivery/http/handler/scene_handler.go`, `router.go`
- **Sprint Phase**: phase-01

## Validation

| Layer | Expected proof | Result |
| --- | --- | --- |
| Unit | `cd src/backend && go test ./...` | PASS (32 tests) |
| Integration | PostgreSQL JSONB, version, tenant, star, trash, restore, folder cascade | PASS |
| E2E | Router-level tenant and camelCase contract tests | PASS |
| Platform | |
| Release | |

## Harness Delta

- Registered EXC-011 in the harness database.

## Evidence

- Added workspace-scoped scene create/list/get/update/star/trash/restore repository methods.
- Scene content persists as JSONB and increments `content_version` on update.
- Folder trash/restore cascades to scenes in the folder subtree in one transaction.
- Registered authenticated, tenant-scoped `/api/v1/scenes` routes.
