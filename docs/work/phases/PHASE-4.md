---
artifact_type: phase
id: PHASE-4
status: in_review
owner: human
human_fields: [goal, scope, out_of_scope, priority, success_criteria]
ai_fields: [risks, dependencies, verification_plan, completion_summary]
shared_fields: [status, trace, tickets_and_bugs]
trace:
  backlog_items: []
  roadmap: ROADMAP
  requirements: [V2-REQ-03, V2-REQ-04, V2-REQ-05, V2-REQ-08]
  tickets: [EXC-024, EXC-025, EXC-026, EXC-027, EXC-028]
  bugs: []
  test_verification: collaboration-v2-real-stack
  validation_matrix: VALIDATION_MATRIX
  adrs: [0006]
  release_notes: 2026-10-19-collaboration-v2
---

# Phase: PHASE-4 Real-time Collaboration & Voice Calling

## Status

- ID: PHASE-4
- Status: in_review
- Owner: human
- Created: 2026-06-03
- Updated: 2026-10-19

## Goal

Implement seamless multi-user collaboration and voice calling with a focus on self-hosted infrastructure and minimal port exposure.

## Scope

- Durable Yjs sync, presence/follow, and voice signaling through the custom
  authenticated `/collab/` WebSocket.
- Voice call support within the whiteboard.
- Unified TURN/STUN relay solution (Single port/endpoint target).
- Multi-user editing with cursor visibility and presence.
- Low-latency synchronization for up to 10 concurrent users per scene.

## Out Of Scope

- AI features (Phase 5).

## Implementation Decisions

- **D-01: Relay-first:** Prioritize relay (TURN) over P2P for reliability in restrictive networks.
- **D-02: Custom authenticated WebSocket:** Use one bounded `/collab/`
  connection for Yjs sync, awareness, presence, follow, and WebRTC voice
  signaling. Socket.IO and WebRTC data-channel scene sync are retired.

## Tickets And Bugs

| ID | Type | Title | Status | Link |
| --- | --- | --- | --- | --- |
| EXC-024 | ticket | Coturn & Docker Setup | implemented | [EXC-024](../tickets/phase-01/EXC-024/EXC-024-coturn-and-docker-setup.md) |
| EXC-025 | ticket | Historical Socket.IO Backend + Frontend Client (superseded) | done | [EXC-025](../tickets/phase-01/EXC-025/EXC-025-socket.io-backend---frontend-client.md) |
| EXC-026 | ticket | Collaborative Cursors & Presence | implemented | [EXC-026](../tickets/phase-01/EXC-026/EXC-026-collaborative-cursors-and-presence.md) |
| EXC-027 | ticket | Historical WebRTC DataChannel + Scene Sync (superseded) | done | [EXC-027](../tickets/phase-01/EXC-027/EXC-027-webrtc-datachannel---scene-sync.md) |
| EXC-028 | ticket | Voice Calling | implemented | [EXC-028](../tickets/phase-01/EXC-028/EXC-028-voice-calling.md) |

## Success Criteria

- [x] Multiple users can edit the same scene simultaneously.
- [x] Voice calls are clear and stable when browser media permissions and TURN are available.
- [x] Local production-stack behavior runs through `/collab/`; Socket.IO is
      absent from runtime/deployment dependencies.
- [ ] VM100 cutover, controlled reboot recovery, migrations, and public-origin
      WebSocket/TURN canary pass.

## Gate Checklist

- [x] Phase has linked requirements or explicit discovery goal
- [x] Tickets/bugs are created or planned
- [x] Risks and dependencies are recorded
- [x] Verification plan is defined
- [x] Release/changelog need is identified

## Completion Summary

The historical Socket.IO/WebRTC-data-channel implementation was superseded by
Collaboration v2. Custom `/collab/`, durable Yjs ACK/replay, native
presence/follow, WebRTC voice signaling, scoped TURN, assets, and recovery pass
the 17/17 local real-stack suite. Production rollout remains under
[PHASE-COLLAB-V2](PHASE-COLLAB-V2.md); enterprise parity remains separate.
