# EXC-006: Google OAuth Backend

## Status

planned

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

- **Link to Detail Design**: [Design Document](EXC-006-design.md)

- **Modified Files**: `internal/usecase/auth_ucase.go`, `auth_ucase_test.go`, `internal/delivery/http/handler/auth_handler.go`, `internal/delivery/http/middleware/auth.go`, `router.go`
- **Sprint Phase**: phase-01

## Validation

| Layer | Expected proof |
| --- | --- |
| Unit | go test -v ./internal/delivery/http/middleware/... |
| Integration | |
| E2E | |
| Platform | |
| Release | |

## Harness Delta

- Registered EXC-006 in the harness database.

## Evidence

(No evidence yet)
