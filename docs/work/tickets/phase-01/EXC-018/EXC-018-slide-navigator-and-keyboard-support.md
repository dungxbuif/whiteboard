# EXC-018: Slide Navigator & Keyboard Support

## Status

done

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

A Detail Design document following `docs/templates/detail_design.md` is recorded at `EXC-018-design.md` and approved for implementation.

- **Link to Detail Design**: [Design Document](EXC-018-design.md)

- **Modified Files**: `src/frontend/src/components/whiteboard/slide-navigator.tsx`, `excalidraw-wrapper.tsx`, `scene-store.ts`
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

- `npm run build` and `npx playwright test` pass. The slide navigator provides
  selection and reorder controls, and the wrapper handles Arrow, Space, and
  Escape keyboard shortcuts while presenting.
