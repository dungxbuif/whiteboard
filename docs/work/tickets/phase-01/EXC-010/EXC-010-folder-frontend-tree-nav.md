# EXC-010: Folder Frontend Tree+Nav

## Status

completed

## Lane

normal

## Product Contract

Zustand folder store (flat to tree, currentFolderID), FolderTree sidebar, Breadcrumbs, dashboard-page layout, file-list component

## Relevant Product Docs

- `01-07-PLAN.md`

## Acceptance Criteria

- Zustand folder store builds tree structure from flat API array.
- Sidebar tree component allows folder expand/collapse and selection.
- Breadcrumbs display path to current folder and allow navigation.
- File list renders contents of the selected folder.

## Design Notes

A Detail Design document following `docs/templates/detail_design.md` must be created at `docs/stories/phase-01/EXC-010-design.md` and approved by the reviewer before implementation starts.

- **Link to Detail Design**: [Design Document](EXC-010-design.md)

- **Modified Files**: `src/frontend/src/store/folder-store.ts`, `src/frontend/src/components/src/frontend/folder-tree.tsx`, `breadcrumbs.tsx`, `file-list.tsx`, `src/frontend/src/app/dashboard-page.tsx`
- **Sprint Phase**: phase-01

## Validation

| Layer | Expected proof | Result |
| --- | --- | --- |
| Unit | `cd src/frontend && npm run test:unit` | PASS (11 tests) |
| Integration | Folder flat-list API uses workspace header and camelCase JSON | PASS |
| E2E | Login, nested navigation, breadcrumbs, folder creation | PASS (5 tests) |
| Platform | |
| Release | |

## Harness Delta

- Registered EXC-010 in the harness database.

## Evidence

- Added Zustand flat-list store with deterministic tree and breadcrumb helpers.
- Added recursive sidebar tree, folder content list, create dialog, and breadcrumb navigation.
- Added `GET /api/v1/folders?all=true` while preserving parent-filtered listing.
- Browser screenshot: `/private/tmp/exc-010-dashboard.png`.
