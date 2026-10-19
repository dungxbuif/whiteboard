---
artifact_type: phase
id: COLLAB-V2
status: in_review
owner: shared
updated: 2026-10-19
---

# Collaboration v2, legacy dashboard and native MCP

Design and implementation were approved by the user. A1–A4, B1–B4, C1/C2,
D1–D4 and the local E1/E2 gates are implemented and verified on merged master.
The guarded real-stack suite passes 17/17, including assets, recovery,
presence/follow, generated two-browser audio, MCP mutation/export and the
50-client durability target. Production remains v1 until E3 rollout and public
canaries pass. User confirmed org means workspace and removed the monochrome
override: preserve the original dashboard style.

## Trace

- [Approved design](../../superpowers/specs/2026-10-15-collaboration-v2-design.md)
- [Programme / task index](../../superpowers/plans/2026-10-16-collaboration-v2.md)
- [Backlog](../BACKLOG.md), [roadmap](../ROADMAP.md), [validation](../VALIDATION_MATRIX.md)
- [Current release checklist](../../operations/RELEASE_CHECKLIST.md), [changelog](../../releases/CHANGELOG.md)
- [Execution evidence](../verification/collaboration-v2.md), [real-stack acceptance](../verification/collaboration-v2-real-stack.md), [release audit](../verification/2026-10-19-release-pause-audit.md).
- ADRs: 0009 access and0010 Yjs writer created;0011 MCP created;0012 records dashboard presentation.

## Execution slices

| Slice | Tasks | Result | State |
| --- | --- | --- | --- |
| A | A1–A4 | A1–A4 verified/reviewed | complete |
| B | B1–B4 | Durable writer/browser integration, image movement/reopen recovery, native presence/follow and voice pass local review and real-stack acceptance | complete |
| C | C1–C2 | C1/C2 verified and reviewed;29browser/35unit/build pass | complete |
| D | D1–D4 | D1–D4 verified and reviewed; D4 lifecycle regressions and local nonroot image pass | complete |
| E | E1–E3 | Local 17/17 acceptance and 650/650 durable ACKs pass; backup rehearsal, packaging and two-network TURN evidence exist; final frozen Linux/amd64 build, fresh cutover backup, VM100 migration/deploy and public canary remain | in_progress |

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

## Open release gates

- Prove automatic recovery after a controlled Docker/VM restart.
- Take and restore-check a fresh cutover PostgreSQL backup.
- Install four distinct v2 service secrets without exposing their values.
- Freeze reviewed Linux/amd64 images and verify migrations 7–13 at version 13,
  dirty=false.
- Run the public-origin roles/invite/share/revoke/S3/custom-WebSocket/MCP/voice,
  restart/replay and denial canaries.
- Finish whole-branch security/dependency review, Homelab reconciliation and
  final release evidence.
- Revoke the retired Google OAuth credential. History scrub and remote push are
  complete, but provider-side revocation remains external.
- Run live AI provider UAT after the user supplies the promised backend-only
  OpenAI-compatible key.

Comments, guest comments, screen sharing, live presentation/spotlight/waiting
room, activity UI, version history, libraries/collections, domain capture,
SSO/SCIM and retention remain approved enterprise-parity backlog. They are not
claimed by this phase.

## Planning verification — 2026-10-16

Source inspected: Go HTTP/router/access/assets/scene persistence, frontend
Excalidraw/realtime stores, legacy dashboard references, existing Compose and
project standards. Self-review added writer fencing, durable dedup after
compaction, scoped offline storage, sync-step-2 authorization, Mermaid native
capability checking and PAT/OAuth compatibility boundaries.
This paragraph records the original planning gate. Runtime implementation and
local UAT have since completed; current proof is the execution and real-stack
evidence linked above. Production UAT remains an E3 release gate.

Planning checks executed: Python structural validation passed for six plan
documents / 17 task IDs, required headers, local links, balanced fences and
absence of TODO/TBD placeholders. `rtk git diff --check` passed before staging.
Public human and agent-readable runtime guides now describe the locally
verified release candidate and explicitly state that it is not deployed.
