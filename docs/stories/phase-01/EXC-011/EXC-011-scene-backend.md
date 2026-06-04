# EXC-011: Scene Backend

## Status

planned

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

- **Modified Files**: `internal/repository/postgres/scene_repo.go`, `scene_repo_test.go`, `internal/usecase/scene_ucase.go`, `scene_ucase_test.go`, `internal/delivery/http/handler/scene_handler.go`, `router.go`
- **Sprint Phase**: phase-01

## Validation

| Layer | Expected proof |
| --- | --- |
| Unit | go test -v ./internal/usecase/... ./internal/repository/postgres/... |
| Integration | |
| E2E | |
| Platform | |
| Release | |

## Harness Delta

- Registered EXC-011 in the harness database.

## Evidence

(No evidence yet)
