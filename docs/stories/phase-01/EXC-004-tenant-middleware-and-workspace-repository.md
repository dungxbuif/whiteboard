# EXC-004: Tenant Middleware & Workspace Repository

## Status

planned

## Lane

normal

## Product Contract

TenantMiddleware (extract X-Workspace-ID, validate UUID, inject context) and WorkspaceRepository (Create, GetByID)

## Relevant Product Docs

- `01-02-PLAN.md`

## Acceptance Criteria

- TenantMiddleware correctly extracts X-Workspace-ID from header and injects to context.
- WorkspaceRepository implements Create and GetByID using pgx pool.
- Middleware and Repository unit tests pass.

## Design Notes

A Detail Design document following `docs/templates/detail_design.md` must be created at `docs/stories/phase-01/EXC-004-design.md` and approved by the reviewer before implementation starts.

- **Link to Detail Design**: [Design Document](EXC-004-design.md)

- **Modified Files**: `internal/delivery/http/middleware/tenant.go`, `tenant_test.go`, `internal/repository/postgres/workspace_repo.go`, `workspace_repo_test.go`
- **Sprint Phase**: phase-01

## Validation

| Layer | Expected proof |
| --- | --- |
| Unit | go test -v ./internal/delivery/http/middleware/... ./internal/repository/postgres/... |
| Integration | |
| E2E | |
| Platform | |
| Release | |

## Harness Delta

- Registered EXC-004 in the harness database.

## Evidence

(No evidence yet)
