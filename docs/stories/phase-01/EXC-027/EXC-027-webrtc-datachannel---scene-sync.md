# EXC-027: WebRTC DataChannel + Scene Sync

## Status

planned

## Lane

high-risk

## Product Contract

WebRTC signaling (offer/answer/ICE via Socket.io), webrtc.ts (RTCPeerConnection, DataChannel), useWhiteboardSync hook

## Relevant Product Docs

- `04-02-PLAN.md`

## Acceptance Criteria

- RTCPeerConnection is established via socket signaling.
- DataChannel synchronizes canvas elements directly between peers.
- Element updates sync smoothly with conflict resolution.

## Design Notes

A Detail Design document following `docs/templates/detail_design.md` must be created at `docs/stories/phase-04/EXC-027-design.md` and approved by the reviewer before implementation starts.

- **Link to Detail Design**: [Design Document](EXC-027-design.md)

- **Modified Files**: `src/frontend/src/lib/webrtc.ts`, `src/backend/internal/delivery/http/socket.go`, `src/frontend/src/hooks/useWhiteboardSync.ts`
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

- Registered EXC-027 in the harness database.

## Evidence

(No evidence yet)
