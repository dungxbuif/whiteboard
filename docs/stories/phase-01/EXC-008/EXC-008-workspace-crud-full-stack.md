# EXC-008: Workspace CRUD Full Stack

## Status

planned

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

- **Link to Detail Design**: [Design Document](EXC-008-design.md)

- **Modified Files**: `internal/usecase/workspace_ucase.go`, `workspace_handler.go`, `router.go`, `dashboard/src/store/workspace-store.ts`, `dashboard/src/components/dashboard/workspace-switcher.tsx`
- **Sprint Phase**: phase-01

## Validation

| Layer | Expected proof |
| --- | --- |
| Unit | go test -v ./internal/usecase/... && cd dashboard && npm run build |
| Integration | |
| E2E | |
| Platform | |
| Release | |

## Harness Delta

- Registered EXC-008 in the harness database.

## Evidence

(No evidence yet)
