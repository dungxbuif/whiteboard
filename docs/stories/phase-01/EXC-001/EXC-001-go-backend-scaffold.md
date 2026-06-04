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
- src/backend/cmd/server/main.go compiles and runs directly, starting a basic HTTP server.
- .env.example is present with direct local PostgreSQL credentials.
- Local PostgreSQL defaults use `POSTGRES_USER=dev`, `POSTGRES_PASSWORD=password`, and app database `POSTGRES_DB=devtool`.

## Design Notes

A Detail Design document following `docs/templates/detail_design.md` must be created at `docs/stories/phase-01/EXC-001-design.md` and approved by the reviewer before implementation starts.

- **Link to Detail Design**: [Design Document](EXC-001-design.md)

- **Modified Files**: `src/backend/go.mod`, `src/backend/cmd/server/main.go`, `src/backend/internal/domain/user.go`, `src/backend/internal/domain/workspace.go`, `src/backend/internal/domain/folder.go`, `src/backend/internal/domain/scene.go`, `src/backend/internal/delivery/http/router.go`, `.env.example`
- **Sprint Phase**: phase-01

## Validation

| Layer | Expected proof |
| --- | --- |
| Unit | go mod verify && go build ./src/backend/cmd/server |
| Integration | Direct local PostgreSQL smoke via `go run ./src/backend/cmd/server` when local service is available |
| E2E | |
| Platform | |
| Release | |

## Harness Delta

- Registered EXC-001 in the harness database.

## Evidence

- 2026-06-04: `go mod verify` passed.
- 2026-06-04: `GOCACHE=/private/tmp/devtool-go-cache go build -o /private/tmp/devtool-server ./src/backend/cmd/server` passed.
- 2026-06-04: Removed Docker Compose path per local environment; backend now defaults to direct local PostgreSQL at `postgres://test:test@127.0.0.1:5432/whiteboard_test?sslmode=disable`.
- 2026-06-04: Direct smoke test `GOCACHE=/private/tmp/devtool-go-cache HTTP_ADDR=:18080 go run ./src/backend/cmd/server` initialized the local PostgreSQL connection and `GET http://127.0.0.1:18080/healthz` returned `{"status":"ok"}`.
