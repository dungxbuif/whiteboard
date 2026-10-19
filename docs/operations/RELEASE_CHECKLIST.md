---
artifact_type: release_checklist
id: RELEASE_CHECKLIST
status: active
owner: shared
human_fields: [release_approval, rollback_acceptance]
ai_fields: [pre_release_checks, post_release_checks, verification_notes]
shared_fields: [status]
---

# Release Checklist

## Field Ownership

- Human owns release approval and rollback acceptance.
- AI records pre-release checks, post-release checks, and verification notes.

## Prior v1 release — 2026-10-15

The checked items below describe the prior v1 release. They are not v2 acceptance.

## Pre-Release

- [x] Backend tests passed (`go test ./...`)
- [x] Frontend unit tests passed (`npm run test:unit`)
- [x] Frontend production build passed (`npm run build`)
- [x] Browser smoke tests passed (`npx playwright test`)
- [x] Docker Compose syntax passed (`docker compose config`)
- [x] Master docs reconciled
- [x] ADRs updated
- [x] Release notes prepared in `docs/releases/CHANGELOG.md` and `docs/releases/2026-10-15-whiteboard-vm100.md`
- [ ] Production `.env.prod` populated in the secret manager
- [ ] JWT/TURN/S3 secrets rotated
- [ ] PostgreSQL backup and restore point confirmed
- [ ] Rollback image tags confirmed

## Smoke Verification

- [x] `GET /healthz` returns HTTP 200 with body `ok`
- [x] Password login completes on the HTTPS origin with a user provisioned by `userctl`
- [x] Workspace list and scene create/save work
- [x] Authenticated scene deep link opens only for the owning workspace
- [x] Public share and password challenge work (automated; live public viewer/revoke smoke also passed)
- [ ] Socket.IO connection and presence update work
- [ ] TURN voice call works from two networks
- [ ] AI chat and diagram generation work with `AI_API_KEY`

## Post-Release

- [x] Deployment verified (VM100 health check and live public-share smoke)
- [ ] Monitoring checked
- [ ] Incidents recorded if any
- [x] `docs/CONTEXT.md` updated

## Evidence Notes — 2026-10-15

- Live public-share smoke used a temporary token and revoked it after the check;
  the revoked token returned `404 Shared scene not found`.
- Socket.IO two-browser presence, cross-network TURN voice, and AI generation
  with a real provider key remain human UAT items.
- Secret rotation, PostgreSQL backup/restore, and rollback image acceptance
  remain human-owned production gates.


## Collaboration v2 release — pending

User authorization covers implementation, local merge and VM100 deploy. The following technical gates remain required; prior v1 checks above do not satisfy them.

- [x] Access, invitation, public links and capability tasks A1–A4 reviewed.
- [x] Durable runtime B1/B2 and dashboard C1/C2 reviewed.
- [x] Agent keys/audit,19presets and live mutation tasks D1–D3 reviewed.
- [x] B3 upload lifetime, flush serialization and same-account draft-conflict corrections reviewed (`3a20bb5`).
- [x] Pending-image movement, concurrent remote-field and reopen recovery regressions resolved (`9d8df7f`, `b89af60`).
- [x] B4 presence/follow/voice and idle heartbeat pass; legacy Socket.IO writer/signaling code is retired (`3d3ec2b`).
- [x] D4 browser startup and per-job process/temp cleanup corrections reviewed (`6c64b12`); local nonroot image smoke passed.
- [x] E1 real API/browser/S3/MCP role/recovery suite passed 17/17; 50-client load completed650/650 durable ACKs at p9513.236ms.
- [x] E2 idle socket through nginx exceeds75seconds after heartbeat fix (160seconds, six pings, zero live disconnects; scoped review passed).
- [ ] Final reviewed source rebuilt for VM100 Linux/amd64; exact image IDs captured.
- [ ] Existing environment/service ownership checked before secret changes; distinct v2 secrets installed without printing values.
- [x] Preflight PostgreSQL backup restored in an isolated container; actual migrator7–13 passed, final13clean. [Evidence](../work/verification/collaboration-v2-backup.md).
- [ ] Fresh cutover backup recorded immediately before production migrations.
- [x] Controlled replacement-scene recovery rehearsal reviewed (`93d57a1`,24independent checks): preserve latest content, receipts and original quarantined identity; assets/overrides refuse. Image-only rollback forbidden after promotion.
- [ ] Apply tested TURN gatewayIP endpoint, repeat actual deployed configuration proof; DNS hostname remains unverified until corrected.
- [ ] Public HTTPS canary: two browsers, public view/edit, invite, roles/revoke, agent native write/export, restart and denied requests.
- [ ] Browser screenshots/desktop/mobile/keyboard/deep links reviewed; original dashboard and native scene style retained.
- [ ] Upstream proxy logs checked for secret-bearing URLs; health/readiness/ack/compaction observation recorded.
- [ ] Internal architecture/API/schema/runbooks plus human and AI-readable public docs reconciled with deployed behavior.
- [ ] Whole-branch review complete, local master integration and release commit recorded (no Git remote configured).

Latest evidence: [collaboration verification](../work/verification/collaboration-v2.md), [packaging/TURN](../work/verification/collaboration-v2-packaging.md), [recovery evidence](../work/verification/collaboration-recovery.md) and [recovery runbook](collaboration-recovery.md). AI provider calls remain dependent on an actual configured API key; MCP native diagram generation does not require one.
