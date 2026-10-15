---
artifact_type: phase
id: PHASE-4
status: implemented
owner: human
human_fields: [goal, scope, out_of_scope, priority, success_criteria]
ai_fields: [risks, dependencies, verification_plan, completion_summary]
shared_fields: [status, trace, tickets_and_bugs]
trace:
  backlog_items: []
  roadmap: ROADMAP
  requirements: [P4-REQ-01, P4-REQ-02, P4-REQ-03, P4-REQ-04]
  tickets: [EXC-024, EXC-025, EXC-026, EXC-027, EXC-028]
  bugs: []
  test_verification: TBD
  validation_matrix: VALIDATION_MATRIX
  adrs: [0006]
  release_notes: TBD
---

# Phase: PHASE-4 Real-time Collaboration & Voice Calling

## Status

- ID: PHASE-4
- Status: implemented
- Owner: human
- Created: 2026-06-03
- Updated: 2026-06-11

## Goal

Implement seamless multi-user collaboration and voice calling with a focus on self-hosted infrastructure and minimal port exposure.

## Scope

- WebRTC/Socket.io integration.
- Voice call support within the whiteboard.
- Unified TURN/STUN relay solution (Single port/endpoint target).
- Multi-user editing with cursor visibility and presence.
- Low-latency synchronization for up to 10 concurrent users per scene.

## Out Of Scope

- AI features (Phase 5).

## Implementation Decisions

- **D-01: Relay-first:** Prioritize relay (TURN) over P2P for reliability in restrictive networks.
- **D-02: Socket.io for Presence:** Use Socket.io for cursor and presence, WebRTC for data/voice.

## Tickets And Bugs

| ID | Type | Title | Status | Link |
| --- | --- | --- | --- | --- |
| EXC-024 | ticket | Coturn & Docker Setup | implemented | [EXC-024](../tickets/phase-01/EXC-024/EXC-024-coturn-and-docker-setup.md) |
| EXC-025 | ticket | Socket.io Backend + Frontend Client | implemented | [EXC-025](../tickets/phase-01/EXC-025/EXC-025-socket.io-backend---frontend-client.md) |
| EXC-026 | ticket | Collaborative Cursors & Presence | implemented | [EXC-026](../tickets/phase-01/EXC-026/EXC-026-collaborative-cursors-and-presence.md) |
| EXC-027 | ticket | WebRTC DataChannel + Scene Sync | implemented | [EXC-027](../tickets/phase-01/EXC-027/EXC-027-webrtc-datachannel---scene-sync.md) |
| EXC-028 | ticket | Voice Calling | implemented | [EXC-028](../tickets/phase-01/EXC-028/EXC-028-voice-calling.md) |

## Success Criteria

- [x] Multiple users can edit the same scene simultaneously.
- [x] Voice calls are clear and stable when browser media permissions and TURN are available.
- [x] Infrastructure runs behind the HTTP endpoint and `/socket.io/` transport with coturn configuration.

## Gate Checklist

- [x] Phase has linked requirements or explicit discovery goal
- [x] Tickets/bugs are created or planned
- [x] Risks and dependencies are recorded
- [x] Verification plan is defined
- [x] Release/changelog need is identified

## Completion Summary

Completed with authenticated Socket.IO rooms, presence/cursor fan-out, scene synchronization, WebRTC voice signaling, and coturn compose configuration. Verified by `go test ./...`, frontend unit/build checks, and `docker compose config`.
