# EXC-020: Sharing Backend

## Status

done

## Lane

normal

## Product Contract

ScenePermissionRepository (GetBySceneID, Upsert), ScenePermissionUsecase, SharingHandler (PUT /api/scenes/{id}/share)

## Relevant Product Docs

- `03-01-PLAN.md`

## Acceptance Criteria

- ScenePermissionRepository correctly fetches and updates scene access settings.
- Sharing REST endpoint allows toggling sharing status with authentication.

## Design Notes

A Detail Design document following `docs/templates/detail_design.md` is recorded at `EXC-020-design.md` and approved for implementation.

- **Link to Detail Design**: [Design Document](EXC-020-design.md)

- **Modified Files**: `src/backend/internal/repository/postgres/scene_permission_repo.go`, `src/backend/internal/usecase/scene_permission_ucase.go`, `src/backend/internal/delivery/http/handler/sharing_handler.go`
- **Sprint Phase**: phase-03

## Validation

| Layer | Expected proof |
| --- | --- |
| Unit | go build ./src/backend/cmd/server |
| Integration | |
| E2E | |
| Platform | |
| Release | |

## Harness Delta

- Registered EXC-020 in the harness database.

## Evidence

- Authenticated share/revoke handling, public token lookup, and bcrypt password
  verification are implemented; `go test ./...` passes with usecase and HTTP
  handler coverage.
