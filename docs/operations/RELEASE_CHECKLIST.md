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


## Collaboration v2 release — deployed, product acceptance pending

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
- [x] Final reviewed source built for VM100 Linux/amd64; exact image IDs are captured in the [release note](../releases/2026-10-19-collaboration-v2.md).
- [x] Existing environment/service ownership checked; four distinct v2 secrets of at least 32 bytes are installed and verified without printing values.
- [x] Preflight PostgreSQL backup restored in an isolated container; actual migrator7–13 passed, final13clean. [Evidence](../work/verification/collaboration-v2-backup.md).
- [x] Fresh cutover backup recorded and restore-checked immediately before production migrations; all 14 table counts matched. [Evidence](../work/verification/collaboration-v2-backup.md).
- [x] Controlled VM reboot recreated/awaited external networks through the installed recovery unit and restored all four healthy application services plus public health without manual Compose intervention.
- [x] Controlled replacement-scene recovery rehearsal reviewed (`93d57a1`,24independent checks): preserve latest content, receipts and original quarantined identity; assets/overrides refuse. Image-only rollback forbidden after promotion.
- [x] Applied the tested `turn:103.82.21.202:5349` endpoint; a fresh scoped credential completed authenticated UDP Allocate, returned a relayed address and was released with Refresh lifetime zero.
- [ ] Public HTTPS canary: authenticated deep-link save/reload, dashboard/settings access UI, S3 write/read/delete, native MCP authentication/read and revoked-key denial, internal-route denial, Docker DNS runtime restart/replay and authenticated TURN allocation pass. Public view/edit mutation, invitation acceptance/role/revoke, native MCP write/export and two-peer browser voice/media remain.
- [ ] Browser screenshots/desktop/mobile/keyboard/deep links reviewed; visible desktop/deep-link and post-restart `People (1)` / `Saved` checks preserve the original dashboard and native scene style. Mobile/keyboard coverage and the complete roles matrix remain.
- [x] Public `/internal` and encoded/normalized variants return 404. Nginx access logging is disabled and local nginx/backend/collab/MCP/Cloudflared logs inspected on VM100 contain no credential-bearing URL; Cloudflare dashboard retention policy was outside VM access.
- [x] Post-restart/reboot health/readiness/admission/sync observation recorded; four services are healthy, migration13 is clean and the public browser returns `People (1)` / `Saved`. Existing local ACK/compaction proof remains linked above.
- [ ] Internal architecture/API/schema/runbooks plus human and AI-readable public docs reconciled with deployed behavior. Application docs are updated in this release commit and Homelab reconciliation is committed as `9722490`; final canary evidence remains.
- [x] Whole-branch source/security review and frozen dependency review complete. `npm audit --omit=dev` reports zero production findings; the nine remaining development/build entries are classified in [dependency evidence](../work/verification/collaboration-dependencies.md).
- [ ] Final release sign-off commit recorded. The Git remote
  is configured and the history-scrubbed master push succeeded.

Latest evidence: [collaboration verification](../work/verification/collaboration-v2.md), [packaging/TURN](../work/verification/collaboration-v2-packaging.md), [cutover backup](../work/verification/collaboration-v2-backup.md), [tmpfs incident](../work/verification/compose-tmpfs-vm100-2026-10-19.md), [Docker DNS recovery](../work/verification/nginx-docker-dns-recovery-2026-10-19.md), [release note](../releases/2026-10-19-collaboration-v2.md), [recovery evidence](../work/verification/collaboration-recovery.md) and [recovery runbook](collaboration-recovery.md). AI provider calls remain dependent on an actual configured API key; MCP native diagram generation does not require one. Retired Google credential revocation remains an external provider operation.
