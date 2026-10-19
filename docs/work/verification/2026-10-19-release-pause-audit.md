# Collaboration v2 release pause audit — 2026-10-19

Status: paused by the user before production v2 cutover. This file is the machine-readable and human-readable resume inventory for the next session.

## Proven at this checkpoint

- Local `master` contains the collaboration v2 implementation through merge commit `6082845`.
- Root workspace tests pass: collaboration runtime149, MCP43, contracts13 and diagram kit41 (246 total).
- Frontend unit tests pass25 files/93 tests and the production build succeeds.
- All Go packages pass.
- Prior guarded real-stack acceptance remains valid:17/17 flows,650/650 durable ACKs and MCP43/43.
- Production remains v1. After the VM100 reboot outage, backend/frontend were manually restarted and `https://whiteboard.dungxbuif.com/healthz` returned HTTP200.
- Production S3 bucket is `whiteboard`.

## P0 release blockers

1. **Retired OAuth credential:** GitHub Push Protection found a Google OAuth client ID and secret in historical `src/frontend/playwright.config.ts`. Rewrite all Git history before the first remote push, verify the values no longer exist in any pushed ref, and revoke the retired credential in Google Cloud. Do not bypass push protection.
2. **Automatic reboot recovery:** external Docker networks were unavailable when v1 containers restarted after the VM reboot. Backend/frontend exited and `restart: unless-stopped` did not reattach them. Define service/network ordering or a retry unit, then prove a controlled Docker/VM restart returns public health to HTTP200 without manual `compose up`.
3. **Fresh cutover backup:** create a private custom-format PostgreSQL dump immediately before migration; record SHA-256, inspect with `pg_restore --list`, restore into an isolated database and compare migration state and row counts.
4. **Four distinct v2 service secrets:** install `COLLAB_JWT_SECRET`, `AUTHORITY_SERVICE_SECRET`, `MUTATION_GRANT_SECRET` and `COLLAB_INTERNAL_SECRET`, each at least32 bytes and different from login JWT. Verify without printing values.
5. **Frozen Linux/amd64 release artifacts:** build from reviewed source, record exact image IDs/platforms, run migrations7–13 and verify schema version13 with `dirty=false`.
6. **Production canary:** verify roles, invitations, revocation, public view/edit, UUID/alias deep-link refresh, S3 paste/reload, custom `/collab/`, MCP native mutation/export, denial paths, runtime restart/replay and TURN voice with multiple principals/browsers.

## P1 release work

- Use the proven TURN endpoint `turn:103.82.21.202:5349` until split DNS is corrected, then repeat relay/audio proof on the deployed configuration.
- Inspect proxy/upstream logs for credential leakage; record health/readiness, ACK, compaction, RSS/error metrics and `/internal` denial.
- Review the whole diff from the pre-programme base through the release commit for correctness, authorization, security and deployment risks.
- Run a fresh dependency audit on the frozen lock/images and disposition inherited advisories by reachability or remediation. The last install reported1 low,2 moderate and8 high aggregate findings.
- Reconcile application and Homelab architecture, runbooks, deployment records, release notes/checklist, context, backlog and validation evidence with the actual deployment.
- Run live OpenAI-compatible AI UAT after the user supplies the promised provider key; keep the key backend-only.

## Product and planning backlog

- Approved-design enterprise parity still includes comments/guest comments, screen sharing, live presentation/spotlight/waiting-room/activity UI, version history/restore, libraries/collections, domain capture, SSO/SCIM and retention. Create requirements/tickets or record an explicit owner decision removing each item.
- Harness `PHASE-2.md` remains ready but unimplemented for CLI hard gates, loop guard and the security standard.
- `REQUIREMENTS.md` and `USER_STORIES.md` remain `TBD` drafts and need traceable content.
- Reconcile stale status only; do not reimplement completed work. E02/E03 are delivered, E05/EXC-030–033 are implemented, E04's Socket.IO/WebRTC-data-channel rows are superseded by custom `/collab/`, password auth supersedes the Google OAuth release wording, and populated project standards likely close BL-001.
- Preserve copied font license bytes or explicitly document any normalization; current whole-tree whitespace checks encounter upstream license formatting.

## Resume order

1. Fix and prove automatic reboot recovery.
2. Finish whole-branch and dependency review.
3. Take the fresh backup, install secrets, build frozen artifacts and migrate.
4. Deploy and run the full public canary.
5. Reconcile app/Homelab/public/agent-readable docs and close the release only when the evidence exists.

Do not perform image-only rollback after a v2 scene is promoted. Use the replacement-scene recovery procedure or a verified database restore.
