# EXC-019: Sharing DB Schema & Domain

## Status

done

## Lane

normal

## Product Contract

Migration: scene_permissions table (id, value, active, timestamps), ScenePermission domain struct + interfaces

## Relevant Product Docs

- `03-01-PLAN.md`

## Acceptance Criteria

- Database migration for scene_permissions compiles and runs.
- Domain definition for ScenePermission matches the schema.

## Design Notes

A Detail Design document following `docs/templates/detail_design.md` is recorded at `EXC-019-design.md` and approved for implementation.

- **Link to Detail Design**: [Design Document](EXC-019-design.md)

- **Modified Files**: `src/backend/migrations/000004_create_scene_permissions.up.sql`, `src/backend/internal/domain/scene_permission.go`
- **Sprint Phase**: phase-03

## Validation

| Layer | Expected proof |
| --- | --- |
| Unit | grep "ScenePermission struct" src/backend/internal/domain/scene_permission.go |
| Integration | |
| E2E | |
| Platform | |
| Release | |

## Harness Delta

- Registered EXC-019 in the harness database.

## Evidence

- Migration `000005_create_scene_permissions` and the ScenePermission domain
  repository contract are implemented; `go test ./...` passes.
