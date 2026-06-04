# EXC-017: Presentation Engine & Controls

## Status

planned

## Lane

normal

## Product Contract

presentation-store.ts (isPresenting, activeSlideIndex, togglePresentation), viewModeEnabled binding, goToSlide(index) with scrollToContent, PresentationControls overlay

## Relevant Product Docs

- `02-02-PLAN.md`

## Acceptance Criteria

- Zustand presentation-store tracks presenter state.
- Toggling presentation mode disables editing tools and enables view mode.
- PresentationControls render navigation controls over the canvas.
- goToSlide centers viewport on active frame bounds.

## Design Notes

A Detail Design document following `docs/templates/detail_design.md` must be created at `docs/stories/phase-02/EXC-017-design.md` and approved by the reviewer before implementation starts.

- **Link to Detail Design**: [Design Document](EXC-017-design.md)

- **Modified Files**: `dashboard/src/store/presentation-store.ts`, `dashboard/src/components/whiteboard/presentation-controls.tsx`, `excalidraw-wrapper.tsx`
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

- Registered EXC-017 in the harness database.

## Evidence

(No evidence yet)
