# EXC-015: Frame Persistence & Scene Store Extension

## Status

done

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

A Detail Design document following `docs/templates/detail_design.md` is recorded at `EXC-015-design.md` and approved for implementation.

- **Link to Detail Design**: [Design Document](EXC-015-design.md)

- **Modified Files**: `src/frontend/src/store/scene-store.ts`
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

- `npm run test:unit`, `npm run build`, and `npx playwright test` pass. The
  ordering helper preserves existing frame order, appends new frames, and the
  debounced scene payload persists `presentationOrder`.
