---
artifact_type: phase
id: COLLAB-V2
status: in_review
owner: shared
updated: 2026-10-16
---

# Collaboration v2, legacy dashboard and native MCP

Design approved by user; implementation plans in review. No runtime implementation in this phase has started.

## Trace

- [Approved design](../../superpowers/specs/2026-10-15-collaboration-v2-design.md)
- [Programme / task index](../../superpowers/plans/2026-10-16-collaboration-v2.md)
- [Backlog](../BACKLOG.md), [roadmap](../ROADMAP.md), [validation](../VALIDATION_MATRIX.md)
- [Current release checklist](../../operations/RELEASE_CHECKLIST.md), [changelog](../../releases/CHANGELOG.md)
- Planned execution evidence: `docs/work/verification/collaboration-v2.md` (created by E1).
- Planned ADRs: 0009 access, 0010 Yjs writer, 0011 MCP; create during owning tasks.

## Execution slices

| Slice | Tasks | Result | State |
| --- | --- | --- | --- |
| A | A1–A4 | authoritative ACL, invites, public sessions, capabilities | in_review |
| B | B1–B4 | durable CRDT, offline editing, presence/follow/voice | in_review |
| C | C1–C2 | legacy dashboard style and management flows | in_review |
| D | D1–D4 | scoped MCP, editable diagram catalog, export | in_review |
| E | E1–E3 | integration/load proof, VM100 canary/rollback, docs | in_review |

Task identifiers in the subsystem plans are the bounded execution work items;
each owns files, public interfaces, failing proof, implementation, checks,
docs reconciliation and commit. No additional ticket duplication is required.

## Scope and acceptance

- Preserve old dashboard visual style and native scene library style.
- Invite email-bound users by copied seven-day single-use links; owner-only management.
- Independent public view/edit links, asset scope, live revocation and authorization checks.
- Custom Node WebSocket, Yjs snapshot/log, offline replay and durable acknowledgements.
- Agent discovers workspaces and diagrams, creates scenes, writes native elements live, exports safely.
- Verify browser, API, database and collab together, then deploy/observe on VM100 with tested rollback.
- Human and agent-readable docs reconciled to actual shipped behavior.

Organization hierarchy remains a separate open domain question: current code
has workspace ownership/membership only. This blocks org-specific tools, not
workspace/MCP planning. Do not silently equate org and workspace.
Enterprise parity P1/P2 candidates remain separate from these delivery slices.

## Planning verification — 2026-10-16

Source inspected: Go HTTP/router/access/assets/scene persistence, frontend
Excalidraw/realtime stores, legacy dashboard references, existing Compose and
project standards. Self-review added writer fencing, durable dedup after
compaction, scoped offline storage, sync-step-2 authorization, Mermaid native
capability checking and PAT/OAuth compatibility boundaries.
Runtime tests and UAT are pending implementation; documentation checks are
recorded in the planning commit, not represented as feature verification.

Planning checks executed: Python structural validation passed for six plan
documents / 17 task IDs, required headers, local links, balanced fences and
absence of TODO/TBD placeholders. `rtk git diff --check` passed before staging.
Public user/agent runtime documentation is not changed in this planning-only
commit; its creation and reconciliation are assigned to A–E and depend on
actual implementation evidence.
