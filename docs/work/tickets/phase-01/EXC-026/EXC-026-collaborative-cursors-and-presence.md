# EXC-026: Collaborative Cursors & Presence

## Status

implemented

## Lane

high-risk

## Product Contract

Socket.io: broadcast pointer-move + user-list-update, useCollaborationStore (Zustand), CollaborativeCursors overlay

## Relevant Product Docs

- `04-02-PLAN.md`

## Acceptance Criteria

- Pointer positions are broadcasted to all users in the same scene.
- Active users are tracked and displayed in the presence bar.
- Cursors render correctly over Excalidraw canvas.

## Design Notes

A Detail Design document following `docs/templates/detail_design.md` must be created at `docs/stories/phase-04/EXC-026-design.md` and approved by the reviewer before implementation starts.

- **Link to Detail Design**: [Design Document](EXC-026-design.md)

- **Modified Files**: `src/backend/internal/delivery/http/socket.go`, `src/frontend/src/store/useCollaborationStore.ts`, `src/frontend/src/components/whiteboard/CollaborativeCursors.tsx`
- **Sprint Phase**: phase-04

## Validation

| Layer | Expected proof |
| --- | --- |
| Unit | cd dashboard && npm run build |
| Integration | |
| E2E | |
| Platform | |
| Release | |

## Harness Delta

- Registered EXC-026 in the harness database.

## Evidence

Presence and cursor events are handled by the realtime store and collaboration toolbar; frontend build passes.
