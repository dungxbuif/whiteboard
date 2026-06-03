# EXC-001: Go Backend Scaffold

## Status

planned

## Lane

normal

## Product Contract

Init Go module, Clean Architecture dirs, domain entities, chi router, Docker Compose (Postgres + MinIO)

## Relevant Product Docs

- `01-01-PLAN.md`

## Acceptance Criteria

- Clean Architecture directory structure is established.
- Go module 'devtool' is initialized.
- cmd/server/main.go compiles and runs, starting a basic HTTP server.
- docker-compose.yml defines postgres and minio services.
- .env.example is present with dev credentials.

## Design Notes

A Detail Design document following `docs/templates/detail_design.md` must be created at `docs/stories/phase-01/EXC-001-design.md` and approved by the reviewer before implementation starts.

- **Link to Detail Design**: [Design Document](EXC-001-design.md)

- **Modified Files**: `go.mod`, `cmd/server/main.go`, `internal/domain/user.go`, `internal/domain/workspace.go`, `internal/domain/folder.go`, `internal/domain/scene.go`, `internal/delivery/http/router.go`, `docker-compose.yml`, `.env.example`
- **Sprint Phase**: phase-01

## Validation

| Layer | Expected proof |
| --- | --- |
| Unit | go mod verify && go build ./cmd/server |
| Integration | |
| E2E | |
| Platform | |
| Release | |

## Harness Delta

- Registered EXC-001 in the harness database.

## Evidence

(No evidence yet)
