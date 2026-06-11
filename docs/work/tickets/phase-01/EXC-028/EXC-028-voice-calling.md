# EXC-028: Voice Calling

## Status

planned

## Lane

high-risk

## Product Contract

Voice signaling, useVoiceStore, audio.ts (getUserMedia, volume detection), VoiceProvider, VoiceControls UI

## Relevant Product Docs

- `04-03-PLAN.md`

## Acceptance Criteria

- Media devices are accessed and local audio track obtained.
- Audio streams are sent/received via peer connections.
- Mute controls and speaker indicators are functional.

## Design Notes

A Detail Design document following `docs/templates/detail_design.md` must be created at `docs/stories/phase-04/EXC-028-design.md` and approved by the reviewer before implementation starts.

- **Link to Detail Design**: [Design Document](EXC-028-design.md)

- **Modified Files**: `src/backend/internal/delivery/http/socket.go`, `src/frontend/src/store/useVoiceStore.ts`, `src/frontend/src/lib/audio.ts`, `src/frontend/src/components/collaboration/VoiceProvider.tsx`, `VoiceControls.tsx`
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

- Registered EXC-028 in the harness database.

## Evidence

(No evidence yet)
