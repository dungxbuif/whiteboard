---
artifact_type: phase
id: PHASE-3
status: done
owner: human
human_fields: [goal, scope, out_of_scope, priority, success_criteria]
ai_fields: [risks, dependencies, verification_plan, completion_summary]
shared_fields: [status, trace, tickets_and_bugs]
trace:
  backlog_items: []
  roadmap: ROADMAP
  requirements: [P3-REQ-01, P3-REQ-02, P3-REQ-03, P3-REQ-04]
  tickets: [EXC-019, EXC-020, EXC-021, EXC-022, EXC-023]
  bugs: []
  test_verification: VALIDATION_MATRIX
  validation_matrix: VALIDATION_MATRIX
  adrs: []
  release_notes: CHANGELOG
---

# Phase: PHASE-3 Shareable Production

## Status

- ID: PHASE-3
- Status: done
- Owner: human
- Created: 2026-06-03
- Updated: 2026-10-19

## Goal

Enable public sharing of drawings with fine-grained access control, replicating the original Excalidraw production sharing logic.

## Scope

- Public read-only sharing links.
- Link permissions and scene-level access control.
- Legacy parity with existing `references/excalidraw` sharing logic.
- Basic password protection on shared links.

## Out Of Scope

- Real-time collaboration (Phase 4).
- AI features (Phase 5).

## Implementation Decisions

- **D-01: DB Persistence:** Use a dedicated `scene_permissions` table for share links.
- **D-02: Parity:** Mirror the NestJS public module implementation exactly.

## Tickets And Bugs

| ID | Type | Title | Status | Link |
| --- | --- | --- | --- | --- |
| EXC-019 | ticket | Sharing DB Schema & Domain | done | [EXC-019](../tickets/phase-01/EXC-019/EXC-019-sharing-db-schema-and-domain.md) |
| EXC-020 | ticket | Sharing Backend | done | [EXC-020](../tickets/phase-01/EXC-020/EXC-020-sharing-backend.md) |
| EXC-021 | ticket | Public API & Share Dialog UI | done | [EXC-021](../tickets/phase-01/EXC-021/EXC-021-public-api-and-share-dialog-ui.md) |
| EXC-022 | ticket | Password Protection | done | [EXC-022](../tickets/phase-01/EXC-022/EXC-022-password-protection.md) |
| EXC-023 | ticket | Sharing Integration Tests | done | [EXC-023](../tickets/phase-01/EXC-023/EXC-023-sharing-integration-tests.md) |

## Success Criteria

- [x] Public links are accessible without login.
- [x] Revoking a link immediately blocks access.

## Gate Checklist

- [x] Phase has linked requirements or explicit discovery goal
- [x] Tickets/bugs are created or planned
- [x] Risks and dependencies are recorded in the ticket designs
- [x] Verification plan is defined in the validation matrix
- [x] Release/changelog need is identified

## Completion Summary

Implemented opaque public links, authenticated revoke/update controls,
optional bcrypt password challenges, a share dialog, and a read-only public
viewer. Backend and browser verification pass.
