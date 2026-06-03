# EXC-018: Slide Navigator & Keyboard Support

## Status

planned

## Lane

normal

## Product Contract

SlideNavigator sidebar (list frames, jump, reorder), keyboard hooks (Arrow keys, Space, Escape), smooth transitions

## Relevant Product Docs

- `02-03-PLAN.md`

## Acceptance Criteria

- SlideNavigator sidebar lists all slides with thumbnails.
- Reordering slides updates presentationOrder in the store.
- Keyboard shortcuts (Arrows, Space, Escape) control slide navigation.
- Slide transitions utilize smooth viewport animations.

## Design Notes

A Detail Design document following `docs/templates/detail_design.md` must be created at `docs/stories/phase-02/EXC-018-design.md` and approved by the reviewer before implementation starts.

- **Link to Detail Design**: [Design Document](EXC-018-design.md)

- **Modified Files**: `dashboard/src/components/whiteboard/slide-navigator.tsx`, `excalidraw-wrapper.tsx`, `scene-store.ts`
- **Sprint Phase**: phase-02

## Validation

| Layer | Expected proof |
| --- | --- |
| Unit | cd dashboard && npm run build |
| Integration | |
| E2E | |
| Platform | |
| Release | |

## Harness Delta

- Registered EXC-018 in the harness database.

## Evidence

(No evidence yet)
