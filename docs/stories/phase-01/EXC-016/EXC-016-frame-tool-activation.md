# EXC-016: Frame Tool Activation

## Status

planned

## Lane

normal

## Product Contract

Enable Frame tool in Excalidraw toolbar, detect new frames on onChange, auto-call syncPresentationOrder

## Relevant Product Docs

- `02-01-PLAN.md`

## Acceptance Criteria

- Frame tool is active and visible in the Excalidraw toolbar.
- Adding a frame on the canvas triggers slide order sync.

## Design Notes

A Detail Design document following `docs/templates/detail_design.md` must be created at `docs/stories/phase-02/EXC-016-design.md` and approved by the reviewer before implementation starts.

- **Link to Detail Design**: [Design Document](EXC-016-design.md)

- **Modified Files**: `dashboard/src/components/whiteboard/excalidraw-wrapper.tsx`
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

- Registered EXC-016 in the harness database.

## Evidence

(No evidence yet)
