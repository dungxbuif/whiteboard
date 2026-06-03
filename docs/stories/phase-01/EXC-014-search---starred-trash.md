# EXC-014: Search + Starred/Trash

## Status

planned

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

- **Modified Files**: `internal/repository/postgres/search_repo.go`, `search_repo_test.go`, `internal/usecase/search_ucase.go`, `search_ucase_test.go`, `search_handler.go`, `router.go`, `dashboard/src/components/dashboard/search-bar.tsx`, `sidebar-starred.tsx`, `sidebar-trash.tsx`
- **Sprint Phase**: phase-01

## Validation

| Layer | Expected proof |
| --- | --- |
| Unit | go test -v ./internal/... && cd dashboard && npm run build |
| Integration | |
| E2E | |
| Platform | |
| Release | |

## Harness Delta

- Registered EXC-014 in the harness database.

## Evidence

(No evidence yet)
