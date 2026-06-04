# EXC-004: Tenant Middleware & Workspace Repository

## Status

completed

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

- **Modified Files**: `src/backend/internal/delivery/http/middleware/tenant.go`, `src/backend/internal/delivery/http/middleware/tenant_test.go`, `src/backend/internal/repository/postgres/workspace_repo.go`, `src/backend/internal/repository/postgres/workspace_repo_test.go`
- **Sprint Phase**: phase-01

## Validation

| Layer | Expected proof | Result |
| --- | --- | --- |
| Unit | go test -v ./src/backend/internal/delivery/http/middleware/... ./src/backend/internal/repository/postgres/... | PASS |
| Integration | go test -v ./src/backend/internal/repository/postgres/... (with DB) | PASS |
| E2E | | |
| Platform | | |
| Release | | |

## Harness Delta

- Registered EXC-004 in the harness database.
- Implemented TenantMiddleware and WorkspaceRepository.
- Verified with unit and integration tests.

## Evidence

Tests passed locally:
```text
ok      devtool/internal/delivery/http/middleware       0.561s
ok      devtool/internal/repository/postgres    0.627s
```

