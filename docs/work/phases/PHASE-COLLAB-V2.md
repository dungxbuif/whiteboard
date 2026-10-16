---
artifact_type: phase
id: COLLAB-V2
status: in_progress
owner: shared
updated: 2026-10-16
---

# Collaboration v2, legacy dashboard and native MCP

Design and implementation approved by user. A1–A4, B1/B2, C1/C2 and D1/D2 are verified/reviewed in codex/collaboration-v2; production remains v1. User confirmed org means workspace and removed the monochrome override: port the original dashboard style.

## Trace

- [Approved design](../../superpowers/specs/2026-10-15-collaboration-v2-design.md)
- [Programme / task index](../../superpowers/plans/2026-10-16-collaboration-v2.md)
- [Backlog](../BACKLOG.md), [roadmap](../ROADMAP.md), [validation](../VALIDATION_MATRIX.md)
- [Current release checklist](../../operations/RELEASE_CHECKLIST.md), [changelog](../../releases/CHANGELOG.md)
- [Execution evidence](../verification/collaboration-v2.md).
- ADRs: 0009 access and0010 Yjs writer created;0011 MCP created;0012 records dashboard presentation.

## Execution slices

| Slice | Tasks | Result | State |
| --- | --- | --- | --- |
| A | A1–A4 | A1–A4 verified/reviewed | complete |
| B | B1–B4 | B1/B2 verified and reviewed; B3 browser integration active; B4 pending | in_progress |
| C | C1–C2 | C1/C2 verified and reviewed;29browser/35unit/build pass | complete |
| D | D1–D4 | D1–D3 verified and reviewed; D4 lifecycle review corrections active (196 JS/full Go implementation gate passed) | in_progress |
| E | E1–E3 | local packaging/proxy and diagnostic cross-network TURN audio pass; full integration/load/canary/rollback pending | in_progress |

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

User explicitly confirmed that org means workspace. Implement workspace
ownership/membership and MCP tools; no separate organization hierarchy.
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
