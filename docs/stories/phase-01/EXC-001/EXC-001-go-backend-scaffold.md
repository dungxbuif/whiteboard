# EXC-001: Go Backend Scaffold

## Status

implemented

## Lane

normal

## Product Contract

Init Go module, Clean Architecture dirs, domain entities, chi router, and direct local PostgreSQL configuration.

## Relevant Product Docs

- `01-01-PLAN.md`

## Acceptance Criteria

- Clean Architecture directory structure is established.
- Go module 'devtool' is initialized.
- cmd/server/main.go compiles and runs directly, starting a basic HTTP server.
- .env.example is present with direct local PostgreSQL credentials.
- Local PostgreSQL defaults use `POSTGRES_USER=dev`, `POSTGRES_PASSWORD=password`, and app database `POSTGRES_DB=devtool`.

## Design Notes

A Detail Design document following `docs/templates/detail_design.md` must be created at `docs/stories/phase-01/EXC-001-design.md` and approved by the reviewer before implementation starts.

- **Link to Detail Design**: [Design Document](EXC-001-design.md)

- **Modified Files**: `go.mod`, `cmd/server/main.go`, `internal/domain/user.go`, `internal/domain/workspace.go`, `internal/domain/folder.go`, `internal/domain/scene.go`, `internal/delivery/http/router.go`, `.env.example`
- **Sprint Phase**: phase-01

## Validation

| Layer | Expected proof |
| --- | --- |
| Unit | go mod verify && go build ./cmd/server |
| Integration | Direct local PostgreSQL smoke via `go run ./cmd/server` when local service is available |
| E2E | |
| Platform | |
| Release | |

## Harness Delta

- Registered EXC-001 in the harness database.

## Evidence

- 2026-06-04: `go mod verify` passed.
- 2026-06-04: `GOCACHE=/private/tmp/devtool-go-cache go build -o /private/tmp/devtool-server ./cmd/server` passed.
- 2026-06-04: Removed Docker Compose path per local environment; backend now defaults to direct local PostgreSQL at `postgres://test:test@127.0.0.1:5432/whiteboard_test?sslmode=disable`.
- 2026-06-04: Direct smoke test `GOCACHE=/private/tmp/devtool-go-cache HTTP_ADDR=:18080 go run ./cmd/server` initialized the local PostgreSQL connection and `GET http://127.0.0.1:18080/healthz` returned `{"status":"ok"}`.
