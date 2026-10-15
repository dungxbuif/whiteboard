# EXC-014: Search + Starred/Trash

## Status

done

## Lane

normal

## Product Contract

SearchRepository (ILIKE query), SearchUsecase, SearchHandler + route, SearchBar component, Starred sidebar, Trash sidebar

## Relevant Product Docs

- `01-11-PLAN.md`

## Acceptance Criteria

- Search API allows searching across scenes and folders using ILIKE query.
- SearchBar component debounces typing and updates search results.
- Starred and Trash sidebars display starred/deleted items with restore options.

## Design Notes

A Detail Design document following `docs/templates/detail_design.md` must be created at `docs/stories/phase-01/EXC-014-design.md` and approved by the reviewer before implementation starts.

- **Link to Detail Design**: [Design Document](EXC-014-design.md)

- **Modified Files**: `src/backend/internal/domain/search.go`, `src/backend/internal/repository/postgres/search_repo.go`, `src/backend/internal/usecase/search_ucase.go`, `search_ucase_test.go`, `search_handler.go`, `router.go`, `src/frontend/src/store/search-store.ts`, `search-bar.tsx`, `search-results.tsx`, `folder-actions.tsx`, `folder-tree.tsx`, `folder-list.tsx`, `scene-store.ts`, `folder-store.ts`, `App.tsx`
- **Sprint Phase**: phase-01

## Validation

| Layer | Expected proof |
| --- | --- |
| Unit | `go test ./...` and `npm run test:unit` |
| Integration | |
| E2E | |
| Platform | |
| Release | |

## Harness Delta

- Registered EXC-014 in the harness database.

## Evidence

- `go test ./...` passed after adding the tenant-scoped search repository/usecase/handler and updating the router tests.
- `npm run test:unit` passed with 16 tests.
- `npm run build` passed with TypeScript compilation and Vite production output.
- `npx playwright test` passed (11 browser flows, including folder navigation,
  scene lifecycle, Starred/Trash controls, and mobile canvas layout).
