# EXC-015: Frame Persistence & Scene Store Extension

## Status

planned

## Lane

normal

## Product Contract

Extend SceneData interface with presentationOrder, implement syncPresentationOrder(elements), ensure order persists in auto-sync payload

## Relevant Product Docs

- `02-01-PLAN.md`

## Acceptance Criteria

- SceneData state includes presentationOrder list.
- syncPresentationOrder updates order array based on current frames in scene.
- Auto-save preserves presentationOrder in database.

## Design Notes

A Detail Design document following `docs/templates/detail_design.md` must be created at `docs/stories/phase-02/EXC-015-design.md` and approved by the reviewer before implementation starts.

- **Link to Detail Design**: [Design Document](EXC-015-design.md)

- **Modified Files**: `dashboard/src/store/scene-store.ts`
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

- Registered EXC-015 in the harness database.

## Evidence

(No evidence yet)
