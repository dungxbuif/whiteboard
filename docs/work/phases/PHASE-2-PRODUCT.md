---
artifact_type: phase
id: PHASE-2-PRODUCT
status: planned
owner: human
human_fields:
  - goal
  - scope
  - out_of_scope
  - priority
  - success_criteria
ai_fields:
  - risks
  - dependencies
  - verification_plan
  - completion_summary
shared_fields:
  - status
  - trace
  - tickets_and_bugs
trace:
  backlog_items: []
  roadmap: ROADMAP
  requirements: [P2-REQ-01, P2-REQ-02, P2-REQ-03, P2-REQ-04, P2-REQ-05]
  tickets: [EXC-015, EXC-016, EXC-017, EXC-018]
  bugs: []
  test_verification: TBD
  validation_matrix: VALIDATION_MATRIX
  adrs: []
  release_notes: TBD
---

# Phase: PHASE-2-PRODUCT Presentation Mode

## Status

- ID: PHASE-2-PRODUCT
- Status: planned
- Owner: human
- Created: 2026-06-03
- Updated: 2026-06-11

## Goal

Implement a high-fidelity presentation mode that mirrors Excalidraw Plus functionality, allowing users to define frames as slides and navigate through them.

## Scope

- Frame-based slide management.
- Presentation view with navigation.
- Focus mode for distraction-free viewing.

## Out Of Scope

- Sharing (Phase 3).
- Collaboration (Phase 4).

## Implementation Decisions

- **D-01: Frame-based:** Use Excalidraw's native frame element as the slide unit.
- **D-02: State Sync:** Presentation state (active frame) is local to the viewer.

## Tickets And Bugs

| ID | Type | Title | Status | Link |
| --- | --- | --- | --- | --- |
| EXC-015 | ticket | Frame Persistence & Scene Store Extension | planned | [EXC-015](../tickets/phase-01/EXC-015/EXC-015-frame-persistence-and-scene-store-extension.md) |
| EXC-016 | ticket | Frame Tool Activation | planned | [EXC-016](../tickets/phase-01/EXC-016/EXC-016-frame-tool-activation.md) |
| EXC-017 | ticket | Presentation Engine & Controls | planned | [EXC-017](../tickets/phase-01/EXC-017/EXC-017-presentation-engine-and-controls.md) |
| EXC-018 | ticket | Slide Navigator & Keyboard Support | planned | [EXC-018](../tickets/phase-01/EXC-018/EXC-018-slide-navigator-and-keyboard-support.md) |

## Success Criteria

- [ ] Frames can be created and ordered.
- [ ] Presentation mode activates/deactivates cleanly.
- [ ] Keyboard navigation works between frames.

## Gate Checklist

- [x] Phase has linked requirements or explicit discovery goal
- [x] Tickets/bugs are created or planned
- [ ] Risks and dependencies are recorded
- [ ] Verification plan is defined
- [ ] Release/changelog need is identified

## Completion Summary

Complete this section when the phase is done.
