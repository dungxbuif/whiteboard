# EXC-006: Google OAuth Backend

## Status

completed

## Lane

normal

## Product Contract

AuthUsecase (Google OAuth exchange), AuthHandler (login/callback routes), AuthMiddleware (JWT validation, context injection)

## Relevant Product Docs

- `01-04-PLAN.md`

## Acceptance Criteria

- AuthUsecase exchanges Google OAuth token for user session.
- AuthHandler implements login and callback endpoints.
- AuthMiddleware validates JWT token and injects User to context.
- Auth-related routes are registered on chi router.

## Design Notes

A Detail Design document following `docs/templates/detail_design.md` must be created at `docs/stories/phase-01/EXC-006-design.md` and approved by the reviewer before implementation starts.

- **Link to Detail Design**: [Design Document](../EXC-006-design.md)

- **Modified Files**: `src/backend/internal/usecase/auth_ucase.go`, `src/backend/internal/delivery/http/handler/auth_handler.go`, `src/backend/internal/delivery/http/middleware/auth.go`, `src/backend/internal/delivery/http/middleware/auth_test.go`, `src/backend/internal/repository/postgres/user_repo.go`, `src/backend/internal/repository/postgres/user_repo_test.go`, `src/backend/internal/delivery/http/router.go`, `src/backend/cmd/server/main.go`
- **Sprint Phase**: phase-01

## Validation

| Layer | Expected proof | Result |
| --- | --- | --- |
| Unit | go test -v ./src/backend/internal/delivery/http/middleware/... | PASS |
| Integration | go test -v ./src/backend/internal/repository/postgres/... | PASS |
| E2E | | |
| Platform | | |
| Release | | |

## Harness Delta

- Registered EXC-006 in the harness database.
- Implemented Google OAuth backend flow and JWT middleware.
- Verified with unit and repository tests.

## Evidence

Tests passed:
```text
ok      devtool/internal/delivery/http/middleware       0.481s
ok      devtool/internal/repository/postgres    0.542s
```

## Current Runtime Note

The Google OAuth implementation is the active runtime login path. EXC-034
removed the temporary single-admin development login and added state protection,
Google profile upsert, and browser verification.
