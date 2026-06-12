# EXC-012: Excalidraw Integration

## Status

done

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

A Detail Design document following `docs/templates/detail_design.md` must be created and approved by the reviewer before implementation starts.

- **Link to Detail Design**: [Design Document](EXC-012-design.md) (approved 2026-06-12)

- **Modified Files**: `src/frontend/src/components/whiteboard/excalidraw-wrapper.tsx`, `src/frontend/src/store/scene-store.ts`, `src/frontend/src/components/src/frontend/file-list.tsx`, `scene-actions.tsx`
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

- `npm run test:unit`: pass, 16 tests across 5 files.
- `npm run build`: pass; Excalidraw JavaScript is emitted in lazy chunks.
- `npx playwright test --reporter=line --workers=1`: pass, 7 Chromium tests.
- Browser flow verified: create scene, render canvas, draw, one debounced PATCH, return, rename, star, and trash.
- Responsive proof: 1280x720 and 390x844 screenshots; mobile document width does not exceed viewport.
- Console proof: no application console errors in the Excalidraw action flow.
- Browser plugin was not available; repository Playwright was used per frontend testing policy.
- `npm audit --omit=dev`: 1 high and 8 moderate transitive advisories remain in Excalidraw's Mermaid/nanoid dependencies. No compatible fix is currently available; see ADR-0007.

## Docs Review

- Requirements: no change; REQ-10 behavior was implemented as specified.
- Architecture: updated editor ownership, lazy loading, and save ordering.
- API: no change; EXC-011 endpoints were consumed without contract changes.
- ERD/data: no change; existing scene JSONB is used.
- SDD: created `docs/architecture/SDD.md` for the client persistence flow.
- ADR: added ADR-0007 for the major runtime dependency and tradeoffs.
- Context, backlog, phase, validation matrix, and changelog: updated.
