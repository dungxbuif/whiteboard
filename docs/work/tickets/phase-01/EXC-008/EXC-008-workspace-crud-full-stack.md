# EXC-008: Workspace CRUD Full Stack

## Status

completed

## Lane

normal

## Product Contract

WorkspaceUsecase (ListByUser, Create), WorkspaceHandler, route registration, Zustand workspace store, WorkspaceSwitcher component

## Relevant Product Docs

- `01-05-PLAN.md`

## Acceptance Criteria

- Workspace REST endpoints (GET/POST /workspaces) are functional.
- Zustand workspace store manages active workspace and user workspaces.
- WorkspaceSwitcher UI component allows user to switch between workspaces.

## Design Notes

A Detail Design document following `docs/templates/detail_design.md` must be created at `docs/stories/phase-01/EXC-008-design.md` and approved by the reviewer before implementation starts.

- **Link to Detail Design**: [Design Document](../EXC-008-design.md)

- **Modified Files**: `src/backend/internal/usecase/workspace_ucase.go`, `src/backend/internal/delivery/http/handler/workspace_handler.go`, `src/backend/internal/delivery/http/router.go`, `src/frontend/src/store/workspace-store.ts`, `src/frontend/src/components/workspace-switcher.tsx`, `src/frontend/src/App.tsx`
- **Sprint Phase**: phase-01

## Validation

| Layer | Expected proof | Result |
| --- | --- | --- |
| Unit | go test -v ./src/backend/internal/usecase/... && cd src/frontend && npm run build | PASS |
| Integration | go test -v ./src/backend/internal/repository/postgres/... | PASS |
| E2E | | |
| Platform | | |
| Release | | |

## Harness Delta

- Registered EXC-008 in the harness database.
- Implemented full-stack Workspace CRUD with Zustand and shadcn/ui.

## Evidence

Backend tests passed:
```text
ok      devtool/internal/usecase        0.609s
ok      devtool/internal/repository/postgres    0.699s
```

Frontend build passed:
```text
vite v7.3.5 building for production...
✓ built in 1.28s
```

