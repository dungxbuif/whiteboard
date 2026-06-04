# EXC-021: Public API & Share Dialog UI

## Status

planned

## Lane

normal

## Product Contract

PublicHandler (GET /api/public/scenes/{id} — no auth), useSharingStore, ShareDialog component, PublicView page, route /p/:sceneId

## Relevant Product Docs

- `03-02-PLAN.md`

## Acceptance Criteria

- PublicHandler serves scenes to unauthenticated requests if sharing is active.
- ShareDialog allows copying the public sharing link.
- PublicView page renders the read-only Excalidraw canvas.

## Design Notes

A Detail Design document following `docs/templates/detail_design.md` must be created at `docs/stories/phase-03/EXC-021-design.md` and approved by the reviewer before implementation starts.

- **Link to Detail Design**: [Design Document](EXC-021-design.md)

- **Modified Files**: `internal/delivery/http/handler/public_handler.go`, `src/features/sharing/stores/useSharingStore.ts`, `src/features/sharing/components/ShareDialog.tsx`, `src/pages/PublicView.tsx`
- **Sprint Phase**: phase-03

## Validation

| Layer | Expected proof |
| --- | --- |
| Unit | cd dashboard && npm run build |
| Integration | |
| E2E | |
| Platform | |
| Release | |

## Harness Delta

- Registered EXC-021 in the harness database.

## Evidence

(No evidence yet)
