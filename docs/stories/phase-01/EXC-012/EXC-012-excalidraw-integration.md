# EXC-012: Excalidraw Integration

## Status

planned

## Lane

normal

## Product Contract

Install @excalidraw/excalidraw, ExcalidrawWrapper component, Zustand scene-store (currentScene, debounced saveScene, LWW), Scene actions UI

## Relevant Product Docs

- `01-09-PLAN.md`

## Acceptance Criteria

- Excalidraw canvas component is correctly rendered and responsive.
- Zustand scene store debounces saving changes to the backend database.
- Last-Write-Wins logic is implemented for initial conflict resolution.
- Scene actions UI (star, trash, rename, delete) are functional.

## Design Notes

A Detail Design document following `docs/templates/detail_design.md` must be created at `docs/stories/phase-01/EXC-012-design.md` and approved by the reviewer before implementation starts.

- **Link to Detail Design**: [Design Document](EXC-012-design.md)

- **Modified Files**: `dashboard/src/components/whiteboard/excalidraw-wrapper.tsx`, `dashboard/src/store/scene-store.ts`, `dashboard/src/components/dashboard/file-list.tsx`, `scene-actions.tsx`
- **Sprint Phase**: phase-01

## Validation

| Layer | Expected proof |
| --- | --- |
| Unit | cd dashboard && npm run build |
| Integration | |
| E2E | |
| Platform | |
| Release | |

## Harness Delta

- Registered EXC-012 in the harness database.

## Evidence

(No evidence yet)
